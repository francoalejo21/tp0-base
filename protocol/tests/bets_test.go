package tests

import (
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/protocol"
)

func TestBetInitMustKeepFields(t *testing.T) {
	expectedDate, _ := time.Parse("2006-01-02", "2000-12-20")
	b, err := protocol.NewBet("1", "first", "last", "10000000", "2000-12-20", "7500")

	if err != nil {
		t.Fatalf("Error al crear Bet: %v", err)
	}
	if b.Agency != 1 {
		t.Errorf("Expected 1, got %d", b.Agency)
	}
	if b.FirstName != "first" {
		t.Errorf("Expected first, got %s", b.FirstName)
	}
	if b.LastName != "last" {
		t.Errorf("Expected last, got %s", b.LastName)
	}
	if b.Document != "10000000" {
		t.Errorf("Expected 10000000, got %s", b.Document)
	}
	if !b.Birthdate.Equal(expectedDate) {
		t.Errorf("Fecha no coincide")
	}
	if b.Number != 7500 {
		t.Errorf("Expected 7500, got %d", b.Number)
	}
}

func TestHasWonWithWinnerNumberMustBeTrue(t *testing.T) {
	b1, _ := protocol.NewBet("1", "first", "last", "10000000", "2000-12-20", strconv.Itoa(protocol.LOTTERY_WINNER_NUMBER))
	if !protocol.HasWon(b1) {
		t.Error("Debería haber ganado")
	}
}

func TestHasWonWithWinnerNumberMustBeFalse(t *testing.T) {
	b2, _ := protocol.NewBet("1", "first", "last", "10000000", "2000-12-20", strconv.Itoa(protocol.LOTTERY_WINNER_NUMBER+1))
	if protocol.HasWon(b2) {
		t.Error("No debería haber ganado")
	}
}

func TestStoreAndLoadBetsKeepsFieldsData(t *testing.T) {
	setup(t)
	b, _ := protocol.NewBet("1", "first", "last", "10000000", "2000-12-20", "7500")
	toStore := []protocol.Bet{b}

	protocol.StoreBets(toStore)
	channelBets, errCh := protocol.LoadBets()
	fromLoad, err := collectBets(channelBets, errCh)
	if err != nil {
		t.Fatalf("Error cargando bets: %v", err)
	}
	if len(fromLoad) != 1 {
		t.Fatalf("Esperaba 1 apuesta, obtuve %d", len(fromLoad))
	}
	assertEqualBets(t, toStore[0], fromLoad[0])
}

func TestStoreAndLoadBetsKeepsRegistryOrder(t *testing.T) {
	setup(t)
	b0, _ := protocol.NewBet("0", "first_0", "last_0", "10000000", "2000-12-20", "7500")
	b1, _ := protocol.NewBet("1", "first_1", "last_1", "10000001", "2000-12-21", "7501")
	toStore := []protocol.Bet{b0, b1}

	protocol.StoreBets(toStore)
	channelBets, errCh := protocol.LoadBets()
	fromLoad, err := collectBets(channelBets, errCh)
	if err != nil {
		t.Fatalf("Error cargando bets: %v", err)
	}
	if len(fromLoad) != 2 {
		t.Fatalf("Esperaba 2 apuestas, obtuve %d", len(fromLoad))
	}
	assertEqualBets(t, toStore[0], fromLoad[0])
	assertEqualBets(t, toStore[1], fromLoad[1])
}

// Funciones auxiliares

// Auxiliar para limpiar el archivo antes y después de cada test
func setup(t *testing.T) {
	os.Remove(protocol.STORAGE_FILEPATH)
	t.Cleanup(func() {
		os.Remove(protocol.STORAGE_FILEPATH)
	})
}

// Función auxiliar para comparar structs
func assertEqualBets(t *testing.T, b1, b2 protocol.Bet) {
	if !reflect.DeepEqual(b1, b2) {
		t.Errorf("Las apuestas no son iguales: \n%+v\n%+v", b1, b2)
	}
}

// Función auxiliar para leer del channel todas las bets y convertirlas en una lista
func collectBets(stream <-chan protocol.Bet, errCh <-chan error) ([]protocol.Bet, error) {
	var bets []protocol.Bet

	for bet := range stream {
		bets = append(bets, bet)
	}

	// leer error al final
	if err := <-errCh; err != nil {
		return nil, err
	}

	return bets, nil
}
