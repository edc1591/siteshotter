package phantom

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/timehop/golog/log"
)

type RenderServer interface {
	Start() error
	Shutdown() error
	RenderPage(url string) ([]byte, error)
	CurrentLoad() int
	AtCapacity() bool
}

var (
	port = 7000
	mu   sync.Mutex
)

const (
	// PhantomJS enforces a limit of 10 concurrent web requests
	maxConcurrentRequests = 10
)

func nextPort() string {
	p := port
	mu.Lock()
	port++
	mu.Unlock()
	return fmt.Sprint(p)
}

func NewRenderServer() RenderServer {
	return &renderServer{}
}

// Not safe for concurrent use!
type renderServer struct {
	load    int
	process *os.Process
	port    string
	mu      sync.Mutex
}

func (s *renderServer) Start() error {
	if s.process != nil {
		return errors.New("phantom: server already started")
	}

	host := "http://127.0.0.1:" + os.Getenv("PORT")
	port := nextPort()
	cmd := exec.Command("phantomjs", "js/server.js", host, port)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	s.process = cmd.Process
	s.port = port
	s.waitForUp()
	return nil
}

func (s *renderServer) waitForUp() {
	for {
		time.Sleep(200 * time.Millisecond)
		if _, err := http.Get("http://127.0.0.1:" + s.port + "/ping"); err == nil {
			log.Info("siteshotter/phantom", "Up", "address", "127.0.0.1:"+s.port)
			return
		}
	}
}

func (s *renderServer) Shutdown() error {
	if s.process == nil {
		return nil
	}
	log.Info("siteshotter/phantom", "Shutting down...", "port", s.port)
	if err := s.process.Kill(); err != nil {
		log.Error("siteshotter/phantom", "Failed to kill phantomjs process", "pid", s.process.Pid)
		return err
	}
	return nil
}

func (s *renderServer) RenderPage(url string) ([]byte, error) {
	if s.AtCapacity() {
		return nil, errors.New("phantom: at capacity")
	}
	s.mu.Lock()
	s.load++
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.load--
		s.mu.Unlock()
	}()

	resp, err := http.Get("http://127.0.0.1:" + s.port + "/image.png?url=" + url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// Indicates how many requests are currently being processed
func (s *renderServer) CurrentLoad() int {
	return s.load
}

// Indicates if the server is processing all the requests it can
func (s *renderServer) AtCapacity() bool {
	return s.load >= maxConcurrentRequests
}
