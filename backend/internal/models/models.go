package models

import "time"

type AccountType string

const (
	AccountAsset     AccountType = "ASSET"
	AccountLiability AccountType = "LIABILITY"
	AccountEquity    AccountType = "EQUITY"
	AccountRevenue   AccountType = "REVENUE"
	AccountExpense   AccountType = "EXPENSE"
)

type EntryDirection string

const (
	DirectionDebit  EntryDirection = "DEBIT"
	DirectionCredit EntryDirection = "CREDIT"
)

type Account struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        AccountType `json:"type"`
	Currency    string      `json:"currency"`
	Balance     int64       `json:"balance"` // Computed balance in smallest unit
	Description string      `json:"description"`
	CreatedAt   time.Time   `json:"created_at"`
}

type Entry struct {
	ID            string         `json:"id"`
	TransactionID string         `json:"transaction_id"`
	AccountID     string         `json:"account_id"`
	Direction     EntryDirection `json:"direction"`
	Amount        int64          `json:"amount"` // Always positive integer
	Currency      string         `json:"currency"`
	CreatedAt     time.Time      `json:"created_at"`
}

type TransactionStatus string

const (
	StatusCommitted TransactionStatus = "COMMITTED"
	StatusRejected  TransactionStatus = "REJECTED"
)

type Transaction struct {
	ID             string            `json:"id"`
	IdempotencyKey string            `json:"idempotency_key"`
	Description    string            `json:"description"`
	Status         TransactionStatus `json:"status"`
	TotalAmount    int64             `json:"total_amount"`
	Currency       string            `json:"currency"`
	Entries        []Entry           `json:"entries"`
	CreatedAt      time.Time         `json:"created_at"`
}

type BalanceSheet struct {
	TotalAssets      int64 `json:"total_assets"`
	TotalLiabilities int64 `json:"total_liabilities"`
	TotalEquity      int64 `json:"total_equity"`
	TotalRevenue     int64 `json:"total_revenue"`
	TotalExpenses    int64 `json:"total_expenses"`
	IsBalanced       bool  `json:"is_balanced"`
}
