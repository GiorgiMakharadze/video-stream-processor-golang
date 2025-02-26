package main

import (
	"log"
	"net/http"

	"github.com/GiorgiMakharadze/video-stream-processor-golang.git/internal/processor"
	"github.com/GiorgiMakharadze/video-stream-processor-golang.git/internal/websocket"
	config "github.com/GiorgiMakharadze/video-stream-processor-golang.git/pkg"
)

func main() {
	cfg := config.LoadConfig()

	videoProcessor := processor.NewVideoProcessor(5)
	videoProcessor.Start()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.WsHandler(w, r, cfg, videoProcessor)
	})

	port := ":" + cfg.WebSocketPort
	log.Println("Go WebSocket server is running on:", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server error:", err)
	}
}
