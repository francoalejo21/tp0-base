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
const WinnersSeparator = ","

// MessagesTypes
type CmdType string

const (
	CmdBatch   CmdType = "BATCH"
	CmdFin     CmdType = "FIN"
	CmdQuery   CmdType = "QUERY"
	CmdWait    CmdType = "WAIT"
	CmdWinners CmdType = "WINNERS"
)
const ConfirmOK = "OK"
const ConfirmERR = "ERR"

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

func SerializeBatch(bets []Bet) []byte {
	lines := make([]string, len(bets))
	for i, b := range bets {
		lines[i] = string(serialize(b))
	}
	return []byte(strings.Join(lines, betSeparator))
}

func DeserializeBatch(payload []byte) ([]Bet, error) {
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

// SendBatchConfirmation sends OK on success or ERR on failure.
func SendBatchConfirmation(conn net.Conn, success bool) error {
	if success {
		return SendFrame(conn, []byte(ConfirmOK))
	}
	return SendFrame(conn, []byte(ConfirmERR))
}

// ReceiveBatchConfirmation reads the server's response for a batch.
// Returns true on "OK", false on "ERR", and an error for anything unexpected.
func ReceiveBatchConfirmation(conn net.Conn) (bool, error) {
	payload, err := RecvFrame(conn)
	if err != nil {
		return false, err
	}
	switch string(payload) {
	case ConfirmOK:
		return true, nil
	case ConfirmERR:
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

func SendBatch(conn net.Conn, bets []Bet) error {
	serializedBets := SerializeBatch(bets)
	msg := fmt.Sprintf("%s%s%s", CmdBatch, fieldSeparator, string(serializedBets))
	return SendFrame(conn, []byte(msg))
}

func SendFin(conn net.Conn) error {
	return SendFrame(conn, []byte(CmdFin))
}

func SendQuery(conn net.Conn, agencyID string) error {
	msg := fmt.Sprintf("%s%s%s", CmdQuery, fieldSeparator, agencyID)
	return SendFrame(conn, []byte(msg))
}

func SendWait(conn net.Conn) error {
	return SendFrame(conn, []byte(CmdWait))
}

func SendWinners(conn net.Conn, winners []string) error {
	msg := fmt.Sprintf("%s%s%s", CmdWinners, fieldSeparator, strings.Join(winners, WinnersSeparator))
	return SendFrame(conn, []byte(msg))
}

func ReadCommand(conn net.Conn) (CmdType, []byte, error) {
	payload, err := RecvFrame(conn)
	if err != nil {
		return "", nil, err
	}

	parts := strings.SplitN(string(payload), fieldSeparator, 2)

	cmd := CmdType(parts[0])

	if len(parts) == 1 {
		return cmd, nil, nil
	}
	return cmd, []byte(parts[1]), nil
}
