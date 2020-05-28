package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
)

type WebSocketClient struct {
	// The websocket connection.
	conn *websocket.Conn
}

func (ws *WebSocketClient) parseLorcServerMessage(jsonMessage []byte) {
	newLorcMessage := NewLorcMessage()
	if err := json.Unmarshal(bytes.Trim(jsonMessage, "\x00"), &newLorcMessage); err != nil {
		log.Printf("Could not unmarshal LorcMessage, error: %s \n", err)
		return
	} else {
		switch newLorcMessage.MessageType {
		case ClientCapabilitiesMessage:
			capabilitiesMessage := NewLorcCapabilitiesMessage()
			if err := json.Unmarshal(bytes.Trim(jsonMessage, "\x00"), &capabilitiesMessage); err != nil {
				log.Printf("Could not unmarshal LorcMessage, error: %s \n", err)
				return
			} else {
				for capability, exists := range capabilitiesMessage.RequestedCapabilities {
					fmt.Println("Capability:", capability, "Exists:", exists)
					_, err := exec.LookPath(capability)
					if err == nil {
						fmt.Println("Found tool: " + capability)
						capabilitiesMessage.RequestedCapabilities[capability] = true
					}
				}
				jsonData, _ := json.Marshal(capabilitiesMessage)

				ws.conn.WriteMessage(websocket.TextMessage, jsonData)
			}
		}
	}
}

func NewWebSocketClient() *WebSocketClient {
	return &WebSocketClient{}
}

func (ws *WebSocketClient) StartWebsocketClient() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:8888", Path: "/ws"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	ws.conn = c
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			ws.parseLorcServerMessage(message)
		}
	}()

	helloMessage := NewLorcMessageWithType(HelloMessage)
	jsonMessage, err := json.Marshal(helloMessage)

	ws.conn.WriteMessage(websocket.TextMessage, jsonMessage)

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			}
			return
		}
	}
}
