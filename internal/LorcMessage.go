package internal

const (
	HelloMessage              string = "Hello"
	ClientCapabilitiesMessage string = "Capabilities"
	NewJobMessage             string = "NewJob"
	JobResultMessage          string = "JobResult"
	JobDoneMessage            string = "JobDone"
)

//{"type":""}
type LorcMessage struct {
	MessageType string `json:"type"`
}

type LorcCapabilitiesMessage struct {
	LorcMessage
	RequestedCapabilities map[string]bool `json:"capabilities"`
}

type LorcNewJobMessage struct {
	LorcMessage
	Job Job `json:"job"`
}

type LorcJobResultMessage struct {
	LorcMessage
	JobId  string `json:"jobId"`
	Output []byte `json:"output"`
}

type LorcJobDoneMessage struct {
	LorcMessage
	JobId string `json:"jobId"`
}

func NewLorcJobDoneMessage(jobId string) *LorcJobDoneMessage {
	return &LorcJobDoneMessage{
		*NewLorcMessageWithType(JobDoneMessage),
		jobId,
	}
}

func NewLorcJobResultMessage(jobId string, output []byte) *LorcJobResultMessage {
	return &LorcJobResultMessage{
		*NewLorcMessageWithType(JobResultMessage),
		jobId,
		output,
	}
}

func NewLorcNewJobMessage(job Job) *LorcNewJobMessage {
	return &LorcNewJobMessage{
		*NewLorcMessageWithType(NewJobMessage),
		job,
	}
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
