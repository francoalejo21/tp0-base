package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// 4 bytes to identify how size is the message to read
const HEADER_SIZE uint32 = 4

// SendAll sends all bytes to the connection
func sendAll(conn net.Conn, data []byte) error {
	total := len(data)
	sent := 0
	for sent < total {
		n, err := conn.Write(data[sent:])
		if err != nil {
			return fmt.Errorf("short-write: sent %d/%d bytes: %w", sent, total, err)
		}
		sent += n
	}
	return nil
}

// RecvAll reads exactly n bytes from the connection
func recvAll(conn net.Conn, n uint32) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, fmt.Errorf("short-read: expected %d bytes: %w", n, err)
	}
	return buf, nil
}

// SendFrame sends a length-prefixed frame: [4-byte big-endian length][payload]
func SendFrame(conn net.Conn, payload []byte) error {
	header := make([]byte, HEADER_SIZE)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))

	frame := append(header, payload...)
	return sendAll(conn, frame)
}

// RecvFrame reads a length-prefixed frame and returns the payload.
func RecvFrame(conn net.Conn) ([]byte, error) {
	header, err := recvAll(conn, HEADER_SIZE)
	if err != nil {
		return nil, fmt.Errorf("error reading frame header: %w", err)
	}

	msgLen := binary.BigEndian.Uint32(header)
	if msgLen == 0 {
		return nil, fmt.Errorf("invalid frame length: %d", msgLen)
	}

	return recvAll(conn, msgLen)
}
