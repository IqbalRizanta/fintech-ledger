# Financial Core Ledger & Idempotent Payment Orchestrator

An enterprise-grade financial ledger and payment orchestrator built in **Golang**, with a modern double-entry accounting console built in **Next.js 14 / React**.

Designed to handle mission-critical financial transactions with **double-entry zero-sum guarantees**, **strict idempotency to prevent double-spending**, and an **immutable audit trail**.

---

## 🌟 Key Features

* **Double-Entry Bookkeeping Engine**: Strictly enforces `Sum(Debits) == Sum(Credits)`. Unbalanced transactions are rejected atomically before mutating balances.
* **Strict Idempotency Protection**: Evaluates `Idempotency-Key` headers using mutex-guarded state locks. Prevents duplicate debit operations from accidental retries or network double-submits.
* **Overdraft & Balance Invariants**: Automatic check prevents liability accounts (customer wallets) from spending more than available settled funds.
* **Immutable General Journal**: Append-only transaction ledger records. Account balances are dynamic projections over committed entries.
* **Next.js Financial Dashboard**: Live balance sheet verification, account wallet balances, transfer simulator with a **"Double-Click Glitch" test**, and full general journal view.

---

## 📐 Architecture Overview

```text
               POST /api/v1/transfer
           (Idempotency-Key: req_abc123)
                         │
                         ▼
┌────────────────────────────────────────────────────────┐
│            Financial Ledger Core Engine                │
│                                                        │
│  1. Idempotency Lock Check                             │
│     ├── Key In Progress ─────► HTTP 409 Conflict       │
│     ├── Key Completed ───────► HTTP 200 (Replay Cache) │
│     └── New Key ─────────────┐                         │
│                              ▼                         │
│  2. Double-Entry Balancing Verification                │
│     ├── Sum(Debit) != Sum(Credit) ──► HTTP 422 Reject │
│     └── Balanced ────────────┐                         │
│                              ▼                         │
│  3. Overdraft & Settlement Invariant Evaluation        │
│     ├── Source Balance < Debit ─────► HTTP 422 Reject │
│     └── Sufficient Funds ────┐                         │
│                              ▼                         │
│  4. Atomic Append to Immutable General Journal         │
│     - Leg 1: DEBIT Source Wallet ($255.00)             │
│     - Leg 2: CREDIT Destination Wallet ($250.00)       │
│     - Leg 3: CREDIT Platform Revenue ($5.00)           │
│     ─────────────────────────────────────────          │
│     Sum = $0.00 (Equilibrium Preserved)                │
│                                                        │
│  5. Release Idempotency Lock with Committed Transaction│
└──────────────────────────────┬─────────────────────────┘
                               │
                               ▼
               HTTP 201 Created (Committed)
```

---

## 🚀 Getting Started

### 1. Prerequisites
* **Go** 1.22+
* **Node.js** 18+ (for Next.js frontend)

### 2. Run the Golang Backend
```bash
cd backend
go run ./cmd/server/main.go
```
The server will start on `http://localhost:8083`. Initial accounts (Bank Reserve, Alice Wallet, Bob Wallet, Platform Fees) will be seeded automatically!

### 3. Run the Next.js Financial Dashboard
```bash
cd frontend
npm install
npm run dev
```
Open [http://localhost:3003](http://localhost:3003) in your browser.

*(Optional)* You can also double click `frontend/public/standalone.html` to view the financial console immediately without npm!

---

## 🧪 Automated Tests

Run backend unit tests verifying double-entry rejection and idempotency protection:
```bash
cd backend
go test -v ./...
```

---

## 📂 Project Structure

```text
04-fintech-ledger/
├── backend/
│   ├── cmd/server/main.go          # Entrypoint & startup banner
│   ├── internal/
│   │   ├── api/                    # REST handlers & double-entry transfer router
│   │   ├── idempotency/            # Thread-safe mutex lock & replay store
│   │   ├── ledger/                 # Double-entry engine, invariants, unit tests
│   │   └── models/                 # Account, Entry, Transaction domain types
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── app/
│   │   ├── page.tsx                # Next.js balance sheet & transfer simulator
│   │   ├── layout.tsx
│   │   └── globals.css
│   ├── public/standalone.html      # Zero-dependency instant preview UI
│   ├── package.json
│   └── tailwind.config.js
└── README.md
```
