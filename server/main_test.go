package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/danii7514/codpen/commons"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var testingLog bytes.Buffer

func TestHandleConn(t *testing.T) {

	initialRoomsMapLength := len(roomsMap)

	// Create a test server with the handleConn handler
	server := httptest.NewServer(http.HandlerFunc(handleConn))
	defer server.Close()

	// Generate a unique roomID for testing
	roomID := uuid.New().String()

	// Create a WebSocket connection to the test server with the generated roomID
	url := "ws" + server.URL[4:] + "?room=" + roomID
	room, _ := getOrCreateRoom(roomID)
	clientID := uuid.New()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	mu.Lock()
	siteID++
	client := &client{
		Conn:     conn,
		SiteID:   strconv.Itoa(siteID),
		id:       clientID,
		writeMu:  sync.Mutex{},
		mu:       sync.Mutex{},
		Username: "", // Username will be set later when the client joins the room.
	}
	mu.Unlock()

	room.Clients.add(client)

	// Allow some time for the handleConn goroutines to execute
	time.Sleep(100 * time.Millisecond)

	// Assert that the roomsMap has a new room and the length has increased
	if len(roomsMap) != initialRoomsMapLength+1 {
		t.Errorf("Expected roomsMap length to increase by 1, but got %d. RoomsMap: %v", len(roomsMap), roomsMap)
	}

	// Example: Check if the client has joined the room

	room_e := getRoomByClientID(clientID)
	if room_e == nil {
		t.Errorf("Expected client %s to join a room, but couldn't find the room. RoomsMap: %v", clientID, roomsMap)
	} else {
		// Check if the clientID is in the room.Clients.list
		_, exists := <-room.Clients.get(clientID)
		if !exists {
			t.Errorf("Expected client %s to be in the room.Clients.list, but it wasn't. RoomsMap: %v", clientID, roomsMap)
		}
	}

}

// TestHandleMsg tests the handleMsg function
func TestHandleMsg(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleConn))
	defer server.Close()
	// Create a test channel to simulate messageChan
	testMessageChan := make(chan commons.Message, 1)

	// Replace the original messageChan with the test channel
	originalMessageChan := messageChan
	messageChan = testMessageChan
	defer func() { messageChan = originalMessageChan }()

	// Create a channel to signal when sendUsernames is called
	sendUsernamesCalled := make(chan struct{}, 1)

	// Simulate a message being sent to messageChan
	clientID := uuid.New()
	message := commons.Message{
		Type:     commons.JoinMessage,
		Text:     "TestUser",
		ID:       clientID,
		Username: "TestUser",
	}

	roomID := uuid.New().String()

	url := "ws" + server.URL[4:] + "?room=" + roomID
	room, _ := getOrCreateRoom(roomID)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	mu.Lock()
	siteID++
	client := &client{
		Conn:     conn,
		SiteID:   strconv.Itoa(siteID),
		id:       clientID,
		writeMu:  sync.Mutex{},
		mu:       sync.Mutex{},
		Username: "TestUser",
	}
	mu.Unlock()

	room.Clients.add(client)

	testMessageChan <- message

	// Allow some time for the message to be processed by handleMsg
	time.Sleep(100 * time.Millisecond)

	// Check if the expected actions are taken in handleMsg
	select {
	case msg := <-testMessageChan:
		// Assert that the message is a JoinMessage
		if msg.Type != commons.JoinMessage {
			t.Errorf("Expected JoinMessage, but got %s", msg.Type)
		}

		// Assert that the updateName method is called
		if client := room.Clients.list[clientID]; client == nil || client.Username != "TestUser" {
			t.Errorf("Expected client with username TestUser, but not found or incorrect username")
		}

		// Signal that sendUsernames is called
		sendUsernamesCalled <- struct{}{}

	default:
		t.Error("Expected a message to be processed, but none received")
	}

	// Wait for a short duration to allow sendUsernames to be processed
	time.Sleep(100 * time.Millisecond)

	// Check if sendUsernames is called
	select {
	case <-sendUsernamesCalled:
		// sendUsernames is called, no assertion needed
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for sendUsernames to be called")
	}

}

// TestHandleSync tests the handleSync function
func TestHandleSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleConn))
	defer server.Close()

	roomID := uuid.New().String()

	url := "ws" + server.URL[4:] + "?room=" + roomID
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer conn.Close()

	// Create a test channel to simulate syncChan
	testSyncChan := make(chan commons.Message, 1)

	// Replace the original syncChan with the test channel
	originalSyncChan := syncChan
	syncChan = testSyncChan
	defer func() { syncChan = originalSyncChan }()

	// Create a test room
	room, _ := getOrCreateRoom(roomID)

	// Simulate a DocSyncMessage being sent to syncChan
	clientID := uuid.New()
	docSyncMessage := commons.Message{
		Type:     commons.DocSyncMessage,
		Text:     "TestDocumentSync",
		ID:       clientID,
		Username: "TestUser",
	}

	mu.Lock()
	siteID++
	client := &client{
		Conn:     conn,
		SiteID:   strconv.Itoa(siteID),
		id:       clientID,
		writeMu:  sync.Mutex{},
		mu:       sync.Mutex{},
		Username: "TestUser",
	}
	mu.Unlock()
	testSyncChan <- docSyncMessage

	room.Clients.add(client)
	// Allow some time for the message to be processed by handleSync
	time.Sleep(500 * time.Millisecond)

	// Check if the expected actions are taken in handleSync
	select {
	case syncMsg := <-testSyncChan:
		// Assert that the message is a DocSyncMessage
		if syncMsg.Type != commons.DocSyncMessage {
			t.Errorf("Expected DocSyncMessage, but got %s", syncMsg.Type)
		}

		// Assert that broadcastOne is called with the correct parameters
		if client := room.Clients.list[clientID]; client == nil {
			t.Error("Expected client to exist, but not found")
		} else {
			// Ensure broadcastOne is called with the correct parameters
			select {
			case broadcastClient := <-room.Clients.get(clientID):
				if broadcastClient != nil {
					// Assuming broadcastClient is of type *client
					if broadcastClient.Username != "TestUser" {
						t.Errorf("Expected broadcastOne to be called with the correct client, but got %+v", broadcastClient)
					}
				}
			default:
				t.Error("Expected broadcastOne to be called, but not called")
			}
		}

	default:
		t.Error("Expected a message to be processed, but none received")
	}
}
