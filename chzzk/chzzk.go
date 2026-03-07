package chzzk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type ChzzkApi struct{}

func (c *ChzzkApi) GetChatChannelId(channelId string) (string, error) {
	url := fmt.Sprintf("https://api.chzzk.naver.com/polling/v2/channels/%s/live-status", channelId)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	content, ok := data["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid content")
	}

	id, ok := content["chatChannelId"].(string)
	if !ok {
		return "", fmt.Errorf("chatChannelId not found")
	}
	return id, nil
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
		return "", "", fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", err
	}

	content, ok := data["content"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("invalid content")
	}

	accessToken, ok1 := content["accessToken"].(string)
	extraToken, ok2 := content["extraToken"].(string)
	if !ok1 || !ok2 {
		return "", "", fmt.Errorf("tokens not found")
	}
	return accessToken, extraToken, nil
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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

	wsURL := url.URL{Scheme: "wss", Host: "kr-ss1.chat.naver.com", Path: "/chat"}
	externalConn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		log.Println("external WS connect error:", err)
		return
	}
	defer externalConn.Close()

	// 연결 메시지
	connectMsg := map[string]interface{}{
		"ver":   "3",
		"cmd":   100,
		"svcid": "game",
		"cid":   chatChannelId,
		"bdy": map[string]interface{}{
			"uid":      "go-client",
			"devType":  2001,
			"accTkn":   accessToken,
			"auth":     "SEND",
			"libVer":   "4.10.1",
			"osVer":    "Go",
			"devName":  "Golang Client",
			"locale":   "ko-KR",
			"timezone": "Asia/Seoul",
			"uuid":     "local-uuid",
		},
		"tid": 1,
	}
	externalConn.WriteJSON(connectMsg)

	// Ping 유지
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			pong := map[string]interface{}{"cmd": 10000, "ver": 2}
			externalConn.WriteJSON(pong)
		}
	}()

	// 외부 WS -> 클라이언트 포워딩
	go func() {
		for {
			_, msg, err := externalConn.ReadMessage()
			if err != nil {
				log.Println("external read error:", err)
				return
			}
			clientConn.WriteMessage(websocket.TextMessage, msg)
		}
	}()

	// 클라이언트 -> 외부 WS 포워딩
	for {
		_, msg, err := clientConn.ReadMessage()
		if err != nil {
			log.Println("client read error:", err)
			return
		}
		// 필요하면 JSON 파싱 후 일부 수정 가능
		externalConn.WriteMessage(websocket.TextMessage, bytes.TrimSpace(msg))
	}
}
