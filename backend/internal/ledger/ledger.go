package ledger

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/portfolio/fintech-ledger/internal/models"
)

var (
	ErrAccountNotFound       = errors.New("account not found")
	ErrUnbalancedTransaction = errors.New("unbalanced transaction: total debits must equal total credits")
	ErrInsufficientFunds     = errors.New("insufficient funds in source account")
	ErrInvalidAmount         = errors.New("amount must be strictly positive")
	ErrEmptyEntries          = errors.New("transaction must contain at least 2 entry legs")
)

type EntryDraft struct {
	AccountID string                `json:"account_id"`
	Direction models.EntryDirection `json:"direction"`
	Amount    int64                 `json:"amount"`
}

type Ledger struct {
	mu           sync.RWMutex
	accounts     map[string]*models.Account
	transactions map[string]*models.Transaction
	entries      []models.Entry
	txOrder      []string
}

func NewLedger() *Ledger {
	return &Ledger{
		accounts:     make(map[string]*models.Account),
		transactions: make(map[string]*models.Transaction),
		entries:      make([]models.Entry, 0),
		txOrder:      make([]string, 0),
	}
}

func (l *Ledger) CreateAccount(name string, accType models.AccountType, currency string, desc string) *models.Account {
	l.mu.Lock()
	defer l.mu.Unlock()

	acc := &models.Account{
		ID:          uuid.New().String(),
		Name:        name,
		Type:        accType,
		Currency:    currency,
		Balance:     0,
		Description: desc,
		CreatedAt:   time.Now().UTC(),
	}
	l.accounts[acc.ID] = acc
	return acc
}

func (l *Ledger) GetAccount(id string) (*models.Account, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	acc, exists := l.accounts[id]
	if !exists {
		return nil, ErrAccountNotFound
	}
	// Return copy with updated balance
	accCopy := *acc
	accCopy.Balance = l.calculateAccountBalance(id, acc.Type)
	return &accCopy, nil
}

func (l *Ledger) ListAccounts() []*models.Account {
	l.mu.RLock()
	defer l.mu.RUnlock()

	res := make([]*models.Account, 0, len(l.accounts))
	for _, acc := range l.accounts {
		accCopy := *acc
		accCopy.Balance = l.calculateAccountBalance(acc.ID, acc.Type)
		res = append(res, &accCopy)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res
}

// RecordTransaction atomically validates and appends double-entry legs
func (l *Ledger) RecordTransaction(idempotencyKey, desc string, drafts []EntryDraft, currency string) (*models.Transaction, error) {
	if len(drafts) < 2 {
		return nil, ErrEmptyEntries
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var totalDebits int64
	var totalCredits int64

	// 1. Verify existence of all accounts and calculate total debits / credits
	for _, d := range drafts {
		if d.Amount <= 0 {
			return nil, ErrInvalidAmount
		}
		acc, exists := l.accounts[d.AccountID]
		if !exists {
			return nil, fmt.Errorf("%w: %s", ErrAccountNotFound, d.AccountID)
		}

		switch d.Direction {
		case models.DirectionDebit:
			totalDebits += d.Amount
			// If liability account (e.g. customer wallet) is debited (withdrawal/transfer), ensure no overdraft
			if acc.Type == models.AccountLiability {
				currBal := l.calculateAccountBalance(acc.ID, acc.Type)
				if currBal < d.Amount {
					return nil, fmt.Errorf("%w: account '%s' has balance %d, requested %d", ErrInsufficientFunds, acc.Name, currBal, d.Amount)
				}
			}
		case models.DirectionCredit:
			totalCredits += d.Amount
		default:
			return nil, fmt.Errorf("invalid entry direction: %s", d.Direction)
		}
	}

	// 2. Strict Double-Entry Equilibrium Check: Sum(Debits) == Sum(Credits)
	if totalDebits != totalCredits {
		return nil, fmt.Errorf("%w (total debits: %d != total credits: %d)", ErrUnbalancedTransaction, totalDebits, totalCredits)
	}

	now := time.Now().UTC()
	txID := uuid.New().String()
	createdEntries := make([]models.Entry, 0, len(drafts))

	// 3. Append-only ledger write
	for _, d := range drafts {
		entry := models.Entry{
			ID:            uuid.New().String(),
			TransactionID: txID,
			AccountID:     d.AccountID,
			Direction:     d.Direction,
			Amount:        d.Amount,
			Currency:      currency,
			CreatedAt:     now,
		}
		l.entries = append(l.entries, entry)
		createdEntries = append(createdEntries, entry)
	}

	tx := &models.Transaction{
		ID:             txID,
		IdempotencyKey: idempotencyKey,
		Description:    desc,
		Status:         models.StatusCommitted,
		TotalAmount:    totalDebits,
		Currency:       currency,
		Entries:        createdEntries,
		CreatedAt:      now,
	}

	l.transactions[txID] = tx
	l.txOrder = append(l.txOrder, txID)

	// Update cached account balances
	for _, d := range drafts {
		if acc, ok := l.accounts[d.AccountID]; ok {
			acc.Balance = l.calculateAccountBalance(acc.ID, acc.Type)
		}
	}

	return tx, nil
}

func (l *Ledger) ListTransactions(limit int) []*models.Transaction {
	l.mu.RLock()
	defer l.mu.RUnlock()

	total := len(l.txOrder)
	if limit <= 0 || limit > total {
		limit = total
	}

	res := make([]*models.Transaction, 0, limit)
	for i := total - 1; i >= total-limit && i >= 0; i-- {
		txID := l.txOrder[i]
		if tx, ok := l.transactions[txID]; ok {
			res = append(res, tx)
		}
	}
	return res
}

func (l *Ledger) GetBalanceSheet() models.BalanceSheet {
	l.mu.RLock()
	defer l.mu.RUnlock()

	sheet := models.BalanceSheet{}
	for _, acc := range l.accounts {
		bal := l.calculateAccountBalance(acc.ID, acc.Type)
		switch acc.Type {
		case models.AccountAsset:
			sheet.TotalAssets += bal
		case models.AccountLiability:
			sheet.TotalLiabilities += bal
		case models.AccountEquity:
			sheet.TotalEquity += bal
		case models.AccountRevenue:
			sheet.TotalRevenue += bal
		case models.AccountExpense:
			sheet.TotalExpenses += bal
		}
	}

	// Accounting Equation: Assets == Liabilities + Equity + (Revenue - Expenses)
	rightSide := sheet.TotalLiabilities + sheet.TotalEquity + (sheet.TotalRevenue - sheet.TotalExpenses)
	sheet.IsBalanced = (sheet.TotalAssets == rightSide)
	return sheet
}

func (l *Ledger) calculateAccountBalance(accountID string, accType models.AccountType) int64 {
	var debits int64
	var credits int64

	for _, e := range l.entries {
		if e.AccountID == accountID {
			if e.Direction == models.DirectionDebit {
				debits += e.Amount
			} else {
				credits += e.Amount
			}
		}
	}

	// Assets and Expenses are Debit-normal
	if accType == models.AccountAsset || accType == models.AccountExpense {
		return debits - credits
	}
	// Liabilities, Equity, and Revenue are Credit-normal
	return credits - debits
}
