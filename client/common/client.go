package common

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

const MaxSizeBatch = 8 * 1024 // 8 KB

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

	filePath := fmt.Sprintf("agency-%s.csv", c.config.ID)
	file, err := os.Open(filePath)
	if err != nil {
		log.Criticalf("action: open_dataset | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	defer file.Close()

	if err := c.createClientSocket(); err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	defer func() {
		c.conn.Close()
		log.Infof("action: close_resource | result: success | resource: client_socket | client_id: %v", c.config.ID)
	}()

	reader := csv.NewReader(file)

	batch := make([]protocol.Bet, 0, c.config.BatchMaxAmount)
	batchBytes := 0
	const separatorSize = 1

	flushBatch := func() bool {
		if len(batch) == 0 {
			return true
		}

		if err := protocol.SendBatch(c.conn, batch); err != nil {
			log.Errorf("action: send_batch | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return false
		}

		ok, err := protocol.ReceiveBatchConfirmation(c.conn)
		if err != nil {
			log.Errorf("action: receive_confirmation | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return false
		}
		if !ok {
			log.Errorf("action: send_batch | result: fail | client_id: %v | server rejected batch", c.config.ID)
			return false
		}

		log.Infof("action: batch_enviado | result: success | client_id: %v | cantidad: %v", c.config.ID, len(batch))
		batch = batch[:0]
		batchBytes = 0
		return true
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("action: read_dataset | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}

		// CSV columns: firstname, lastname, document, birthdate, number
		if len(record) < 5 {
			log.Errorf("action: parse_record | result: fail | client_id: %v | error: expected 5 fields, got %d", c.config.ID, len(record))
			return
		}

		bet, err := protocol.NewBet(
			c.config.ID, // agency
			record[0],   // firstname
			record[1],   // lastname
			record[2],   // document
			record[3],   // birthdate
			record[4],   // number
		)
		if err != nil {
			log.Errorf("action: parse_bet | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}

		// How many bytes does this bet contribute to the payload?
		// First bet in a batch: just its own serialized length.
		// Every subsequent bet: its length + 1 byte for the '\n' separator.
		betSize := protocol.BetSerializedSize(bet)
		incomingSize := betSize
		if len(batch) > 0 {
			incomingSize += separatorSize
		}

		// Flush before appending if either limit would be breached.
		if len(batch) >= c.config.BatchMaxAmount || (len(batch) > 0 && batchBytes+incomingSize > MaxSizeBatch) {
			if !flushBatch() {
				return
			}
			// Batch is now empty: no separator cost for the first record.
			incomingSize = betSize
		}

		batch = append(batch, bet)
		batchBytes += incomingSize
	}

	// Flush any remaining bets.
	flushBatch()
}
