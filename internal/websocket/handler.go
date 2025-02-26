package websocket

import (
	"log"
	"net/http"
	"os"

	"github.com/GiorgiMakharadze/video-stream-processor-golang.git/internal/processor"
	config "github.com/GiorgiMakharadze/video-stream-processor-golang.git/pkg"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func WsHandler(w http.ResponseWriter, r *http.Request, cfg *config.Config, vp *processor.VideoProcessor) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	tmpFile, err := os.CreateTemp("", "video_input_*.tmp")
	if err != nil {
		log.Println("Error creating temp file:", err)
		return
	}

	log.Println("Receiving video stream...")
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Println("Stream closed or error reading message:", err)
			break
		}
		if messageType == websocket.BinaryMessage {
			if _, err = tmpFile.Write(data); err != nil {
				log.Println("Error writing chunk to file:", err)
				return
			}
		}
	}
	tmpFile.Close()

	vp.EnqueueTask(tmpFile.Name(), cfg)
}
