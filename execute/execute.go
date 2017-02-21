package execute

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

type Execute struct {
	timeout     int
	folder      string
	executables []string
	reportChan  chan int
}

type SingleExecutable struct {
	name       string
	cmd        *exec.Cmd
	pipeLock   *sync.Mutex
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

func New(timeout int, folder string) (*Execute, error) {
	executables, err := findExecutables(folder)

	if err != nil {
		return nil, err
	}

	return &Execute{
		timeout:     timeout,
		folder:      folder,
		executables: executables,
		reportChan:  make(chan int),
	}, nil
}

func (e *Execute) GetExecutableList() []string {
	return e.executables
}

func findExecutables(folder string) ([]string, error) {
	file, err := os.Open(folder)

	executables := make([]string, 0, 0)

	if err != nil {
		return nil, err
	}

	stats, err := file.Stat()

	if err != nil {
		return nil, err
	}

	if !stats.IsDir() {
		return nil, errors.New("Provided path isnt a directory")
	}

	files, err := file.Readdir(0)

	if err != nil {
		return nil, err
	}

	for _, file := range files {
		mode := file.Mode()

		//this definitely can be improved as it actually doesn't check if user current
		//effective user has permissions to execute file
		if mode.Perm()&os.ModePerm&0555 != 1 {
			executables = append(executables, path.Join(folder, file.Name()))
		}
	}

	return executables, nil
}

func (e *Execute) ExecuteExecutables() error {
	for index, execPath := range e.GetExecutableList() {
		go e.execSingle(execPath, index)
	}

	for {
		index := <-e.reportChan
		go e.execSingle(e.GetExecutableList()[index], index)
	}
}

func (e *Execute) execSingle(execPath string, index int) {
	s := &SingleExecutable{
		name:     path.Base(execPath),
		cmd:      exec.Command(execPath),
		pipeLock: &sync.Mutex{},
	}

	s.Exec()

	time.Sleep(time.Duration(e.timeout) * time.Millisecond)

	e.reportChan <- index
}

func (s *SingleExecutable) Exec() {
	err := s.SetupPipes()

	if err != nil {
		log.WithError(err).Errorf("[%s]Failed to create output pipe", s.name)
		return
	}

	clearOut := make(chan bool)
	clearErr := make(chan bool)

	go s.ReadPipe("stdout", clearOut)
	go s.ReadPipe("stderr", clearErr)

	log.Infof("[%s]Will start", s.name)

	err = s.cmd.Run()
	if err != nil {
		log.WithError(err).Errorf("[%s]Exited with an error", s.name)
	} else {
		log.Infof("[%s]Exited without an error", s.name)
	}

	clearErr <- true
	clearOut <- true
}

func (s *SingleExecutable) SetupPipes() error {
	s.pipeLock.Lock()

	defer s.pipeLock.Unlock()

	pipe, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to create stdout pipe for %s because of %s", s.name, err)
	}

	s.stdoutPipe = pipe

	pipe, err = s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to create stderr pipe for %s because of %s", s.name, err)
	}

	s.stderrPipe = pipe

	return nil
}

func (s *SingleExecutable) ReadPipe(t string, clear chan bool) {
	previousFailed := false
	p := make([]byte, 1024)
	var pipe io.ReadCloser

	s.pipeLock.Lock()
	if t == "stdout" {
		pipe = s.stdoutPipe
	} else {
		pipe = s.stderrPipe
	}
	s.pipeLock.Unlock()

	for {
		cnt, err := pipe.Read(p)

		if err != nil && !previousFailed && err != io.EOF {
			log.WithError(err).Debugf("[%s][%s]Failed to read from pipe", s.name, t)
			previousFailed = true
		}

		if err == nil && cnt > 0 {
			output := string(p[0:cnt])
			for _, line := range strings.Split(output, "\n") {
				if len(line) == 0 {
					continue
				}
				log.Debugf("[%s][%s]%s", s.name, t, line)
			}

			previousFailed = false
		}

		select {
		case <-clear:
			return
		case <-time.After(time.Millisecond * 25):
			continue
		}
	}
}
