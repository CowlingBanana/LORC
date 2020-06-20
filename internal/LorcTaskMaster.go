package internal

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
)

var templates *template.Template

type TaskMaster struct {
	clients     map[string]LorcClient
	jobs        map[string]Job
	jobsChannel chan Job
	Workflows   map[string]Workflow
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
		make(map[string]Job),
		make(chan Job),
		make(map[string]Workflow),
	}
	go func() {
		for {
			newJob := <-master.jobsChannel
			for _, client := range master.clients {
				if !client.executingTask {
					client.sendJob(newJob)
				}
			}
		}
	}()
	return master
}

func (t *TaskMaster) ViewWorkflow(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		vars := mux.Vars(request)
		requestedWorkflow := t.Workflows[vars["workflowName"]]
		templates.ExecuteTemplate(writer, "viewWorkflow.gohtml", requestedWorkflow)
	case http.MethodPost:
	default:
		fmt.Fprintln(writer, "Go Away.")
	}
}

func (t *TaskMaster) ViewWorkflows(writer http.ResponseWriter, request *http.Request) {
	if path.Base(request.URL.Path) == "new" {
		switch request.Method {
		case http.MethodGet:
			templates.ExecuteTemplate(writer, "newworkflow.gohtml", t.Workflows)
		case http.MethodPost:
			workflowName := request.FormValue("name")
			newWorkflow := &Workflow{[]Job{}, workflowName}
			t.Workflows[workflowName] = *newWorkflow
			http.Redirect(writer, request, "/workflows", 302)
		default:
			fmt.Fprintln(writer, "Go Away.")
		}
	} else {
		templates.ExecuteTemplate(writer, "workflows.gohtml", t.Workflows)
	}
}

func (t *TaskMaster) JobViewer(writer http.ResponseWriter, request *http.Request) {
	if path.Base(request.URL.Path) == "update" {
		vars := mux.Vars(request)
		requestedJob := t.jobs[vars["jobId"]]
		writer.Write([]byte(requestedJob.Result))
	} else {
		vars := mux.Vars(request)
		requestedJob := t.jobs[vars["jobId"]]
		templates.ExecuteTemplate(writer, "job.gohtml", requestedJob)
	}
}

func (t *TaskMaster) JobsHandler(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		fmt.Println("Handling get")
		templates.ExecuteTemplate(writer, "jobs.gohtml", t.jobs)
	case http.MethodPost:
		fmt.Println("Handling post")
		var fileBytes []byte
		var inputFiles []InputFile

		commandString := request.FormValue("command")
		fmt.Println(commandString)

		request.ParseMultipartForm(32 << 20)
		file, handler, err := request.FormFile("file")

		if err == nil {
			defer file.Close()
			fileBytes = make([]byte, handler.Size)
			file.Read(fileBytes)
			inputFiles = append(inputFiles, *NewInputFile(handler.Filename, fileBytes))
		}
		jobId, err := NewUUID()
		if err == nil {
			job := *NewJobWithFiles(jobId, commandString, inputFiles)
			go func() { t.jobsChannel <- job }()
			t.jobs[jobId] = job
			templates.ExecuteTemplate(writer, "jobs.gohtml", t.jobs)
		} else {
			fmt.Println(err)
			fmt.Fprintln(writer, "error")
		}

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
