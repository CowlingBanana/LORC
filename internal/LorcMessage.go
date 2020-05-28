package internal

const (
	HelloMessage              string = "Hello"
	ClientCapabilitiesMessage string = "Capabilities"
)

//{"type":""}
type LorcMessage struct {
	MessageType string `json:"type"`
}

type LorcCapabilitiesMessage struct {
	LorcMessage
	RequestedCapabilities map[string]bool `json:"capabilities"`
}

func NewLorcCapabilitiesMessage() *LorcCapabilitiesMessage {
	return &LorcCapabilitiesMessage{
		*NewLorcMessageWithType(ClientCapabilitiesMessage),
		make(map[string]bool),
	}
}

func NewLorcMessageWithType(messageType string) *LorcMessage {
	return &LorcMessage{
		messageType,
	}
}

func NewLorcMessage() *LorcMessage {
	return &LorcMessage{}
}
