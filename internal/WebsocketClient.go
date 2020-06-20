package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
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
		case NewJobMessage:
			var newJobMessage LorcNewJobMessage
			if err := json.Unmarshal(bytes.Trim(jsonMessage, "\x00"), &newJobMessage); err != nil {
				log.Printf("Could not unmarshal LorcMessage, error: %s \n", err)
				return
			} else {
				fmt.Println(newJobMessage.Job)
				go func() {
					if strings.Contains(newJobMessage.Job.Command, "|") {
						commandSlice := strings.Split(newJobMessage.Job.Command, "|")
						var commands []exec.Cmd
						for _, command := range commandSlice {
							cmdArgs := strings.Fields(strings.TrimSpace(command))
							_, err := exec.LookPath(cmdArgs[0])
							if err != nil {
								break
							}
							dir, _ := os.Getwd()
							for _, inputFile := range newJobMessage.Job.Files {
								fmt.Println("writing file to : " + dir + string(os.PathSeparator) + inputFile.FileName)
								err = ioutil.WriteFile(dir+string(os.PathSeparator)+inputFile.FileName, inputFile.FileContents, 0644)
								if err != nil {
									fmt.Println(err)
								}
								commands = append(commands, *exec.Command(cmdArgs[0], cmdArgs[1:]...))
							}
							if len(commandSlice) == len(commands) {
								finalOut, _ := commands[1].StdoutPipe()
								buf := bufio.NewReader(finalOut)

								r, w := io.Pipe()
								commands[0].Stdout = w
								commands[1].Stdin = r
								commands[0].Start()
								commands[1].Start()
								commands[0].Wait()
								w.Close()

								for {
									line, _, _ := buf.ReadLine()
									if line == nil {
										break
									}
									jobResult := NewLorcJobResultMessage(newJobMessage.Job.JobId, line)
									jsonResultMessage, _ := json.Marshal(jobResult)
									fmt.Println(string(line))
									fmt.Println(string(jsonResultMessage))
									ws.conn.WriteMessage(websocket.TextMessage, jsonResultMessage)
								}
								commands[1].Wait()

								jobResult := NewLorcJobDoneMessage(newJobMessage.Job.JobId)
								jsonResultMessage, _ := json.Marshal(jobResult)
								ws.conn.WriteMessage(websocket.TextMessage, jsonResultMessage)
							}
						}
					} else {
						cmdArgs := strings.Fields(newJobMessage.Job.Command)
						_, err := exec.LookPath(cmdArgs[0])
						if err == nil {
							dir, _ := os.Getwd()
							for _, inputFile := range newJobMessage.Job.Files {
								fmt.Println("writing file to : " + dir + string(os.PathSeparator) + inputFile.FileName)
								err = ioutil.WriteFile(dir+string(os.PathSeparator)+inputFile.FileName, inputFile.FileContents, 0644)
								if err != nil {
									fmt.Println(err)
								}
							}
							cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
							stdout, _ := cmd.StdoutPipe()
							cmd.Start()
							buf := bufio.NewReader(stdout)
							for {
								line, _, _ := buf.ReadLine()
								if line == nil {
									break
								}
								jobResult := NewLorcJobResultMessage(newJobMessage.Job.JobId, line)
								jsonResultMessage, _ := json.Marshal(jobResult)
								fmt.Println(string(line))
								fmt.Println(string(jsonResultMessage))
								ws.conn.WriteMessage(websocket.TextMessage, jsonResultMessage)
							}

							cmd.Wait()
							//jobResult := NewLorcJobDoneMessage(newJobMessage.Job.JobId)
							//jsonResultMessage, _ := json.Marshal(jobResult)
							//ws.conn.WriteMessage(websocket.TextMessage, jsonResultMessage)
						}
					}
				}()
			}
		default:
			fmt.Println("Unknown Job Type")
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
