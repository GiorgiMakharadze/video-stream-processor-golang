package websocket

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	StreamKey string `json:"streamKey,omitempty"`
	HLSURL    string `json:"hlsUrl,omitempty"`
	Message   string `json:"message,omitempty"`
}

func HandleStreamsList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	type StreamInfo struct {
		StreamKey string    `json:"streamKey"`
		CreatedAt time.Time `json:"createdAt"`
	}

	roomsList := rooms.Manager.ListRooms()

	var streams []StreamInfo
	for _, room := range roomsList {
		streams = append(streams, StreamInfo{
			StreamKey: room.ID,
			CreatedAt: room.CreatedAt,
		})
	}

	response := map[string]interface{}{
		"count":   len(streams),
		"streams": streams,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding streams list: %v", err)
		http.Error(w, "failed to fetch streams list", http.StatusInternalServerError)
	}
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
	streamKey := r.URL.Query().Get("streamKey")
	if streamKey == "" {
		http.Error(w, "streamKey query parameter required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		http.Error(w, "failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	log.Printf("Publisher connected with streamKey: %s", streamKey)

	room, err := rooms.Manager.CreateRoomWithKey(streamKey, conn)
	if err != nil {
		log.Printf("Failed to create room: %v", err)
		http.Error(w, err.Error(), http.StatusConflict)
		conn.Close()
		return
	}
	hlsDir := fmt.Sprintf("/tmp/hls/live/%s", streamKey)
	if err := os.MkdirAll(hlsDir, 0755); err != nil {
		log.Println("Failed to pre-create HLS directory:", err)
		http.Error(w, "failed to prepare HLS directory", http.StatusInternalServerError)
		return
	}

	resp := Response{
		StreamKey: streamKey,
		Message:   "Room created",
	}
	if err := conn.WriteJSON(resp); err != nil {
		log.Println("Error sending streamKey to publisher:", err)
		conn.Close()
		rooms.Manager.DeleteRoom(streamKey)
		return
	}

	pr, pw := io.Pipe()
	rtmpURL := fmt.Sprintf("%s/%s", cfg.RTMPBaseURL, streamKey)

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
		rooms.Manager.DeleteRoom(streamKey)
		return
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Error creating FFmpeg stdout pipe:", err)
		conn.Close()
		rooms.Manager.DeleteRoom(streamKey)
		return
	}

	go streamLogger(stderrPipe, "STDERR")
	go streamLogger(stdoutPipe, "STDOUT")

	if err := cmd.Start(); err != nil {
		log.Println("Error starting FFmpeg process:", err)
		conn.Close()
		rooms.Manager.DeleteRoom(streamKey)
		return
	}
	log.Printf("FFmpeg started for streamKey %s, streaming to %s", streamKey, rtmpURL)

	go func() {
		defer pw.Close()
		for data := range room.DataChan {
			if _, err := pw.Write(data); err != nil {
				log.Println("Error writing data to FFmpeg pipe:", err)
				break
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-room.CloseChan:
				return
			case <-ticker.C:
			}
		}
	}()

	startTime := time.Now()
	chunkCount := 0

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Publisher WebSocket read error: %v (streamKey: %s)", err, streamKey)
			break
		}
		if messageType == websocket.BinaryMessage {
			select {
			case room.DataChan <- data:
				chunkCount++
			default:
				log.Printf("Stream %s data channel full - dropping packet", streamKey)
			}
		}
	}

	close(room.DataChan)
	conn.Close()

	log.Printf("Waiting for FFmpeg process to exit for streamKey %s...", streamKey)
	if err := cmd.Wait(); err != nil {
		log.Printf("FFmpeg process ended with error (streamKey %s): %v", streamKey, err)
	} else {
		log.Printf("FFmpeg process cleanly exited for streamKey %s", streamKey)
	}

	rooms.Manager.DeleteRoom(streamKey)
	log.Printf("Stream %s closed (duration: %s, chunks received: %d)",
		streamKey, time.Since(startTime), chunkCount)
}

func handleViewer(w http.ResponseWriter, r *http.Request, cfg *pkg.Config) {
	streamKey := r.URL.Query().Get("streamKey")
	if streamKey == "" {
		http.Error(w, "streamKey query parameter required", http.StatusBadRequest)
		return
	}

	_, exists := rooms.Manager.GetRoom(streamKey)
	if !exists {
		http.Error(w, "stream not found", http.StatusNotFound)
		return
	}

	hlsURL := fmt.Sprintf("%s/%s/index.m3u8", cfg.HLSBaseURL, streamKey)
	resp := Response{
		HLSURL:  hlsURL,
		Message: "Stream found",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error responding to viewer request (streamKey: %s): %v", streamKey, err)
	}
}

func streamLogger(pipe io.ReadCloser, label string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		log.Printf("[FFmpeg %s] %s", label, scanner.Text())
	}
}
