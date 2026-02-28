package main

import (
	"net/http"

	"github.com/as7ar/golive/logger"
	"github.com/as7ar/golive/weflab"
)

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/api", wsHandler)

	http.HandleFunc("/api/weflab", weflab.WeflabHandler)

	logger.Info("server started on :8080")
	http.ListenAndServe(":8080", nil)
}
