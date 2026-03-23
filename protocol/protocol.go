package protocol

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const fieldSeparator = "|"
const expectedFields = 6

// first approach format: agency|firstname|lastname|document|birthdate|number
func serialize(b Bet) []byte {
	fields := []string{
		strconv.Itoa(b.Agency),
		b.FirstName,
		b.LastName,
		b.Document,
		b.Birthdate.Format(DATE_FORMAT),
		strconv.Itoa(b.Number),
	}
	return []byte(strings.Join(fields, fieldSeparator))
}

// deserialize decodes a raw payload back into a Bet.
func deserialize(payload []byte) (Bet, error) {
	fields := strings.Split(string(payload), fieldSeparator)
	if len(fields) != expectedFields {
		return Bet{}, fmt.Errorf("expected %d fields, got %d", expectedFields, len(fields))
	}
	return NewBet(fields[0], fields[1], fields[2], fields[3], fields[4], fields[5])
}

// SendBet serializes a Bet and sends it as a length-prefixed frame.
// This is where it communicates with the transport layer to take care of the sending
func SendBet(conn net.Conn, bet Bet) error {
	return SendFrame(conn, serialize(bet))
}

// ReceiveBet reads a frame and deserializes it as a Bet.
func ReceiveBet(conn net.Conn) (Bet, error) {
	payload, err := RecvFrame(conn)
	if err != nil {
		return Bet{}, err
	}
	return deserialize(payload)
}

// SendConfirmation sends an OK acknowledgement.
func SendConfirmation(conn net.Conn) error {
	return SendFrame(conn, []byte("OK"))
}

// ReceiveConfirmation reads and validates the server's acknowledgement.
func ReceiveConfirmation(conn net.Conn) error {
	payload, err := RecvFrame(conn)
	if err != nil {
		return err
	}
	if string(payload) != "OK" {
		return fmt.Errorf("unexpected confirmation: %q", string(payload))
	}
	return nil
}
