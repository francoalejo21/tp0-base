package common

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Infof("action: shutdown | result: in_progress | signal: SIGTERM | client_id: %v", c.config.ID)
		if c.conn != nil {
			c.conn.Close()
			log.Infof("action: close_resource | result: success | resource: client_socket | client_id: %v", c.config.ID)
		}
		log.Infof("action: shutdown | result: success | client_id: %v", c.config.ID)
		os.Exit(0)
	}()

	// ── Build Bet from environment variables ──────────────────────────────────
	bet, err := protocol.NewBet(
		c.config.ID, // agency ID viene de la config (seteado por Docker Compose)
		mustGetenv("NOMBRE"),
		mustGetenv("APELLIDO"),
		mustGetenv("DOCUMENTO"),
		mustGetenv("NACIMIENTO"),
		mustGetenv("NUMERO"),
	)
	if err != nil {
		log.Fatalf("action: build_bet | result: fail | client_id: %v | error: %v", c.config.ID, err)
	}

	// ── Connect ───────────────────────────────────────────────────────────────
	if err := c.createClientSocket(); err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	defer func() {
		c.conn.Close()
		log.Infof("action: close_resource | result: success | resource: client_socket | client_id: %v", c.config.ID)
	}()
	// ── Send bet ──────────────────────────────────────────────────────────────
	if err := protocol.SendBet(c.conn, bet); err != nil {
		log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

	// ── Wait for server confirmation ──────────────────────────────────────────
	if err := protocol.ReceiveConfirmation(c.conn); err != nil {
		log.Errorf("action: receive_confirmation | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

	log.Infof(
		"action: apuesta_enviada | result: success | dni: %v | numero: %v",
		bet.Document,
		bet.Number,
	)
}

func mustGetenv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("action: read_env | result: fail | variable: %v | error: not set or empty", key)
	}
	return val
}
