package common

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type Server struct {
	listener net.Listener
	running  bool
}

func NewServer(port string) (*Server, error) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: listener,
		running:  true,
	}

	// Manejo de SIGTERM
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM)

	go func() {
		<-sigchan
		log.Info("action: shutdown | result: in_progress | signal: SIGTERM")
		s.running = false
		s.listener.Close()
		log.Info("action: close_resource | result: success | resource: server_socket")
	}()

	return s, nil
}

func (s *Server) Run() {
	for s.running {
		conn := s.acceptNewConnection()
		if conn != nil {
			s.handleClientConnection(conn)
		}
	}
	log.Info("action: shutdown | result: success")
}

func (s *Server) handleClientConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		log.Info("action: close_resource | result: success | resource: client_socket")
	}()

	buffer := make([]byte, 1024)
	// TODO: Modify the receive to avoid short-reads
	n, err := conn.Read(buffer)
	if err != nil {
		log.Criticalf("action: receive_message | result: fail | error: %v", err)
		return
	}

	msg := string(buffer[:n])
	addr := conn.RemoteAddr().String()

	log.Infof("action: receive_message | result: success | ip: %s | msg: %s", addr, msg)
	// TODO: Modify the send to avoid short-writes
	_, err = conn.Write([]byte(msg + "\n"))
	if err != nil {
		log.Criticalf("action: send_message | result: fail | error: %v", err)
	}
}

func (s *Server) acceptNewConnection() net.Conn {
	log.Info("action: accept_connections | result: in_progress")

	conn, err := s.listener.Accept()
	if err != nil {
		return nil
	}

	addr := conn.RemoteAddr().String()
	log.Infof("action: accept_connections | result: success | ip: %s", addr)

	return conn
}

func main() {
	server, err := NewServer("")
	if err != nil {
		log.Fatal(err)
	}

	server.Run()
}
