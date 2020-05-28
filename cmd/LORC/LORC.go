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
	router.HandleFunc("/ws", taskMaster.ServeWs)

	http.ListenAndServe(":8888", router)
}
