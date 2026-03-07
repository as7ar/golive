package weflab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type XHRListener struct {
	Key    string
	Client *http.Client
}

func NewXHRListener(key string) *XHRListener {
	return &XHRListener{
		Key: key,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (x *XHRListener) Start() {
	fmt.Println("Start to Listening XHR...")

	ticker := time.NewTicker(2 * time.Second)

	for range ticker.C {

		req, err := http.NewRequest(
			"GET",
			"https://weflab.com/page/"+x.Key,
			nil,
		)
		if err != nil {
			fmt.Println("req error:", err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Referer", "https://weflab.com/")

		resp, err := x.Client.Do(req)
		if err != nil {
			fmt.Println("xhr error:", err)
			continue
		}

		//body, _ := io.ReadAll(resp.Body)
		//fmt.Println(string(body))

		var raw map[string]any
		json.NewDecoder(resp.Body).Decode(&raw)
		resp.Body.Close()

		//fmt.Println("Response:", raw)
	}
}
