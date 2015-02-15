package phantom

import "sync"

func NewRenderServerPool(size int) RenderServer {
	servers := make([]RenderServer, size)
	for i := 0; i < size; i++ {
		servers[i] = NewRenderServer()
	}
	return &renderServerPool{servers: servers}
}

// Not safe for concurrent use!
type renderServerPool struct {
	servers []RenderServer
}

func (s *renderServerPool) Start() error {
	failures := make(chan error, len(s.servers))
	defer close(failures)
	var wg sync.WaitGroup
	for _, server := range s.servers {
		wg.Add(1)
		go func(svr RenderServer) {
			defer wg.Done()
			if err := svr.Start(); err != nil {
				failures <- err
			}
		}(server)
	}
	wg.Wait()
	select {
	case err := <-failures:
		s.Shutdown()
		return err
	default:
		return nil
	}
}

func (s *renderServerPool) Shutdown() error {
	failures := make(chan error, len(s.servers))
	defer close(failures)
	var wg sync.WaitGroup
	for _, server := range s.servers {
		wg.Add(1)
		go func(svr RenderServer) {
			defer wg.Done()
			if err := svr.Shutdown(); err != nil {
				failures <- err
			}
		}(server)
	}
	wg.Wait()
	select {
	case err := <-failures:
		s.Shutdown()
		return err
	default:
		return nil
	}
}

func (s *renderServerPool) RenderPage(url string, selector string) ([]byte, error) {
	var leastBusyServer RenderServer
	for _, server := range s.servers {
		if leastBusyServer == nil || leastBusyServer.CurrentLoad() > server.CurrentLoad() {
			leastBusyServer = server
		}
	}
	return leastBusyServer.RenderPage(url, selector)
}

func (s *renderServerPool) CurrentLoad() int {
	load := 0
	for _, server := range s.servers {
		load += server.CurrentLoad()
	}
	return load
}

func (s *renderServerPool) AtCapacity() bool {
	for _, server := range s.servers {
		if !server.AtCapacity() {
			return false
		}
	}
	return true
}
