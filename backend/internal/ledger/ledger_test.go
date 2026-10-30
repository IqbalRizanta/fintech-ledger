package ledger

import (
	"testing"

	"github.com/portfolio/fintech-ledger/internal/idempotency"
	"github.com/portfolio/fintech-ledger/internal/models"
)

func TestUnbalancedTransactionRejection(t *testing.T) {
	l := NewLedger()
	acc1 := l.CreateAccount("Bank Asset", models.AccountAsset, "USD", "")
	acc2 := l.CreateAccount("Customer Liability", models.AccountLiability, "USD", "")

	// Debit 100, Credit 50 -> Unbalanced!
	unbalancedDrafts := []EntryDraft{
		{AccountID: acc1.ID, Direction: models.DirectionDebit, Amount: 100},
		{AccountID: acc2.ID, Direction: models.DirectionCredit, Amount: 50},
	}

	_, err := l.RecordTransaction("tx_unbalanced", "Unbalanced Test", unbalancedDrafts, "USD")
	if err == nil {
		t.Fatal("expected error for unbalanced transaction, got nil")
	}
}

func TestBalancedTransactionAndBalanceSheet(t *testing.T) {
	l := NewLedger()
	bank := l.CreateAccount("Bank Reserve", models.AccountAsset, "USD", "")
	alice := l.CreateAccount("Alice", models.AccountLiability, "USD", "")

	// Balanced transaction: Debit Bank Reserve $500, Credit Alice $500
	drafts := []EntryDraft{
		{AccountID: bank.ID, Direction: models.DirectionDebit, Amount: 50000},
		{AccountID: alice.ID, Direction: models.DirectionCredit, Amount: 50000},
	}

	tx, err := l.RecordTransaction("tx_001", "Deposit Funding", drafts, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Status != models.StatusCommitted {
		t.Fatalf("expected status COMMITTED, got %s", tx.Status)
	}

	// Verify Account Balances
	bankAcc, _ := l.GetAccount(bank.ID)
	aliceAcc, _ := l.GetAccount(alice.ID)

	if bankAcc.Balance != 50000 {
		t.Fatalf("expected bank balance 50000, got %d", bankAcc.Balance)
	}
	if aliceAcc.Balance != 50000 {
		t.Fatalf("expected alice balance 50000, got %d", aliceAcc.Balance)
	}

	// Verify Balance Sheet Equilibrium
	sheet := l.GetBalanceSheet()
	if !sheet.IsBalanced {
		t.Fatal("expected balance sheet to be balanced")
	}
}

func TestIdempotencyPreventionOfDoubleSpend(t *testing.T) {
	l := NewLedger()
	idemp := idempotency.NewStore()

	acc1 := l.CreateAccount("Bank", models.AccountAsset, "USD", "")
	acc2 := l.CreateAccount("Bob", models.AccountLiability, "USD", "")

	key := "idem_req_abc123"

	// 1st Call: Locks and completes
	_, isCached, err := idemp.LockKey(key)
	if err != nil || isCached {
		t.Fatalf("first call should not be cached: err=%v", err)
	}

	drafts := []EntryDraft{
		{AccountID: acc1.ID, Direction: models.DirectionDebit, Amount: 1000},
		{AccountID: acc2.ID, Direction: models.DirectionCredit, Amount: 1000},
	}
	tx, err := l.RecordTransaction(key, "First Charge", drafts, "USD")
	if err != nil {
		t.Fatalf("unexpected tx error: %v", err)
	}
	idemp.ReleaseKey(key, tx)

	// 2nd Call with SAME Idempotency-Key: Must return cached without double mutation!
	cachedTx, isCached2, err2 := idemp.LockKey(key)
	if err2 != nil {
		t.Fatalf("unexpected lock error on repeat: %v", err2)
	}
	if !isCached2 {
		t.Fatal("expected second call to be cached")
	}
	if cachedTx.ID != tx.ID {
		t.Fatalf("expected same transaction ID %s, got %s", tx.ID, cachedTx.ID)
	}

	// Verify Bank balance was NOT doubled
	bankAcc, _ := l.GetAccount(acc1.ID)
	if bankAcc.Balance != 1000 {
		t.Fatalf("expected balance to remain 1000, but got %d (Double spend happened!)", bankAcc.Balance)
	}
}
