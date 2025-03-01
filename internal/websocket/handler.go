package websocket

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/GiorgiMakharadze/video-stream-processor-golang.git/internal/rooms"
	pkg "github.com/GiorgiMakharadze/video-stream-processor-golang.git/pkg"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  65536,
	WriteBufferSize: 65536,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Response struct {
	StreamID string `json:"streamId,omitempty"`
	HLSURL   string `json:"hlsUrl,omitempty"`
	Message  string `json:"message,omitempty"`
}

func WsHandler(w http.ResponseWriter, r *http.Request, cfg *pkg.Config) {
	role := r.URL.Query().Get("role")
	switch role {
	case "publisher":
		handlePublisher(w, r, cfg)
	case "viewer":
		handleViewer(w, r, cfg)
	default:
		http.Error(w, "role query parameter required (publisher/viewer)", http.StatusBadRequest)
	}
}

func handlePublisher(w http.ResponseWriter, r *http.Request, cfg *pkg.Config) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		http.Error(w, "failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	log.Println("Publisher connected")

	room := rooms.Manager.CreateRoom(conn)
	log.Printf("Created room with ID: %s", room.ID)

	resp := Response{
		StreamID: room.ID,
		Message:  "Room created",
	}
	if err := conn.WriteJSON(resp); err != nil {
		log.Println("Error sending room ID to publisher:", err)
		conn.Close()
		rooms.Manager.DeleteRoom(room.ID)
		return
	}

	pr, pw := io.Pipe()
	rtmpURL := cfg.RTMPBaseURL + "/" + room.ID

	cmd := exec.Command("ffmpeg",
		"-loglevel", "debug",
		"-re",
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-f", "flv",
		rtmpURL,
	)

	cmd.Stdin = pr

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Println("Error creating FFmpeg stderr pipe:", err)
		conn.Close()
		rooms.Manager.DeleteRoom(room.ID)
		return
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Error creating FFmpeg stdout pipe:", err)
		conn.Close()
		rooms.Manager.DeleteRoom(room.ID)
		return
	}

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			log.Println("[FFmpeg STDERR]", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			log.Println("[FFmpeg STDOUT]", scanner.Text())
		}
	}()

	if err := cmd.Start(); err != nil {
		log.Println("Error starting FFmpeg process:", err)
		conn.Close()
		rooms.Manager.DeleteRoom(room.ID)
		return
	}
	log.Printf("FFmpeg started for room %s, streaming to %s", room.ID, rtmpURL)

	go func() {
		defer pw.Close()
		for data := range room.DataChan {
			if _, err := pw.Write(data); err != nil {
				log.Println("Error writing data to FFmpeg pipe:", err)
				break
			}
		}
	}()

	startTime := time.Now()
	chunkCount := 0

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Publisher WebSocket read error: %v (room: %s)", err, room.ID)
			break
		}
		if messageType == websocket.BinaryMessage {
			select {
			case room.DataChan <- data:
				chunkCount++
			default:
				log.Printf("Room %s data channel full - dropping packet", room.ID)
			}
		}
	}

	close(room.DataChan)
	conn.Close()

	log.Printf("Waiting for FFmpeg process to exit for room %s...", room.ID)
	if err := cmd.Wait(); err != nil {
		log.Printf("FFmpeg process ended with error (room %s): %v", room.ID, err)
	} else {
		log.Printf("FFmpeg process cleanly exited for room %s", room.ID)
	}

	rooms.Manager.DeleteRoom(room.ID)
	log.Printf("Room %s closed (duration: %s, chunks received: %d)",
		room.ID, time.Since(startTime), chunkCount)
}

func handleViewer(w http.ResponseWriter, r *http.Request, cfg *pkg.Config) {
	streamID := r.URL.Query().Get("streamId")
	if streamID == "" {
		http.Error(w, "missing streamId", http.StatusBadRequest)
		return
	}

	room, exists := rooms.Manager.GetRoom(streamID)
	if !exists {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	hlsURL := cfg.HLSBaseURL + "/" + room.ID + "/index.m3u8"
	resp := Response{
		HLSURL:  hlsURL,
		Message: "Room found",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error responding to viewer request (streamId: %s): %v", streamID, err)
	}
}
