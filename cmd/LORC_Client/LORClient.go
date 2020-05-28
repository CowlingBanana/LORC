package main

import (
	"../../internal"
	"fmt"
)

func main() {
	fmt.Println("Hello from the client")
	webSocketClient := internal.NewWebSocketClient()
	webSocketClient.StartWebsocketClient()

}
