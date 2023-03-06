package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait   = 10 * time.Second
	pingPeriod = (pongWait * 9) / 10 // 90% of pongWait
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

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		fmt.Println("error setting read deadline: ", err)
		return
	}

	c.connection.SetPongHandler(func(string) error {
		fmt.Println("pong")
		return c.connection.SetReadDeadline(time.Now().Add(pongWait))
	})

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

	ticker := time.NewTicker(pingPeriod)

	for {
		select {
		case message, ok := <-c.egress:
			// if the channel is closed, exit the goroutine
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					fmt.Println("connection closed: ", err)
				}
				return
			}

			// parse the message to json and send it to the client.
			// if there is an error, log it and continue
			data, err := json.Marshal(message)
			if err != nil {
				fmt.Println("error marshalling message: ", err)
				continue
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				fmt.Println("error sending message: ", err)
			}
			fmt.Println("message sent: ", string(data))

		case <-ticker.C:
			fmt.Println("ping")
			//send ping to the client
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				fmt.Println("error sending ping: ", err)
				return
			}
		}
	}
}
