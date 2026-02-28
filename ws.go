package main

import (
	"net/http"
	"time"

	"github.com/as7ar/golive/logger"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Err("upgrade error:", err)
		return
	}
	defer conn.Close()

	logger.Info("client connected")

	message := "client connected"

	err = conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		logger.Err("write error:", err)
		return
	}

	time.Sleep(10 * time.Second)
}
