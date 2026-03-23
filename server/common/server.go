package common

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type Server struct {
	listener net.Listener
	running  bool
	conn     net.Conn
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
		if s.conn != nil {
			s.conn.Close()
		}
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
	s.conn = conn
	defer func() {
		conn.Close()
		s.conn = nil
		log.Info("action: close_resource | result: success | resource: client_socket")
	}()

	bet, err := protocol.ReceiveBet(conn)
	if err != nil {
		log.Errorf("action: receive_bet | result: fail | error: %v", err)
		return
	}

	if err := protocol.StoreBets([]protocol.Bet{bet}); err != nil {
		log.Errorf("action: store_bet | result: fail | error: %v", err)
		return
	}

	log.Infof(
		"action: apuesta_almacenada | result: success | dni: %v | numero: %v",
		bet.Document,
		bet.Number,
	)

	if err := protocol.SendConfirmation(conn); err != nil {
		log.Errorf("action: send_confirmation | result: fail | error: %v", err)
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
