package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/danii7514/codpen/commons"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Clients is used to store, reference, and update information about all connected clients.
type Clients struct {
	// list stores information about active clients.
	list map[uuid.UUID]*client

	// clientsMu protects against concurrent read/write access to the activeClients map.
	mu sync.RWMutex

	// deleteRequests indicates which clients to delete from the list of clients.
	deleteRequests chan deleteRequest

	// readRequests indicates which clients to retrieve from the list of clients.
	readRequests chan readRequest

	// addRequests is used to send clients to add to the list of clients.
	addRequests chan *client

	// nameUpdateRequests is used to update a client with their username.
	nameUpdateRequests chan nameUpdate
}

// NewClients returns a new instance of a Clients struct.
func NewClients() *Clients {
	return &Clients{
		list:               make(map[uuid.UUID]*client),
		mu:                 sync.RWMutex{},
		deleteRequests:     make(chan deleteRequest),
		readRequests:       make(chan readRequest, 10000),
		addRequests:        make(chan *client),
		nameUpdateRequests: make(chan nameUpdate),
	}
}

// a client holds the information of a connected client.
type client struct {
	Conn   *websocket.Conn
	SiteID string
	id     uuid.UUID

	// writeMu protects against concurrent writes to a WebSocket connection.
	writeMu sync.Mutex

	// mu protects against data races on a client's info
	mu sync.Mutex

	Username string
}

// Room represents a chat room with its connected clients.
type Room struct {
	ID      string
	Clients *Clients
}

// NewRoom creates a new room with a unique ID.
func NewRoom() *Room {
	return &Room{
		ID:      uuid.New().String(),
		Clients: NewClients(),
	}
}

var (
	// Monotonically increasing site ID, unique to each client.
	siteID = 0

	// Mutex for protecting site ID increment operations.
	mu sync.Mutex

	// Upgrader instance to upgrade all HTTP connections to a WebSocket.
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Channel for client messages.
	messageChan = make(chan commons.Message)

	// Channel for document sync messages.
	syncChan = make(chan commons.Message)

	// Map to store rooms by their unique IDs.
	roomsMap      = make(map[string]*Room)
	roomsMapMutex sync.Mutex
)

func main() {
	addr := flag.String("addr", ":8084", "Server's network address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleConn)

	// Handle incoming messages.
	go handleMsg()

	// Handle document syncing
	go handleSync()

	// Start the server.
	log.Printf("Starting server on %s", *addr)

	server := &http.Server{
		Addr:         *addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Error starting server, exiting.", err)
	}
}

// handleConn handles incoming HTTP connections, assigns clients to rooms, and reads messages from the connection.
func handleConn(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		color.Red("Error upgrading connection to websocket: %v\n", err)
		conn.Close()
		return
	}
	defer conn.Close()

	clientID := uuid.New()
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		color.Red("Room ID not provided.")
		return
	}

	room, exists := getOrCreateRoom(roomID)

	if !exists {
		go room.Clients.handle()
	}

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

	siteIDMsg := commons.Message{Type: commons.SiteIDMessage, Text: client.SiteID, ID: clientID}
	room.Clients.broadcastOne(siteIDMsg, clientID)

	docReq := commons.Message{Type: commons.DocReqMessage, ID: clientID}
	log.Printf("sending docReq with %v", docReq)
	room.Clients.broadcastOneExcept(docReq, clientID)

	room.Clients.sendUsernames()

	for {
		var msg commons.Message
		if err := client.read(&msg); err != nil {
			color.Red("Failed to read message. closing client connection with %s. Error: %s", client.Username, err)
			return
		}

		if msg.Type == commons.DocSyncMessage {
			syncChan <- msg
			continue
		}

		msg.ID = clientID
		messageChan <- msg
	}
}

func getOrCreateRoom(roomID string) (*Room, bool) {
	roomsMapMutex.Lock()
	defer roomsMapMutex.Unlock()

	if room, ok := roomsMap[roomID]; ok {
		return room, false
	}

	room := NewRoom()
	go room.Clients.handle()
	roomsMap[roomID] = room

	return room, true
}

func getRoomByClientID(clientID uuid.UUID) *Room {
	roomsMapMutex.Lock()
	defer roomsMapMutex.Unlock()

	for _, room := range roomsMap {
		if _, exists := room.Clients.list[clientID]; exists {
			return room
		}
	}

	return nil
}

// handleMsg listens to the messageChan channel and broadcasts messages to other clients in the same room.
func handleMsg() {
	for {
		msg := <-messageChan

		t := time.Now().Format(time.ANSIC)
		if msg.Type == commons.JoinMessage {
			room := getRoomByClientID(msg.ID)
			room.Clients.updateName(msg.ID, msg.Username)
			color.Green("%s >> %s %s (ID: %s) in room %s\n", t, msg.Username, msg.Text, msg.ID, room.ID)
			room.Clients.sendUsernames()
		} else if msg.Type == "operation" {
			color.Green("operation >> %+v from ID=%s\n", msg.Operation, msg.ID)
		} else {
			color.Green("%s >> unknown message type:  %v\n", t, msg)
			room := getRoomByClientID(msg.ID)
			room.Clients.sendUsernames()
			continue
		}

		room := getRoomByClientID(msg.ID)
		room.Clients.broadcastAllExcept(msg, msg.ID, room.ID)
	}
}

// handleSync reads from the syncChan and sends the message to the appropriate user(s) in the same room.
func handleSync() {
	for {
		syncMsg := <-syncChan
		switch syncMsg.Type {
		case commons.DocSyncMessage:
			room := getRoomByClientID(syncMsg.ID)
			room.Clients.broadcastOne(syncMsg, syncMsg.ID)
		case commons.UsersMessage:
			room := getRoomByClientID(syncMsg.ID)
			if room != nil {
				color.Blue("usernames in room %s: %s", room.ID, syncMsg.Text)
				room.Clients.broadcastAll(syncMsg, room.ID)
			}
		}
	}
}

// handle acts as a monitor for a Clients type. handle attempts to ensure concurrency safety
// for accessing the Clients struct.
func (c *Clients) handle() {
	for {
		select {
		case req := <-c.deleteRequests:
			c.close(req.id)
			req.done <- 1
			close(req.done)
		case req := <-c.readRequests:
			if req.readAll {
				for _, client := range c.list {
					req.resp <- client
				}
				close(req.resp)
			} else {
				req.resp <- c.list[req.id]
				close(req.resp)
			}
		case client := <-c.addRequests:
			c.mu.Lock()
			c.list[client.id] = client
			c.mu.Unlock()
		case n := <-c.nameUpdateRequests:
			c.list[n.id].mu.Lock()
			c.list[n.id].Username = n.newName
			c.list[n.id].mu.Unlock()
		}
	}
}

// A deleteRequest is used to delete clients from the list of clients.
type deleteRequest struct {
	// id is the ID of the client to be deleted.
	id uuid.UUID

	// done is used to signal that a delete request has been fulfilled.
	done chan int
}

// A readRequest is used to help callers retrieve information about clients.
type readRequest struct {
	// readAll indicates whether the caller wants to receive all clients.
	readAll bool

	// id is the id of the client to be retrieved from the list of clients. id is the
	// zero value of uuid.UUID if readAll is true.
	id uuid.UUID

	// resp is the channel from which requesters can read the response.
	resp chan *client
}

// getAll retrieves all active clients in the same room.
func (c *Clients) getAll() chan *client {
	c.mu.RLock()
	resp := make(chan *client, len(c.list))
	c.mu.RUnlock()
	c.readRequests <- readRequest{readAll: true, resp: resp}
	return resp
}

// get requests a client with the given id, and returns a channel containing the client. If
// the client doesn't exist, the channel will be empty
func (c *Clients) get(id uuid.UUID) chan *client {
	resp := make(chan *client)
	c.readRequests <- readRequest{readAll: false, id: id, resp: resp}
	return resp
}

// add adds a client to the list of clients.
func (c *Clients) add(client *client) {
	c.addRequests <- client
}

// A nameUpdate is used as a message to update the name of a client.
type nameUpdate struct {
	id      uuid.UUID
	newName string
}

// updateName updates the name field of a client with the given id.
func (c *Clients) updateName(id uuid.UUID, newName string) {
	c.nameUpdateRequests <- nameUpdate{id, newName}
}

// delete deletes a client from the list of active clients.
func (c *Clients) delete(id uuid.UUID) {
	req := deleteRequest{id, make(chan int)}
	c.deleteRequests <- req
	<-req.done
	c.sendUsernames()
}

// broadcastAll sends a message to all active clients in the same room.
func (c *Clients) broadcastAll(msg commons.Message, roomID string) {
	color.Blue("sending message to all users in room %s. Text: %s", roomID, msg.Text)
	for client := range c.getAll() {
		if err := client.send(msg); err != nil {
			color.Red("ERROR: %s", err)
			c.delete(client.id)
		}
	}
}

// broadcastAllExcept sends a message to all clients except for the one whose ID
// matches except in the same room.
func (c *Clients) broadcastAllExcept(msg commons.Message, except uuid.UUID, roomID string) {
	for client := range c.getAll() {
		if client == nil {
			continue
		}
		if client.id == except {
			continue
		}
		if err := client.send(msg); err != nil {
			color.Red("ERROR: %s", err)
			c.delete(client.id)
		}
	}
}

// broadcastOne sends a message to a single client with the ID matching dst.
func (c *Clients) broadcastOne(msg commons.Message, dst uuid.UUID) {
	client := <-c.get(dst)
	if client != nil {
		if err := client.send(msg); err != nil {
			color.Red("ERROR: %s", err)
			c.delete(client.id)
		}
	}
}

// broadcastOneExcept sends a message to any one client whose ID does not match except.
func (c *Clients) broadcastOneExcept(msg commons.Message, except uuid.UUID) {
	log.Printf("except in broadcastOneExcept: %v", except)
	for client := range c.getAll() {
		if client == nil {
			continue
		}
		if client.id == except {
			continue
		}
		log.Printf("client in broadcastOneExcept: %v", client.id)

		if err := client.send(msg); err != nil {
			color.Red("ERROR: %s", err)
			c.delete(client.id)
			continue
		}
		break
	}
}

// close closes a WebSocket connection and removes it from the list of clients in a
// concurrency safe manner.
func (c *Clients) close(id uuid.UUID) {
	c.mu.RLock()
	client, ok := c.list[id]
	if ok {
		if err := client.Conn.Close(); err != nil {
			color.Red("Error closing connection: %s\n", err)
		}
	} else {
		color.Red("Couldn't close connection: client not in list")
		return
	}
	color.Red("Removing %v from client list.\n", c.list[id].Username)
	c.mu.RUnlock()

	c.mu.Lock()
	delete(c.list, id)
	c.mu.Unlock()

}

// read reads a message over the client Conn, and stores the result in msg.
func (c *client) read(msg *commons.Message) error {
	err := c.Conn.ReadJSON(msg)

	c.mu.Lock()
	name := c.Username
	c.mu.Unlock()

	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			color.Red("Failed to read message from client %s: %v", name, err)
		}
		color.Red("client %v disconnected", name)
		room := getRoomByClientID(c.id)
		room.Clients.delete(c.id)
		return err
	}
	return nil
}

// send sends a message over the client Conn while protecting from
// concurrent writes.
func (c *client) send(v interface{}) error {
	c.writeMu.Lock()
	err := c.Conn.WriteJSON(v)
	c.writeMu.Unlock()
	return err
}

// sendUsernames sends a message containing the names of all active clients
// to the syncChan, to be broadcast to all clients and displayed in their editor.
func (c *Clients) sendUsernames() {
	var users string
	for client := range c.getAll() {
		users += client.Username + ","
	}

	syncChan <- commons.Message{Text: users, Type: commons.UsersMessage}
}
