package common

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

const MaxSizeBatch = 8 * 1024 // 8 KB
const TimeSleep = 2

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID             string
	ServerAddress  string
	LoopAmount     int
	LoopPeriod     time.Duration
	BatchMaxAmount int
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

func (c *Client) StartClientLoop() {
	err := c.createClientSocket()
	if err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	c.processDataset()
	c.cleanupConnection()
	c.pollWinners()
}

func (c *Client) SetupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Infof("action: shutdown | result: in_progress | signal: SIGTERM | client_id: %v", c.config.ID)
		if c.conn != nil {
			c.conn.Close()
		}
		os.Exit(0)
	}()
}

func (c *Client) processDataset() {
	filePath := fmt.Sprintf("agency-%s.csv", c.config.ID)
	file, err := os.Open(filePath)
	if err != nil {
		log.Criticalf("action: open_dataset | result: fail | error: %v", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	batcher := NewBatcher(c.conn, c.config.ID, c.config.BatchMaxAmount, MaxSizeBatch)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 5 {
			continue // decido omitir bets incompletas
		}

		bet, err := protocol.NewBet(c.config.ID, record[0], record[1], record[2], record[3], record[4])
		if err != nil {
			log.Errorf("action: parse_bet | result: fail | error: %v", err)
			break
		}

		err = batcher.Add(bet)
		if err != nil {
			log.Errorf("action: batching | result: fail | error: %v", err)
			break
		}
	}
	err = batcher.Flush()
	if err != nil {
		log.Errorf("action: flush_batch | result: fail | error: %v", err)
	}
	err = protocol.SendFin(c.conn)
	if err != nil {
		log.Errorf("action: send_fin | result: fail | error: %v", err)
	}
}

func (c *Client) cleanupConnection() {
	c.conn.Close()
	log.Infof("action: close_resource | result: success | resource: client_socket | client_id: %v", c.config.ID)
}

// Poll to server asking for the winners
func (c *Client) pollWinners() {
	for {
		time.Sleep(TimeSleep * time.Second)

		err := c.createClientSocket()
		if err != nil {
			continue
		}

		_ = protocol.SendQuery(c.conn, c.config.ID)

		cmd, data, err := protocol.ReadCommand(c.conn)

		c.cleanupConnection()

		if err != nil {
			continue
		}

		if cmd == protocol.CmdWait {
			continue
		} else if cmd == protocol.CmdWinners {
			winnersStr := string(data)
			cantGanadores := 0
			if winnersStr != "" {
				cantGanadores = len(strings.Split(winnersStr, protocol.WinnersSeparator))
			}
			log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", cantGanadores)
			break
		}
	}
}
