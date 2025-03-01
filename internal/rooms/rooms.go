package rooms

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Room struct {
	ID        string
	CreatedAt time.Time
	DataChan  chan []byte
	Publisher *websocket.Conn
	CloseChan chan struct{}
}

type RoomManager struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

var Manager = &RoomManager{
	rooms: make(map[string]*Room),
}

func (rm *RoomManager) CreateRoom(publisher *websocket.Conn) *Room {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	roomID := uuid.New().String()
	room := &Room{
		ID:        roomID,
		CreatedAt: time.Now(),
		DataChan:  make(chan []byte, 1024),
		Publisher: publisher,
		CloseChan: make(chan struct{}),
	}
	rm.rooms[roomID] = room
	return room
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
	delete(rm.rooms, id)
}
