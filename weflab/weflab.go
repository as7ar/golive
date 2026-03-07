package weflab

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Donation struct {
	Uid      string `json:"uid"`
	Uname    string `json:"uname"`
	Message  string `json:"message"`
	Value    int    `json:"value"`
	Platform string `json:"platform"`
	Type     string `json:"type"`
}

type WeflabClient struct {
	Key    string
	Idx    string
	PageID string
	Conn   *websocket.Conn
}

func NewWeflabClient(key string) (*WeflabClient, error) {
	w := &WeflabClient{Key: key}
	if err := w.loadPage(); err != nil {
		return nil, err
	}
	if err := w.connect(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *WeflabClient) loadPage() error {
	resp, err := http.Get("https://weflab.com/page/" + w.Key)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	var scriptText string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		txt := s.Text()
		if strings.Contains(txt, "loginData = {") {
			scriptText = txt
		}
	})

	re := regexp.MustCompile(`loginData\s*=\s*(\{.*?});`)
	match := re.FindStringSubmatch(scriptText)
	if len(match) < 2 {
		return err
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(match[1]), &parsed)

	w.Idx = parsed["idx"].(string)
	w.PageID = parsed["pageid"].(string)

	return nil
}

func (w *WeflabClient) connect() error {
	url := "wss://ssmain.weflab.com/socket.io/?idx=" +
		w.Idx + "&type=page&page=" +
		w.PageID + "&EIO=4&transport=websocket"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	w.Conn = conn

	conn.WriteMessage(websocket.TextMessage, []byte("40"))

	joinMsg := `42["msg",{"type":"join","page":"page","idx":"` +
		w.Idx + `","pageid":"` +
		w.PageID + `","preset":"0"}]`

	conn.WriteMessage(websocket.TextMessage, []byte(joinMsg))

	return nil
}

func (w *WeflabClient) ReadLoop(onDonation func(Donation)) {
	for {
		_, msg, err := w.Conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}

		text := string(msg)

		if text == "2" { // ping
			w.Conn.WriteMessage(websocket.TextMessage, []byte("3"))
			continue
		}

		if strings.HasPrefix(text, "42") {
			raw := text[2:]

			var arr []interface{}
			if err := json.Unmarshal([]byte(raw), &arr); err != nil {
				continue
			}

			event := arr[0].(string)
			if event != "msg" {
				continue
			}

			data := arr[1].(map[string]interface{})
			tp := data["type"].(string)

			if tp != "donation" && tp != "test_donation" {
				continue
			}

			d := data["data"].(map[string]interface{})
			//logger.Debug(d)

			v, _ := strconv.Atoi(d["value"].(string))
			donation := Donation{
				Uid:      d["uid"].(string),
				Uname:    d["uname"].(string),
				Message:  d["msg"].(string),
				Value:    v,
				Platform: d["platform"].(string),
				Type:     d["type"].(string),
			}

			onDonation(donation)
		}
	}
}

func WeflabHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("client connected, key:", key)

	listener := NewXHRListener(key)
	go listener.Start()

	wef, err := NewWeflabClient(key)
	if err != nil {
		log.Println("weflab error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte("weflab connect failed"))
		return
	}

	go wef.ReadLoop(func(d Donation) {
		msg, _ := json.Marshal(d)
		conn.WriteMessage(websocket.TextMessage, msg)
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
