package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"websocket-go/db"
)

var (
	pongWait = 10 * time.Second

	pingInterval = (pongWait * 9) / 10
)

type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	username   string

	//egress is used to avoid concurrent writes on websocket connection
	egress chan []byte
}

type Message struct {
	Content   string    `json:"content"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}

func NewClient(conn *websocket.Conn, manager *Manager, username string) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		username:   username,
		egress:     make(chan []byte),
	}
}

func (c *Client) readMessages() {
	defer func() {
		//cleanup the manager
		c.manager.removeClient(c)
	}()

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}

	//c.connection.SetReadLimit(512)

	c.connection.SetPongHandler(c.pongHandler)

	for {
		_, payload, err := c.connection.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		content := string(payload)

		dbMessage, err := db.AddMessage(c.username, content)
		if err != nil {
			log.Printf("Error saving message to database: %v", err)
		} else {
			log.Printf("Message saved to database with ID: %s", dbMessage.ID.Hex())
		}

		message := Message{
			Content:   content,
			Username:  c.username,
			Timestamp: time.Now(),
		}

		messageJSON, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			continue
		}

		//log.Printf("Message received: [Type: %d] %s", messageType, string(payload))

		//send the message to all clients
		for wsclient := range c.manager.clients {
			if wsclient != c {
				wsclient.egress <- messageJSON
			}
		}

		//log.Println(messageType)
		//log.Println(string(payload))
	}
}

func (c *Client) writeMessages() {
	defer func() {
		//cleanup the manager
		c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("connection closed: ", err)
				}
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("failed to send message: %v", err)
			}
			log.Println("message sent")

		case <-ticker.C:
			log.Println("ping")

			//send a ping to the client
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("write msg err: ", err)
				return
			}
		}
	}

}

func (c *Client) pongHandler(pongMsg string) error {
	log.Println("pong")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
