package main

import (
	"../../internal"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {
	fmt.Println("hello!")
	taskMaster := internal.NewTaskMaster()
	router := mux.NewRouter()
	router.HandleFunc("/clients", taskMaster.GetClients).Methods("GET")
	router.HandleFunc("/jobs", taskMaster.JobsHandler).Methods("GET", "POST")
	router.HandleFunc("/jobs/{jobId}", taskMaster.JobViewer).Methods("GET")
	router.HandleFunc("/jobs/{jobId}/update", taskMaster.JobViewer).Methods("GET")
	router.HandleFunc("/workflows", taskMaster.ViewWorkflows).Methods("GET")
	router.HandleFunc("/workflows/new", taskMaster.ViewWorkflows).Methods("GET", "POST")
	router.HandleFunc("/workflows/{workflowName}", taskMaster.ViewWorkflow).Methods("GET", "POST")
	router.HandleFunc("/ws", taskMaster.ServeWs)
	http.ListenAndServe(":8888", router)
}
