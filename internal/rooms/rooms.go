package rooms

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/GiorgiMakharadze/video-stream-processor-golang.git/pkg"
	"github.com/gorilla/websocket"
)

type Room struct {
	ID          string
	CreatedAt   time.Time
	DataChan    chan []byte
	Publisher   *websocket.Conn
	CloseChan   chan struct{}
	ErrorChan   chan error
	StoppedChan chan struct{}
	Cmd         *exec.Cmd
}

type RoomManager struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

var Manager = &RoomManager{
	rooms: make(map[string]*Room),
}

func (rm *RoomManager) CreateRoomWithKey(streamKey string, publisher *websocket.Conn) (*Room, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.rooms[streamKey]; exists {
		return nil, fmt.Errorf("stream with key '%s' already exists", streamKey)
	}

	room := &Room{
		ID:          streamKey,
		CreatedAt:   time.Now(),
		DataChan:    make(chan []byte, 100),
		Publisher:   publisher,
		CloseChan:   make(chan struct{}),
		ErrorChan:   make(chan error, 1),
		StoppedChan: make(chan struct{}),
	}
	rm.rooms[streamKey] = room
	return room, nil
}

func (r *Room) StartFFmpeg(cfg *pkg.Config) error {
	rtmpURL := fmt.Sprintf("%s/%s", cfg.RTMPBaseURL, r.ID)
	log.Printf("Starting FFmpeg for streamKey %s -> %s", r.ID, rtmpURL)

	pr, pw := io.Pipe()
	r.Cmd = exec.Command("ffmpeg",
		"-loglevel", "debug",
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-g", "30",
		"-c:a", "aac",
		"-b:a", "128k",
		"-flush_packets", "1",
		"-fflags", "nobuffer",
		"-f", "flv",
		rtmpURL,
	)
	r.Cmd.Stdin = pr

	stderrPipe, err := r.Cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get FFmpeg stderr pipe: %w", err)
	}
	stdoutPipe, err := r.Cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get FFmpeg stdout pipe: %w", err)
	}

	go streamLogger(stderrPipe, "STDERR")
	go streamLogger(stdoutPipe, "STDOUT")

	go func() {
		for data := range r.DataChan {
			_, err := pw.Write(data)
			if err != nil {
				r.ErrorChan <- err
				break
			}
		}
		pw.Close()
	}()

	return r.Cmd.Start()
}

func (r *Room) Monitor() {
	defer close(r.StoppedChan)

	go func() {
		err := r.Cmd.Wait()
		r.ErrorChan <- err
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.CloseChan:
			log.Printf("Stopping stream %s", r.ID)
			r.Cmd.Process.Kill()
			return
		case err := <-r.ErrorChan:
			log.Printf("FFmpeg process error for stream %s: %v", r.ID, err)
			return
		case <-ticker.C:
		}
	}
}

func (rm *RoomManager) ListRooms() []Room {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var list []Room
	for _, room := range rm.rooms {
		list = append(list, *room)
	}
	return list
}

func (rm *RoomManager) GetRoom(id string) (*Room, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	room, ok := rm.rooms[id]
	return room, ok
}

func (rm *RoomManager) DeleteRoom(id string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	room, exists := rm.rooms[id]
	if exists {
		close(room.CloseChan)
		<-room.StoppedChan
		delete(rm.rooms, id)
		cleanupHLSFiles(id)
	}
}

func cleanupHLSFiles(streamKey string) {
	hlsPath := fmt.Sprintf("/tmp/hls/live/%s*", streamKey)
	exec.Command("sh", "-c", fmt.Sprintf("rm -f %s", hlsPath)).Run()
}

func streamLogger(pipe io.ReadCloser, label string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		log.Printf("[FFmpeg %s] %s", label, scanner.Text())
	}
}
