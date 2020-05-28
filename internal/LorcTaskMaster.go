package internal

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type TaskMaster struct {
	clients map[string]LorcClient
}

func NewTaskMaster() *TaskMaster {
	return &TaskMaster{
		make(map[string]LorcClient),
	}
}

func (t *TaskMaster) AddClient(lorcClient LorcClient) {
	t.clients[lorcClient.Id] = lorcClient
}

func (t *TaskMaster) removeClient(clientId string) {
	delete(t.clients, clientId)
}

func (t *TaskMaster) GetClients(writer http.ResponseWriter, request *http.Request) {
	clientsJson, _ := json.Marshal(t.clients)
	writer.Write(clientsJson)
}

func (t *TaskMaster) newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func (t *TaskMaster) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	uuid, err := t.newUUID()
	if err != nil {
		log.Println(err)
		return
	}
	client := &LorcClient{conn: conn, send: make(chan []byte), Id: uuid, master: *t}
	t.AddClient(*client)
	fmt.Println("Got new client!")
	go client.writePump()
	go client.readPump()
}
