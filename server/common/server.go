package common

import (
	"errors"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type Server struct {
	listener         net.Listener
	running          bool
	clientAmounts    int
	finishedAgencies int
	drawCompleted    bool
	stateMutex       sync.Mutex // Mutex para cambiar estado de finishedAgencies y drawCompleted de forma atómica
	fileMutex        sync.Mutex
	wg               sync.WaitGroup
}

func NewServer(port string, clients int) (*Server, error) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener:         listener,
		running:          true,
		clientAmounts:    clients,
		finishedAgencies: 0,
		drawCompleted:    false,
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
		s.listener.Close()
		log.Info("action: close_resource | result: success | resource: server_socket")
	}()
}

func (s *Server) Run() {
	for s.running {
		conn := s.acceptNewConnection()
		if conn != nil {
			s.wg.Add(1)
			go s.handleClientConnection(conn)
		}
	}
	log.Info("action: shutdown | result: Esperando a que finalicen las conexiones activas...")
	s.wg.Wait()
	log.Info("action: shutdown | result: success | msg: Servidor apagado correctamente")
}

func (s *Server) handleClientConnection(conn net.Conn) {
	defer s.wg.Done()
	defer s.cleanupConnection(conn)

	for {
		cmd, data, err := protocol.ReadCommand(conn)
		if err != nil {
			s.handleDisconnection(err)
			break
		}

		switch cmd {
		case protocol.CmdBatch:
			bets, err := protocol.DeserializeBatch(data)
			if err != nil {
				log.Errorf("action: deserialize | result: fail | error: %v", err)
				continue
			}
			s.storeAndConfirm(conn, bets)

		case protocol.CmdFin:
			s.handleFin()
			return

		case protocol.CmdQuery:
			s.handleQuery(conn, string(data))
			return

		default:
			log.Errorf("Comando desconocido: %s", cmd)
			return
		}
	}
}

func (s *Server) cleanupConnection(conn net.Conn) {
	conn.Close()
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

	s.fileMutex.Lock()
	err := protocol.StoreBets(bets)
	s.fileMutex.Unlock()
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

func (s *Server) handleFin() {
	s.stateMutex.Lock()
	s.finishedAgencies++
	log.Debugf("Agencia finalizada: %d/%d", s.finishedAgencies, s.clientAmounts)

	if s.finishedAgencies == s.clientAmounts && !s.drawCompleted {
		log.Infof("action: sorteo | result: success")
		s.drawCompleted = true
	}
	s.stateMutex.Unlock()
}

func (s *Server) handleQuery(conn net.Conn, agencyID string) {
	s.stateMutex.Lock()
	completed := s.drawCompleted
	s.stateMutex.Unlock()
	if !completed {
		_ = protocol.SendWait(conn)
		return
	}

	winners := s.getWinnersForAgency(agencyID)
	_ = protocol.SendWinners(conn, winners)
}

func (s *Server) getWinnersForAgency(agencyID string) []string {
	outCh, errCh := protocol.LoadBets()
	var winners []string

	for bet := range outCh {
		if strconv.Itoa(bet.Agency) == agencyID {
			if protocol.HasWon(bet) {
				winners = append(winners, bet.Document)
			}
		}
	}

	if err := <-errCh; err != nil {
		log.Errorf("action: load_bets | result: fail | error: %v", err)
	}

	return winners
}
