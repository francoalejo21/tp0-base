package common

import (
	"errors"
	"io"
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
	return s, nil
}

func (s *Server) SetupSignalHandler() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM)

	go func() {
		<-sigchan
		log.Info("action: shutdown | result: in_progress | signal: SIGTERM")
		s.running = false
		if s.conn != nil {
			s.conn.Close()
		}
		s.listener.Close()
		log.Info("action: close_resource | result: success | resource: server_socket")
	}()
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
	defer s.cleanupConnection(conn)

	for {
		err := s.processNextBatch(conn)
		if err != nil {
			s.handleDisconnection(err)
			break
		}
	}
}

func (s *Server) cleanupConnection(conn net.Conn) {
	conn.Close()
	s.conn = nil
	log.Info("action: close_resource | result: success | resource: client_socket")
}

func (s *Server) handleDisconnection(err error) {
	if errors.Is(err, io.EOF) {
		log.Info("action: client_finished | result: success")
	} else {
		log.Errorf("action: connection_error | result: fail | error: %v", err)
	}
}

func (s *Server) storeAndConfirm(conn net.Conn, bets []protocol.Bet) error {
	amount := len(bets)

	err := protocol.StoreBets(bets)
	if err != nil {
		log.Errorf("action: bets_almacenadas | result: fail | cantidad: %v | error: %v", amount, err)
		_ = protocol.SendBatchConfirmation(conn, false)
		return err
	}

	log.Infof("action: apuesta_recibida | result: success | cantidad: %v", amount)
	err = protocol.SendBatchConfirmation(conn, true)
	if err != nil {
		log.Errorf("action: send_confirmation | result: fail | error: %v", err)
		return err
	}
	return nil
}

func (s *Server) processNextBatch(conn net.Conn) error {
	bets, err := protocol.ReceiveBatch(conn)
	if err != nil {
		return err
	}
	return s.storeAndConfirm(conn, bets)
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
