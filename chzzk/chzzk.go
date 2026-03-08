package chzzk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type liveStatusResponse struct {
	Content struct {
		ChatChannelId string `json:"chatChannelId"`
	} `json:"content"`
}

type accessTokenResponse struct {
	Content struct {
		AccessToken string `json:"accessToken"`
		ExtraToken  string `json:"extraToken"`
	} `json:"content"`
}

type ChzzkApi struct{}

type User struct {
	NICKNAME string `json:"nickname"`
	//NicknameColor string `json:"nicknameColor"`
	UserID string `json:"user_id"`
	//OS            string `json:"os"`
}

type Message struct {
	User      User            `json:"user"`
	Msg       string          `json:"msg"`
	MsgType   int             `json:"msgType"`
	MsgStatus int             `json:"msgStatus"`
	MsgTime   int64           `json:"msgTime"`
	Donation  *DonationExtras `json:"donation,omitempty"`
}

type DonationExtras struct {
	PayAmount         int    `json:"payAmount"`
	DonationType      string `json:"donationType"`
	MissionText       string `json:"missionText,omitempty"`
	MissionDonationId string `json:"missionDonationId,omitempty"`
}

type MsgType int

const (
	Chat         MsgType = 1
	Donation     MsgType = 10
	Subscription MsgType = 11
	System       MsgType = 30
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func calculateServerId(channelId string) int {
	serverId := 0
	for _, char := range channelId {
		if digit, err := strconv.Atoi(string(char)); err == nil {
			serverId += digit
		}
	}
	return (serverId % 9) + 1
}

func (c *ChzzkApi) GetChatChannelId(channelId string) (string, error) {
	url := fmt.Sprintf("https://api.chzzk.naver.com/service/v3/channels/%s/live-detail", channelId)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("API_CHAT_CHANNEL_ID_ERROR")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result liveStatusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Content.ChatChannelId == "" {
		return "", errors.New("chatChannelId not found")
	}

	return result.Content.ChatChannelId, nil
}

func (c *ChzzkApi) GetAccessToken(chatChannelId string) (string, string, error) {
	url := fmt.Sprintf("https://comm-api.game.naver.com/nng_main/v1/chats/access-token?channelId=%s&chatType=STREAMING", chatChannelId)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", errors.New("API_ACCESS_TOKEN_ERROR")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var result accessTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	return result.Content.AccessToken, result.Content.ExtraToken, nil
}

func ChzzkHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id missing", http.StatusBadRequest)
		return
	}

	api := &ChzzkApi{}
	chatChannelId, err := api.GetChatChannelId(id)
	if err != nil {
		http.Error(w, "chatChannelId error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	accessToken, _, err := api.GetAccessToken(chatChannelId)
	if err != nil {
		http.Error(w, "accessToken error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer clientConn.Close()

	serverId := calculateServerId(id)
	wsURL := url.URL{Scheme: "wss", Host: fmt.Sprintf("kr-ss%d.chat.naver.com", serverId), Path: "/chat"}
	externalConn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		log.Println("external WS connect error:", err)
		return
	}
	defer externalConn.Close()

	connectMsg := map[string]interface{}{
		"ver":   "3",
		"cmd":   100,
		"svcid": "game",
		"cid":   chatChannelId,
		"bdy": map[string]interface{}{
			"uid":      "",
			"devType":  2001,
			"accTkn":   accessToken,
			"auth":     "READ",
			"libVer":   "4.9.3",
			"osVer":    "Go",
			"devName":  "Golang Client",
			"locale":   "ko-KR",
			"timezone": "Asia/Seoul",
			"uuid":     "local-uuid",
		},
		"tid": 1,
	}
	externalConn.WriteJSON(connectMsg)

	lastPingTime := time.Now()
	lastMessageTime := time.Now()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			if now.Sub(lastPingTime) >= 60*time.Second || now.Sub(lastMessageTime) >= 20*time.Second {
				pingMsg := map[string]interface{}{"cmd": 0, "ver": "3"}
				if err := externalConn.WriteJSON(pingMsg); err != nil {
					return
				}
				lastPingTime = now
			}
		}
	}()

	go func() {
		for {
			_, msg, err := externalConn.ReadMessage()
			if err != nil {
				return
			}

			lastMessageTime = time.Now()

			var packet map[string]interface{}
			if err := json.Unmarshal(msg, &packet); err != nil {
				continue
			}

			cmd, _ := packet["cmd"].(float64)

			if cmd == 0 {
				pong := map[string]interface{}{
					"cmd": 10000,
					"ver": 2,
				}
				externalConn.WriteJSON(pong)
				continue
			}

			if cmd == 93101 {
				bdy, ok := packet["bdy"].([]interface{})
				if !ok {
					continue
				}

				for _, v := range bdy {

					item, ok := v.(map[string]interface{})
					if !ok {
						continue
					}

					msgStr, _ := item["msg"].(string)
					msgType, _ := item["msgTypeCode"].(float64)
					ctime, _ := item["ctime"].(float64)

					profileStr, _ := item["profile"].(string)

					var profile struct {
						UserIdHash string `json:"userIdHash"`
						NameColor  string `json:"nicknameColor"`
						Nickname   string `json:"nickname"`
					}

					json.Unmarshal([]byte(profileStr), &profile)

					extrasStr, _ := item["extras"].(string)

					var donation *DonationExtras

					if int(msgType) == 10 {
						var extra DonationExtras
						if err := json.Unmarshal([]byte(extrasStr), &extra); err == nil {
							donation = &extra
						}
					}

					message := Message{
						User: User{
							NICKNAME: profile.Nickname,
							UserID:   profile.UserIdHash,
							//NicknameColor: profile.NameColor,
						},
						Msg:      msgStr,
						MsgType:  int(msgType),
						MsgTime:  int64(ctime),
						Donation: donation,
					}

					clientConn.WriteJSON(message)
				}
			}
		}
	}()

	for {
		_, msg, err := clientConn.ReadMessage()
		if err != nil {
			return
		}
		externalConn.WriteMessage(websocket.TextMessage, bytes.TrimSpace(msg))
	}
}
