package protocol

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const betSeparator = "\n"
const fieldSeparator = "|"
const expectedFields = 6

const confirmOK = "OK"
const confirmERR = "ERR"

// format: agency|firstname|lastname|document|birthdate|number
func serialize(b Bet) string {
	fields := []string{
		strconv.Itoa(b.Agency),
		b.FirstName,
		b.LastName,
		b.Document,
		b.Birthdate.Format(DATE_FORMAT),
		strconv.Itoa(b.Number),
	}
	return strings.Join(fields, fieldSeparator)
}

// deserialize decodes a string payload back into a Bet.
func deserialize(payload string) (Bet, error) {
	fields := strings.Split(string(payload), fieldSeparator)
	if len(fields) != expectedFields {
		return Bet{}, fmt.Errorf("expected %d fields, got %d", expectedFields, len(fields))
	}
	return NewBet(fields[0], fields[1], fields[2], fields[3], fields[4], fields[5])
}

func serializeBatch(bets []Bet) []byte {
	lines := make([]string, len(bets))
	for i, b := range bets {
		lines[i] = string(serialize(b))
	}
	return []byte(strings.Join(lines, betSeparator))
}

func deserializeBatch(payload []byte) ([]Bet, error) {
	lines := strings.Split(string(payload), betSeparator)
	bets := make([]Bet, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		bet, err := deserialize(line)
		if err != nil {
			return nil, fmt.Errorf("deserializing bet %q: %w", line, err)
		}
		bets = append(bets, bet)
	}
	return bets, nil
}

// SendBatch serializes a slice of Bets and sends them as a single length-prefixed frame.
func SendBatch(conn net.Conn, bets []Bet) error {
	return SendFrame(conn, serializeBatch(bets))
}

// ReceiveBatch reads a single frame and deserializes it as a slice of Bets.
func ReceiveBatch(conn net.Conn) ([]Bet, error) {
	payload, err := RecvFrame(conn)
	if err != nil {
		return nil, err
	}
	return deserializeBatch(payload)
}

// SendBatchConfirmation sends OK on success or ERR on failure.
func SendBatchConfirmation(conn net.Conn, success bool) error {
	if success {
		return SendFrame(conn, []byte(confirmOK))
	}
	return SendFrame(conn, []byte(confirmERR))
}

// ReceiveBatchConfirmation reads the server's response for a batch.
// Returns true on "OK", false on "ERR", and an error for anything unexpected.
func ReceiveBatchConfirmation(conn net.Conn) (bool, error) {
	payload, err := RecvFrame(conn)
	if err != nil {
		return false, err
	}
	switch string(payload) {
	case confirmOK:
		return true, nil
	case confirmERR:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected batch confirmation: %q", string(payload))
	}
}

// BetSerializedSize returns the number of bytes that bet would occupy inside a
// batch frame payload
func BetSerializedSize(b Bet) int {
	return len(serialize(b))
}
