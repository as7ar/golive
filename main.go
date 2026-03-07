package main

import (
	"flag"
	"net/http"

	"github.com/as7ar/golive/logger"
	"github.com/as7ar/golive/soop"
	"github.com/as7ar/golive/weflab"
)

func main() {
	port := *flag.String("port", "8080", "")
	flag.Parse()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/api", wsHandler)

	http.HandleFunc("/api/weflab", weflab.WeflabHandler)
	http.HandleFunc("/api/soop", soop.SoopHandler)

	logger.Info("server started on :" + port)
	http.ListenAndServe(":"+port, nil)
}
