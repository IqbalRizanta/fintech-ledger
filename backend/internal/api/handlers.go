package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/portfolio/fintech-ledger/internal/idempotency"
	"github.com/portfolio/fintech-ledger/internal/ledger"
	"github.com/portfolio/fintech-ledger/internal/models"
)

type Server struct {
	ledger      *ledger.Ledger
	idempotency *idempotency.Store
	mux         *http.ServeMux
}

func NewServer(l *ledger.Ledger, idemp *idempotency.Store) *Server {
	srv := &Server{
		ledger:      l,
		idempotency: idemp,
		mux:         http.NewServeMux(),
	}
	srv.routes()
	srv.seedInitialData()
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS Middleware
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/v1/accounts", s.handleAccounts)
	s.mux.HandleFunc("/api/v1/transactions", s.handleTransactions)
	s.mux.HandleFunc("/api/v1/transfer", s.handleSimpleTransfer)
	s.mux.HandleFunc("/api/v1/balance-sheet", s.handleBalanceSheet)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) seedInitialData() {
	// 1. Create standard ledger accounts
	bankReserve := s.ledger.CreateAccount("Bank Settlement Reserve", models.AccountAsset, "USD", "Central Liquidity Asset")
	aliceWallet := s.ledger.CreateAccount("Alice User Wallet", models.AccountLiability, "USD", "Customer Deposit Account")
	bobWallet := s.ledger.CreateAccount("Bob Merchant Wallet", models.AccountLiability, "USD", "Merchant Settlement Account")
	platformRevenue := s.ledger.CreateAccount("Platform Fee Revenue", models.AccountRevenue, "USD", "Transaction Interchange Revenue")
	_ = platformRevenue

	// 2. Initial funding transaction: Deposit $15,000 to Bank Reserve and credit Alice ($10,000) & Bob ($5,000)
	initialFundingDrafts := []ledger.EntryDraft{
		{AccountID: bankReserve.ID, Direction: models.DirectionDebit, Amount: 1500000}, // +$15,000 Asset
		{AccountID: aliceWallet.ID, Direction: models.DirectionCredit, Amount: 1000000}, // +$10,000 Liability (Alice)
		{AccountID: bobWallet.ID, Direction: models.DirectionCredit, Amount: 500000},   // +$5,000 Liability (Bob)
	}
	_, _ = s.ledger.RecordTransaction("seed_init_001", "Initial System Capitalization & Deposit Funding", initialFundingDrafts, "USD")
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		accounts := s.ledger.ListAccounts()
		writeJSON(w, http.StatusOK, map[string]any{"data": accounts})

	case http.MethodPost:
		var req struct {
			Name        string             `json:"name"`
			Type        models.AccountType `json:"type"`
			Currency    string             `json:"currency"`
			Description string             `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if req.Currency == "" {
			req.Currency = "USD"
		}
		acc := s.ledger.CreateAccount(req.Name, req.Type, req.Currency, req.Description)
		writeJSON(w, http.StatusCreated, acc)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil {
				limit = parsed
			}
		}
		txs := s.ledger.ListTransactions(limit)
		writeJSON(w, http.StatusOK, map[string]any{"data": txs})

	case http.MethodPost:
		idempKey := r.Header.Get("Idempotency-Key")

		// 1. Check Idempotency Cache
		if idempKey != "" {
			cachedTx, isCached, err := s.idempotency.LockKey(idempKey)
			if err != nil {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			if isCached {
				w.Header().Set("X-Idempotent-Replay", "true")
				writeJSON(w, http.StatusOK, cachedTx)
				return
			}
		}

		var req struct {
			Description string              `json:"description"`
			Currency    string              `json:"currency"`
			Entries     []ledger.EntryDraft `json:"entries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.idempotency.FailKey(idempKey)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if req.Currency == "" {
			req.Currency = "USD"
		}

		tx, err := s.ledger.RecordTransaction(idempKey, req.Description, req.Entries, req.Currency)
		if err != nil {
			s.idempotency.FailKey(idempKey)
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}

		s.idempotency.ReleaseKey(idempKey, tx)
		writeJSON(w, http.StatusCreated, tx)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSimpleTransfer provides a user-friendly transfer endpoint that compiles into double-entry ledger legs
func (s *Server) handleSimpleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idempKey := r.Header.Get("Idempotency-Key")
	if idempKey != "" {
		cachedTx, isCached, err := s.idempotency.LockKey(idempKey)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		if isCached {
			w.Header().Set("X-Idempotent-Replay", "true")
			writeJSON(w, http.StatusOK, cachedTx)
			return
		}
	}

	var req struct {
		FromAccountID string `json:"from_account_id"`
		ToAccountID   string `json:"to_account_id"`
		Amount        int64  `json:"amount"` // in cents
		FeeAmount     int64  `json:"fee_amount"` // optional platform fee
		Description   string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.idempotency.FailKey(idempKey)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if req.Amount <= 0 {
		s.idempotency.FailKey(idempKey)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount must be greater than zero"})
		return
	}

	// Find platform revenue account if fee is requested
	var feeAccountID string
	if req.FeeAmount > 0 {
		for _, acc := range s.ledger.ListAccounts() {
			if acc.Type == models.AccountRevenue {
				feeAccountID = acc.ID
				break
			}
		}
	}

	// Build balancing Double-Entry legs:
	// Debit Sender: Total (Amount + Fee)
	// Credit Receiver: Amount
	// Credit Platform Revenue: Fee
	entries := []ledger.EntryDraft{
		{AccountID: req.FromAccountID, Direction: models.DirectionDebit, Amount: req.Amount + req.FeeAmount},
		{AccountID: req.ToAccountID, Direction: models.DirectionCredit, Amount: req.Amount},
	}
	if req.FeeAmount > 0 && feeAccountID != "" {
		entries = append(entries, ledger.EntryDraft{
			AccountID: feeAccountID,
			Direction: models.DirectionCredit,
			Amount:    req.FeeAmount,
		})
	}

	desc := req.Description
	if desc == "" {
		desc = "Customer P2P Wallet Transfer"
	}

	tx, err := s.ledger.RecordTransaction(idempKey, desc, entries, "USD")
	if err != nil {
		s.idempotency.FailKey(idempKey)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	s.idempotency.ReleaseKey(idempKey, tx)
	writeJSON(w, http.StatusCreated, tx)
}

func (s *Server) handleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sheet := s.ledger.GetBalanceSheet()
	writeJSON(w, http.StatusOK, sheet)
}
