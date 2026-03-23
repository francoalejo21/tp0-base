package protocol

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"
)

// Bets storage location.
const STORAGE_FILEPATH = "./bets.csv"

// Simulated winner number in the lottery contest.
const LOTTERY_WINNER_NUMBER = 7574

const DATE_FORMAT = "2006-01-02"

// A lottery bet registry
type Bet struct {
	Agency    int
	FirstName string
	LastName  string
	Document  string
	Birthdate time.Time
	Number    int
}

// Return A lottery bet registry.
func NewBet(agencyStr, firstName, lastName, document, birthdateStr, numberStr string) (Bet, error) {
	// agency must be passed with integer format.
	// birthdate must be passed with format: 'YYYY-MM-DD'.
	// number must be passed with integer format.

	agency, err := strconv.Atoi(agencyStr)
	if err != nil {
		return Bet{}, err
	}

	bDate, err := time.Parse(DATE_FORMAT, birthdateStr)
	if err != nil {
		return Bet{}, err
	}

	num, err := strconv.Atoi(numberStr)
	if err != nil {
		return Bet{}, err
	}

	return Bet{
		Agency:    agency,
		FirstName: firstName,
		LastName:  lastName,
		Document:  document,
		Birthdate: bDate,
		Number:    num,
	}, nil
}

// Checks whether a bet won the prize or not.
func HasWon(bet Bet) bool {
	return bet.Number == LOTTERY_WINNER_NUMBER
}

// Persist the information of each bet in the STORAGE_FILEPATH file.
// Not thread-safe/process-safe.

func StoreBets(bets []Bet) error {
	file, err := os.OpenFile(STORAGE_FILEPATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, bet := range bets {
		record := []string{
			strconv.Itoa(bet.Agency),
			bet.FirstName,
			bet.LastName,
			bet.Document,
			bet.Birthdate.Format(DATE_FORMAT),
			strconv.Itoa(bet.Number),
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

// Loads the information all the bets in the STORAGE_FILEPATH file.
func LoadBets() (<-chan Bet, <-chan error) {
	// forma que se me ocurrió para no tener que cargar todo en memoria, usando channels
	out := make(chan Bet)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		file, err := os.Open(STORAGE_FILEPATH)
		if err != nil {
			errCh <- err
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)

		for {
			record, err := reader.Read()
			if err != nil {
				if err.Error() == "EOF" {
					return
				}
				errCh <- err
				return
			}

			bet, err := NewBet(
				record[0],
				record[1],
				record[2],
				record[3],
				record[4],
				record[5],
			)
			if err != nil {
				errCh <- err
				return
			}

			out <- bet // Vamos mandando de a un bet esperando que el otro extremo del channel lea y procese para que pueda mandar el siguiente
		}
	}()

	return out, errCh
}
