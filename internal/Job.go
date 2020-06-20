package internal

type Job struct {
	JobId   string
	Command string
	Files   []InputFile
	Result  string
}

type InputFile struct {
	FileName     string
	FileContents []byte
}

func (job *Job) UpdateResult(newResultData string) {
	if len(job.Result) == 0 {
		job.Result = newResultData
	} else {
		job.Result = job.Result + "\n" + newResultData
	}
}

func NewInputFile(fileName string, fileContents []byte) *InputFile {
	return &InputFile{FileName: fileName, FileContents: fileContents}
}

func NewJob(jobId string, commandString string) *Job {
	return &Job{JobId: jobId, Command: commandString, Files: []InputFile{}}
}

func NewJobWithFiles(jobId string, commandString string, files []InputFile) *Job {
	return &Job{JobId: jobId, Command: commandString, Files: files}
}
