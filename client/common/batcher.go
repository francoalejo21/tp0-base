package common

import (
	"fmt"
	"net"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"
)

const separatorSize = 1 // size in bytes of  '\n'

// Batcher encapsulates the logic to accumulate bets and send them in batches
// respecting the configured limits of the maximum amount of bets defined
// and the maximum size in bytes that the batch can have
type Batcher struct {
	conn      net.Conn
	clientID  string
	maxAmount int
	maxSize   int

	bets  []protocol.Bet
	bytes int
}

func NewBatcher(conn net.Conn, clientID string, maxAmount, maxSize int) *Batcher {
	return &Batcher{
		conn:      conn,
		clientID:  clientID,
		maxAmount: maxAmount,
		maxSize:   maxSize,
		bets:      make([]protocol.Bet, 0, maxAmount),
		bytes:     0,
	}
}

// Add includes a new bet in the current lot.
// If limits are reached, automatically submit the batch before adding the new one.
func (b *Batcher) Add(bet protocol.Bet) error {
	betSize := protocol.BetSerializedSize(bet)
	incomingSize := b.calculateIncomingSize(betSize)

	if !b.hasCapacityFor(incomingSize) {
		err := b.Flush()
		if err != nil {
			return err
		}
		incomingSize = b.calculateIncomingSize(betSize)
	}

	b.appendBet(bet, incomingSize)
	return nil
}

// Flush the accumulated bets to be sent and waits for confirmation from the server.
func (b *Batcher) Flush() error {
	if b.isEmpty() {
		return nil
	}

	err := b.sendCurrentBatch()
	if err != nil {
		return err
	}

	err = b.waitForConfirmation()
	if err != nil {
		return err
	}

	b.logSuccess()
	b.reset()

	return nil
}

// determines the actual size that the bet will occupy in the lot,
// adding the size of the separator if there are already previous elements.
func (b *Batcher) calculateIncomingSize(betSize int) int {
	if b.isEmpty() {
		return betSize
	}
	return betSize + separatorSize
}

// Checks if the lot can accept a new bet
// without exceeding the configured quantity or size limits.
func (b *Batcher) hasCapacityFor(incomingSize int) bool {
	if len(b.bets) >= b.maxAmount {
		return false
	}

	if (b.bytes + incomingSize) > b.maxSize {
		return false
	}

	return true
}

func (b *Batcher) isEmpty() bool {
	return len(b.bets) == 0
}

func (b *Batcher) appendBet(bet protocol.Bet, incomingSize int) {
	b.bets = append(b.bets, bet)
	b.bytes += incomingSize
}

func (b *Batcher) sendCurrentBatch() error {
	if err := protocol.SendBatch(b.conn, b.bets); err != nil {
		return fmt.Errorf("send_batch falló para el cliente %s: %w", b.clientID, err)
	}
	return nil
}

func (b *Batcher) waitForConfirmation() error {
	ok, err := protocol.ReceiveBatchConfirmation(b.conn)
	if err != nil {
		return fmt.Errorf("receive_confirmation falló para el cliente %s: %w", b.clientID, err)
	}
	if !ok {
		return fmt.Errorf("el servidor rechazó el lote del cliente %s", b.clientID)
	}
	return nil
}

func (b *Batcher) logSuccess() {
	log.Infof("action: batch_enviado | result: success | client_id: %v | cantidad: %v", b.clientID, len(b.bets))
}

// clears the internal state to start accumulating the next batch.
func (b *Batcher) reset() {
	b.bets = b.bets[:0]
	b.bytes = 0
}
