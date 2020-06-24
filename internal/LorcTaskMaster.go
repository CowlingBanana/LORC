package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
)

var templates *template.Template

type TaskMaster struct {
	clients   map[string]LorcClient
	workflows map[string]Workflow
}

func NewTaskMaster() *TaskMaster {
	var err error
	templates, err = template.ParseGlob("./static/*")
	if err != nil {
		log.Println("Cannot parse templates:", err)
		os.Exit(-1)
	}
	master := &TaskMaster{
		make(map[string]LorcClient),
		make(map[string]Workflow),
	}
	return master
}

func (t *TaskMaster) isClientAvailable() (bool, LorcClient) {
	for _, client := range t.clients {
		if !client.executingTask {
			return true, client
		}
	}
	return false, LorcClient{}
}

func (t *TaskMaster) RunWorkflow(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	requestedWorkflow := t.workflows[vars["workflowName"]]
	go func() {
		for _, job := range requestedWorkflow.Jobs {
			fmt.Printf("adding job to hopper: %v \n", job)
			for {
				if available, client := t.isClientAvailable(); available {
					client.sendJob(job)
					client.executingTask = true
					t.clients[client.Id] = client
					break
				}
			}
		}
	}()
	http.Redirect(writer, request, "/workflows/"+vars["workflowName"], http.StatusFound)
}

func (t *TaskMaster) ViewWorkflow(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	requestedWorkflow := t.workflows[vars["workflowName"]]
	switch request.Method {
	case http.MethodGet:
		//view workflow
		templates.ExecuteTemplate(writer, "viewWorkflow.gohtml", requestedWorkflow)
	case http.MethodPost:
		//add job to workflow
		var inputFiles []InputFile
		var commandString string
		var mr *multipart.Reader
		var err error
		var part *multipart.Part
		if mr, err = request.MultipartReader(); err != nil {
			log.Printf("Hit error while opening multipart reader: %s", err.Error())
			writer.WriteHeader(500)
			fmt.Fprintf(writer, "Error occured during upload")
			return
		}
		chunk := make([]byte, 4096)
		var finalBytes []byte
		// continue looping through all parts, *multipart.Reader.NextPart() will
		// return an End of File when all parts have been read.
		for {
			var uploaded bool
			if part, err = mr.NextPart(); err != nil {
				if err != io.EOF {
					log.Printf("Hit error while fetching next part: %s", err.Error())
					writer.WriteHeader(500)
					fmt.Fprintf(writer, "Error occured during upload")
				} else {
					log.Printf("Hit last part of multipart upload")
					writer.WriteHeader(200)
					jobId, err := NewUUID()
					if err == nil {
						job := *NewJobWithFiles(jobId, commandString, inputFiles, requestedWorkflow.Name)
						t.workflows[vars["workflowName"]].Jobs[jobId] = job
						fmt.Println(t.workflows)
						templates.ExecuteTemplate(writer, "viewWorkflow.gohtml", t.workflows[vars["workflowName"]])

					} else {
						fmt.Println(err)
						fmt.Fprintln(writer, "error")
					}
				}
				return
			}
			// at this point the filename and the mimetype is known
			log.Printf("Upload part: %v\n", part)

			if part.FormName() == "command" {
				if _, err = part.Read(chunk); err != nil {
					if err != io.EOF {
						log.Printf("Hit error while reading chunk: %s", err.Error())
						writer.WriteHeader(500)
						fmt.Fprintf(writer, "Error occured during upload")
						return
					}
				}
				commandString = string(bytes.Trim(chunk, "\x00"))
			} else {
				for !uploaded {
					if _, err = part.Read(chunk); err != nil {
						if err != io.EOF {
							log.Printf("Hit error while reading chunk: %s", err.Error())
							writer.WriteHeader(500)
							fmt.Fprintf(writer, "Error occured during upload")
							return
						}
						uploaded = true
					}
					finalBytes = append(finalBytes, bytes.Trim(chunk, "\x00")...)
				}
				inputFiles = append(inputFiles, *NewInputFile(part.FileName(), finalBytes))
				finalBytes = nil
			}
		}
	default:
		fmt.Fprintln(writer, "Go Away.")
	}
}

func (t *TaskMaster) ViewWorkflows(writer http.ResponseWriter, request *http.Request) {
	if path.Base(request.URL.Path) == "new" {
		switch request.Method {
		case http.MethodGet:
			//view workflow
			templates.ExecuteTemplate(writer, "newworkflow.gohtml", t.workflows)
		case http.MethodPost:
			//add new workflow
			workflowName := request.FormValue("name")
			newWorkflow := &Workflow{make(map[string]Job), workflowName}
			t.workflows[workflowName] = *newWorkflow
			http.Redirect(writer, request, "/workflows", 302)
		default:
			fmt.Fprintln(writer, "Go Away.")
		}
	} else {
		templates.ExecuteTemplate(writer, "workflows.gohtml", t.workflows)
	}
}

func (t *TaskMaster) JobViewer(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	if path.Base(request.URL.Path) == "update" {
		writer.Write([]byte(t.workflows[vars["workflowName"]].Jobs[vars["jobId"]].Result))
	} else {
		//view whole job
		templates.ExecuteTemplate(writer, "job.gohtml", t.workflows[vars["workflowName"]].Jobs[vars["jobId"]])
	}
}

func (t *TaskMaster) JobsHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	requestedWorkflow := t.workflows[vars["workflowName"]]
	switch request.Method {
	case http.MethodGet:
		//view all jobs
		fmt.Println("Handling get")
		templates.ExecuteTemplate(writer, "jobs.gohtml", requestedWorkflow.Jobs)
	default:
		fmt.Fprintln(writer, "Go Away.")
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

func (t *TaskMaster) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	uuid, err := NewUUID()
	if err != nil {
		log.Println(err)
		return
	}
	client := &LorcClient{conn: conn, send: make(chan []byte), Id: uuid, master: *t, executingTask: false}
	t.AddClient(*client)
	fmt.Println("Got new client!")
	go client.writePump()
	go client.readPump()
}
