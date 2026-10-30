package idempotency

import (
	"errors"
	"sync"
	"time"

	"github.com/portfolio/fintech-ledger/internal/models"
)

var (
	ErrConcurrentRequest = errors.New("concurrent request in progress for this idempotency key")
)

type ExecutionState string

const (
	StateProcessing ExecutionState = "PROCESSING"
	StateCompleted  ExecutionState = "COMPLETED"
)

type Record struct {
	State       ExecutionState
	Transaction *models.Transaction
	CreatedAt   time.Time
}

type Store struct {
	mu      sync.RWMutex
	records map[string]*Record
}

func NewStore() *Store {
	return &Store{
		records: make(map[string]*Record),
	}
}

// LockKey attempts to claim an idempotency key. Returns existing transaction if already completed.
func (s *Store) LockKey(key string) (*models.Transaction, bool, error) {
	if key == "" {
		return nil, false, nil // No idempotency key provided
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, exists := s.records[key]
	if exists {
		if rec.State == StateProcessing {
			return nil, false, ErrConcurrentRequest
		}
		// Return already completed transaction
		return rec.Transaction, true, nil
	}

	// Claim key as processing
	s.records[key] = &Record{
		State:     StateProcessing,
		CreatedAt: time.Now().UTC(),
	}
	return nil, false, nil
}

func (s *Store) ReleaseKey(key string, tx *models.Transaction) {
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if rec, ok := s.records[key]; ok {
		rec.State = StateCompleted
		rec.Transaction = tx
	}
}

func (s *Store) FailKey(key string) {
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, key)
}
