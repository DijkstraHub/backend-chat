package main

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type ClientList map[*Client]struct{}

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	egress     chan Event
}

func NewClient(connection *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: connection,
		manager:    manager,
		egress:     make(chan Event),
	}
}

func (c *Client) read() {
	defer func() {
		c.manager.removeClient(c)
	}()
	for {
		_, message, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Println("error: ", err)
			}
			break
		}

		var event Event
		if err := json.Unmarshal(message, &event); err != nil {
			fmt.Println("error unmarshalling event: ", err)
			continue
		}
		if err := c.manager.routeEvent(event, c); err != nil {
			fmt.Println("error routing event: ", err)
			continue
		}
	}
}

func (c *Client) write() {
	defer func() {
		c.manager.removeClient(c)
	}()
	for {
		message, ok := <-c.egress
		if !ok {
			if err := c.connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
				fmt.Println("connection closed: ", err)
			}
			return
		}

		data, err := json.Marshal(message)
		if err != nil {
			fmt.Println("error marshalling message: ", err)
			continue
		}
		if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Println("error sending message: ", err)
		}
		fmt.Println("message sent: ", string(data))
	}
}
