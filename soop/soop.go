package soop

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/as7ar/golive/logger"
	"github.com/gorilla/websocket"
)

type SoopLiveInfo struct {
	CHDOMAIN string
	CHATNO   string
	BJID     string
	CHPT     string
}

type Alert struct {
	Platform string
	Name     string
	Value    int
	Message  string
	Type     string
}

type SoopPacket struct {
	Command      string
	DataList     []string
	ReceivedTime time.Time
}

func NewSoopPacket(args []string) *SoopPacket {
	cmd := args[0]
	return &SoopPacket{
		Command:      cmd[:4],
		DataList:     args[1:],
		ReceivedTime: time.Now(),
	}
}

type SoopClient struct {
	conn *websocket.Conn

	liveInfo *SoopLiveInfo

	packetMap map[string]*SoopPacket
	lock      sync.Mutex

	poong   bool
	alive   bool
	handler func(Alert)
}

const (
	F   = "\u000c"
	ESC = "\u001b\t"

	CMD_PING    = "0000"
	CMD_CONNECT = "0001"
	CMD_JOIN    = "0002"
	CMD_CHAT    = "0005"
	CMD_DONE    = "0018"
)

func makePacket(command, data string) string {
	return ESC + command + fmt.Sprintf("%06d00", len(data)) + data
}

var CONNECT_PACKET = makePacket(CMD_CONNECT, fmt.Sprintf("%s16%s", strings.Repeat(F, 3), F))
var CONNECT_RES_PACKET = makePacket(CMD_CONNECT, fmt.Sprintf("%s16|0%s", strings.Repeat(F, 2), F))
var PING_PACKET = makePacket(CMD_PING, F)

func GetPlayerLive(bjid string) (*SoopLiveInfo, error) {

	form := url.Values{}
	form.Set("bid", bjid)
	form.Set("type", "live")
	form.Set("player_type", "html5")

	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("https://live.sooplive.co.kr/afreeca/player_live_api.php?bjid=%s", bjid),
		bytes.NewBufferString(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var raw map[string]any
	json.NewDecoder(resp.Body).Decode(&raw)

	ch := raw["CHANNEL"].(map[string]any)

	chpt, _ := strconv.Atoi(ch["CHPT"].(string))

	return &SoopLiveInfo{
		CHDOMAIN: ch["CHDOMAIN"].(string),
		CHATNO:   ch["CHATNO"].(string),
		BJID:     ch["BJID"].(string),
		CHPT:     strconv.Itoa(chpt + 1),
	}, nil
}

func ConnectSoop(bjid string, poong bool, chat bool, handler func(Alert)) (*SoopClient, error) {
	live, err := GetPlayerLive(bjid)
	if err != nil {
		return nil, err
	}

	server := fmt.Sprintf(
		"wss://%s:%s/Websocket/%s",
		live.CHDOMAIN,
		live.CHPT,
		live.BJID,
	)

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	conn, _, err := dialer.Dial(server, nil)
	if err != nil {
		return nil, err
	}

	c := &SoopClient{
		conn:      conn,
		liveInfo:  live,
		packetMap: map[string]*SoopPacket{},
		poong:     poong,
		alive:     true,
		handler:   handler,
	}

	go c.pingLoop()
	go c.readLoop(chat)

	conn.WriteMessage(websocket.BinaryMessage, []byte(CONNECT_PACKET))

	return c, nil
}

func (c *SoopClient) pingLoop() {
	for c.alive {
		time.Sleep(60 * time.Second)

		c.conn.WriteMessage(websocket.BinaryMessage, []byte(PING_PACKET))
		c.lock.Lock()

		for k, v := range c.packetMap {
			if time.Since(v.ReceivedTime) > time.Minute {
				delete(c.packetMap, k)
			}
		}

		c.lock.Unlock()
	}
}

func (c *SoopClient) readLoop(chat bool) {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		message := string(msg)

		if message == CONNECT_RES_PACKET {
			join := makePacket(
				CMD_JOIN,
				fmt.Sprintf("%s%s%s", F, c.liveInfo.CHATNO, strings.Repeat(F, 5)),
			)

			c.conn.WriteMessage(websocket.BinaryMessage, []byte(join))
			continue
		}

		packet := NewSoopPacket(
			strings.Split(strings.ReplaceAll(message, ESC, ""), F),
		)

		cmd := packet.Command
		data := packet.DataList

		if len(data) == 0 {
			continue
		}

		switch cmd {
		case CMD_DONE:
			nick := data[2]
			c.lock.Lock()
			c.packetMap[nick] = packet
			c.lock.Unlock()

			go func(nick string, data []string) {
				time.Sleep(time.Second)

				c.lock.Lock()
				done := c.packetMap[nick]
				delete(c.packetMap, nick)
				c.lock.Unlock()

				if done == nil {
					return
				}

				amount, _ := strconv.Atoi(data[3])
				if !c.poong {
					amount *= 100
				}

				c.emit(nick, amount, "")

			}(nick, data)

		case CMD_CHAT:
			nick := data[5]
			msg := data[0]

			c.lock.Lock()
			done := c.packetMap[nick]
			delete(c.packetMap, nick)
			c.lock.Unlock()

			if done == nil {
				if chat {
					c.emitChat(nick, msg)
				}
				continue
			}

			amount, _ := strconv.Atoi(done.DataList[3])
			if !c.poong {
				amount *= 100
			}
			c.emit(done.DataList[2], amount, msg)
		}
	}
}

func (c *SoopClient) emit(nickname string, pay int, msg string) {
	if c.handler == nil {
		return
	}

	c.handler(Alert{
		Platform: "Soop",
		Name:     nickname,
		Value:    pay,
		Message:  msg,
		Type:     "Donation",
	})
}

func (c *SoopClient) emitChat(nickname string, msg string) {
	if c.handler == nil {
		return
	}

	c.handler(Alert{
		Platform: "Soop",
		Name:     nickname,
		Message:  msg,
		Type:     "Chat",
	})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func SoopHandler(w http.ResponseWriter, r *http.Request) {
	bjid := r.URL.Query().Get("bjid")
	enableChat, err := strconv.ParseBool(r.URL.Query().Get("chat"))
	if err != nil {
		logger.Err("invalid value:chat: ", r.URL.Query().Get("chat"), ". it should be 'true' or 'false'")
		return
	}
	if bjid == "" {
		http.Error(w, "missing bjid", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client, err := ConnectSoop(bjid, false, enableChat, func(d Alert) {

		data, _ := json.Marshal(d)

		ws.WriteMessage(websocket.TextMessage, data)
	})

	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(`{"error":"connect failed"}`))
		ws.Close()
		return
	}

	defer func() {
		client.alive = false
		ws.Close()
	}()

	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			return
		}
	}
}
