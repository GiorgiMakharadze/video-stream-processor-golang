package rooms

import (
	"fmt"
	"sync"
	"time"

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

func (rm *RoomManager) CreateRoomWithKey(streamKey string, publisher *websocket.Conn) (*Room, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.rooms[streamKey]; exists {
		return nil, fmt.Errorf("stream with key '%s' already exists", streamKey)
	}

	room := &Room{
		ID:        streamKey,
		CreatedAt: time.Now(),
		DataChan:  make(chan []byte, 1024),
		Publisher: publisher,
		CloseChan: make(chan struct{}),
	}
	rm.rooms[streamKey] = room
	return room, nil
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
	delete(rm.rooms, id)
}
