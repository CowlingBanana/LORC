package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 8192 * 8192
)

type LorcClient struct {
	//unique id of client
	Id string `json:"id"`
	// The websocket connection.
	conn *websocket.Conn `json:"-"`
	// Buffered channel of outbound messages.
	send chan []byte `json:"-"`
	//LORC Server that controls this client
	master TaskMaster `json:"-"`
	//is client running a job
	executingTask bool
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192 * 8192,
	WriteBufferSize: 8192 * 8192,
}

// Actual writer to LORC Client's websocket connection
func (c *LorcClient) write(mt int, payload []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(mt, payload)
}

func (c *LorcClient) parseMessage(jsonMessage []byte) {
	newLorcMessage := NewLorcMessage()
	if err := json.Unmarshal(bytes.Trim(jsonMessage, "\x00"), &newLorcMessage); err != nil {
		log.Printf("Could not unmarshal LorcMessage, error: %s \n", err)
		return
	} else {
		switch newLorcMessage.MessageType {
		case HelloMessage:
			replyMessage := NewLorcCapabilitiesMessage()
			replyMessage.MessageType = ClientCapabilitiesMessage
			replyMessage.RequestedCapabilities["ffuf"] = false
			jsonData, _ := json.Marshal(replyMessage)
			c.send <- jsonData
			break
		case ClientCapabilitiesMessage:
			var capabilitiesMessage LorcCapabilitiesMessage
			if err := json.Unmarshal(bytes.Trim(jsonMessage, "\x00"), &capabilitiesMessage); err != nil {
				log.Printf("Could not unmarshal LorcMessage, error: %s \n", err)
				return
			} else {
				for capability, exists := range capabilitiesMessage.RequestedCapabilities {
					fmt.Println("Capability:", capability, "Exists:", exists)
				}
			}
			break
		case JobResultMessage:
			var jobResultMessage LorcJobResultMessage
			if err := json.Unmarshal(bytes.Trim(jsonMessage, "\x00"), &jobResultMessage); err != nil {
				log.Printf("Could not unmarshal LorcMessage, error: %s \n", err)
				return
			} else {
				fmt.Printf("%s\n", string(jobResultMessage.Output))
				job := c.master.jobs[jobResultMessage.JobId]
				job.UpdateResult(string(jobResultMessage.Output))
				c.master.jobs[jobResultMessage.JobId] = job
			}
		case JobDoneMessage:
			var jobResultMessage LorcJobDoneMessage
			if err := json.Unmarshal(bytes.Trim(jsonMessage, "\x00"), &jobResultMessage); err != nil {
				log.Printf("Could not unmarshal LorcMessage, error: %s \n", err)
				return
			} else {
				//delete(c.master.jobs,jobResultMessage.JobId)
			}
		default:
			fmt.Println("Unkown message type")
		}
	}
}

// writePump pumps messages from the LORC Server to the  LORC Client's websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *LorcClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The LORC Server closed the channel.
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.write(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the LORC Client's websocket connection to the LORC Server.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *LorcClient) readPump() {
	defer func() {
		//c.hub.rooms[c.roomName].deleteClient(c.name)
		close(c.send)
		//c.hub.removeClientFromServerList(c.name)
		log.Println("Client Leaving")
		c.master.removeClient(c.Id)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			} else {
				log.Printf("All Errors: %v", err)
			}
			break
		}
		c.parseMessage(msg)

	}
}

func (c *LorcClient) sendJob(job Job) {
	newJobMessage := NewLorcNewJobMessage(job)
	if newJobMessage != nil {
		jsonMessage, _ := json.Marshal(newJobMessage)
		c.send <- jsonMessage
	}
}
