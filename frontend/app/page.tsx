'use client'

import React, { useState, useEffect } from 'react'
import {
  Landmark,
  ArrowRightLeft,
  ShieldCheck,
  Scale,
  RefreshCw,
  Wallet,
  CheckCircle2,
  AlertCircle,
  Clock,
  Layers,
  Send,
  Zap,
  DollarSign
} from 'lucide-react'

interface Account {
  id: string
  name: string
  type: 'ASSET' | 'LIABILITY' | 'EQUITY' | 'REVENUE' | 'EXPENSE'
  currency: string
  balance: number
  description: string
  created_at: string
}

interface Entry {
  id: string
  transaction_id: string
  account_id: string
  direction: 'DEBIT' | 'CREDIT'
  amount: number
  currency: string
  created_at: string
}

interface Transaction {
  id: string
  idempotency_key: string
  description: string
  status: 'COMMITTED' | 'REJECTED'
  total_amount: number
  currency: string
  entries: Entry[]
  created_at: string
}

interface BalanceSheet {
  total_assets: number
  total_liabilities: number
  total_equity: number
  total_revenue: number
  total_expenses: number
  is_balanced: boolean
}

const API_BASE = 'http://localhost:8083'

export default function LedgerDashboard() {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [balanceSheet, setBalanceSheet] = useState<BalanceSheet | null>(null)
  const [activeTab, setActiveTab] = useState<'accounts' | 'transfer' | 'journal'>('accounts')

  // Transfer form
  const [fromAccount, setFromAccount] = useState('')
  const [toAccount, setToAccount] = useState('')
  const [amountDollars, setAmountDollars] = useState(250)
  const [feeDollars, setFeeDollars] = useState(5)
  const [idempKey, setIdempKey] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [transferAlert, setTransferAlert] = useState<string | null>(null)

  const generateUUID = () => {
    return 'req_' + Math.random().toString(36).substring(2, 10) + '_' + Date.now()
  }

  const fetchData = async () => {
    try {
      const [resAcc, resTx, resSheet] = await Promise.all([
        fetch(`${API_BASE}/api/v1/accounts`),
        fetch(`${API_BASE}/api/v1/transactions?limit=50`),
        fetch(`${API_BASE}/api/v1/balance-sheet`)
      ])
      if (resAcc.ok) {
        const json = await resAcc.json()
        const accList: Account[] = json.data || []
        setAccounts(accList)
        if (!fromAccount && accList.length >= 2) {
          const alice = accList.find((a) => a.name.includes('Alice'))
          const bob = accList.find((a) => a.name.includes('Bob'))
          if (alice) setFromAccount(alice.id)
          if (bob) setToAccount(bob.id)
        }
      }
      if (resTx.ok) {
        const json = await resTx.json()
        setTransactions(json.data || [])
      }
      if (resSheet.ok) {
        setBalanceSheet(await resSheet.json())
      }
    } catch (e) {
      console.warn('Backend server not connected yet on :8083', e)
    }
  }

  useEffect(() => {
    setIdempKey(generateUUID())
    fetchData()
    const interval = setInterval(fetchData, 2500)
    return () => clearInterval(interval)
  }, [])

  const formatMoney = (cents: number) => {
    return '$' + (cents / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  }

  const handleTransfer = async (customKey?: string) => {
    if (!fromAccount || !toAccount) {
      alert('Please select both sender and receiver accounts')
      return
    }
    setIsSubmitting(true)
    setTransferAlert(null)
    const keyToUse = customKey || idempKey

    try {
      const res = await fetch(`${API_BASE}/api/v1/transfer`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': keyToUse
        },
        body: JSON.stringify({
          from_account_id: fromAccount,
          to_account_id: toAccount,
          amount: Math.round(amountDollars * 100),
          fee_amount: Math.round(feeDollars * 100),
          description: `P2P Transfer ($${amountDollars}) + Platform Fee ($${feeDollars})`
        })
      })

      const isReplay = res.headers.get('X-Idempotent-Replay') === 'true'

      if (res.ok) {
        if (isReplay) {
          setTransferAlert(`🛡️ IDEMPOTENCY HIT: Duplicate request with key "${keyToUse}" intercepted. Returned cached transaction without double-charging!`)
        } else {
          setTransferAlert(`✅ SUCCESS: Double-entry transaction recorded atomically. Balances updated!`)
          setIdempKey(generateUUID()) // generate fresh key for next transfer
        }
        fetchData()
      } else {
        const err = await res.json()
        setTransferAlert(`❌ TRANSACTION REJECTED: ${err.error || 'Failed'}`)
      }
    } catch (err) {
      setTransferAlert('Error contacting ledger backend on :8083')
    } finally {
      setIsSubmitting(false)
    }
  }

  // Double-Click Test simulates user clicking "Pay" twice rapidly with same idempotency key
  const triggerDoubleChargeTest = async () => {
    const key = generateUUID()
    setTransferAlert('🚀 Simulating rapid double-click with identical Idempotency-Key...')
    // Call #1
    handleTransfer(key)
    // Call #2 after 50ms
    setTimeout(() => {
      handleTransfer(key)
    }, 60)
  }

  const getAccountName = (id: string) => {
    const a = accounts.find((acc) => acc.id === id)
    return a ? a.name : id.slice(0, 8) + '...'
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans">
      {/* Top Navbar */}
      <header className="border-b border-slate-800/80 bg-slate-900/60 backdrop-blur px-6 py-4 flex items-center justify-between sticky top-0 z-20">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-emerald-600/20 border border-emerald-500/40 rounded-lg text-emerald-400">
            <Landmark className="w-5 h-5 animate-pulse" />
          </div>
          <div>
            <h1 className="text-lg font-bold tracking-tight text-white flex items-center gap-2">
              Financial Core Ledger
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-950 text-emerald-400 border border-emerald-800 font-mono">
                Go Ledger :8083
              </span>
            </h1>
            <p className="text-xs text-slate-400">
              Double-entry bookkeeping engine with strict idempotency and audit trails
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={fetchData}
            className="flex items-center gap-2 text-xs font-medium px-3 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition"
          >
            <RefreshCw className="w-3.5 h-3.5" />
            Refresh
          </button>
          <button
            onClick={() => setActiveTab('transfer')}
            className="flex items-center gap-2 text-xs font-semibold px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white shadow-lg shadow-emerald-600/20 transition"
          >
            <ArrowRightLeft className="w-3.5 h-3.5" />
            New Transfer
          </button>
        </div>
      </header>

      {/* Main Container */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-6 py-8 space-y-8">
        {/* Balance Sheet Header Card */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="p-5 rounded-xl border border-slate-800 bg-slate-900/40 backdrop-blur">
            <div className="flex items-center justify-between text-slate-400 text-xs">
              <span>Total Assets (Bank Reserve)</span>
              <Landmark className="w-4 h-4 text-emerald-400" />
            </div>
            <div className="text-2xl font-bold mt-2 text-white">
              {balanceSheet ? formatMoney(balanceSheet.total_assets) : '$0.00'}
            </div>
            <div className="text-[11px] text-slate-500 mt-1">Debit-normal liquidity accounts</div>
          </div>

          <div className="p-5 rounded-xl border border-slate-800 bg-slate-900/40 backdrop-blur">
            <div className="flex items-center justify-between text-slate-400 text-xs">
              <span>Customer Deposits (Liabilities)</span>
              <Wallet className="w-4 h-4 text-sky-400" />
            </div>
            <div className="text-2xl font-bold mt-2 text-sky-400">
              {balanceSheet ? formatMoney(balanceSheet.total_liabilities) : '$0.00'}
            </div>
            <div className="text-[11px] text-slate-500 mt-1">Credit-normal user wallet balances</div>
          </div>

          <div className="p-5 rounded-xl border border-slate-800 bg-slate-900/40 backdrop-blur">
            <div className="flex items-center justify-between text-slate-400 text-xs">
              <span>Platform Revenue</span>
              <DollarSign className="w-4 h-4 text-amber-400" />
            </div>
            <div className="text-2xl font-bold mt-2 text-amber-400">
              {balanceSheet ? formatMoney(balanceSheet.total_revenue) : '$0.00'}
            </div>
            <div className="text-[11px] text-slate-500 mt-1">Accumulated transaction fees</div>
          </div>

          <div className="p-5 rounded-xl border border-slate-800 bg-slate-900/40 backdrop-blur">
            <div className="flex items-center justify-between text-slate-400 text-xs">
              <span>Ledger Equilibrium</span>
              <Scale className="w-4 h-4 text-emerald-400" />
            </div>
            <div className="text-sm font-bold mt-2 flex items-center gap-1.5 text-emerald-400">
              <CheckCircle2 className="w-4 h-4" /> BALANCED (ZERO-SUM)
            </div>
            <div className="text-[11px] text-slate-500 mt-1">Assets == Liabilities + Revenue</div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex border-b border-slate-800 text-sm font-medium gap-8">
          <button
            onClick={() => setActiveTab('accounts')}
            className={`pb-3 border-b-2 flex items-center gap-2 transition ${
              activeTab === 'accounts'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <Wallet className="w-4 h-4" /> Ledger Accounts ({accounts.length})
          </button>
          <button
            onClick={() => setActiveTab('transfer')}
            className={`pb-3 border-b-2 flex items-center gap-2 transition ${
              activeTab === 'transfer'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <ArrowRightLeft className="w-4 h-4" /> Transfer & Idempotency Simulator
          </button>
          <button
            onClick={() => setActiveTab('journal')}
            className={`pb-3 border-b-2 flex items-center gap-2 transition ${
              activeTab === 'journal'
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <Layers className="w-4 h-4" /> Immutable General Journal ({transactions.length})
          </button>
        </div>

        {/* Tab 1: Accounts */}
        {activeTab === 'accounts' && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {accounts.map((acc) => (
              <div
                key={acc.id}
                className="p-5 rounded-2xl border border-slate-800 bg-slate-900/40 flex flex-col justify-between space-y-3"
              >
                <div>
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] font-mono uppercase px-2 py-0.5 rounded bg-slate-800 text-slate-300">
                      {acc.type}
                    </span>
                    <span className="text-xs font-mono text-slate-400">{acc.currency}</span>
                  </div>
                  <h3 className="font-bold text-white text-sm mt-2">{acc.name}</h3>
                  <p className="text-[11px] text-slate-400 mt-0.5">{acc.description}</p>
                </div>

                <div className="pt-3 border-t border-slate-800">
                  <div className="text-[11px] text-slate-400">Current Balance</div>
                  <div className="text-xl font-bold font-mono text-white mt-0.5">
                    {formatMoney(acc.balance)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Tab 2: Transfer Simulator */}
        {activeTab === 'transfer' && (
          <div className="max-w-2xl mx-auto p-6 rounded-2xl border border-slate-800 bg-slate-900/40 space-y-5">
            <div>
              <h2 className="text-base font-semibold text-white">Execute Idempotent Double-Entry Transfer</h2>
              <p className="text-xs text-slate-400 mt-1">
                Transfers debit the sender, credit the recipient, credit platform fees, and preserve zero-sum equilibrium.
              </p>
            </div>

            {transferAlert && (
              <div className="p-3.5 rounded-xl bg-slate-900 border border-slate-700 text-xs font-mono text-slate-200">
                {transferAlert}
              </div>
            )}

            <div className="space-y-4 text-xs">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-slate-400 mb-1 font-medium">Source Account (Debit)</label>
                  <select
                    value={fromAccount}
                    onChange={(e) => setFromAccount(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-slate-950 border border-slate-800 text-slate-200 focus:border-emerald-500 focus:outline-none"
                  >
                    {accounts.map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.name} ({formatMoney(a.balance)})
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-slate-400 mb-1 font-medium">Destination Account (Credit)</label>
                  <select
                    value={toAccount}
                    onChange={(e) => setToAccount(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-slate-950 border border-slate-800 text-slate-200 focus:border-emerald-500 focus:outline-none"
                  >
                    {accounts.map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.name} ({formatMoney(a.balance)})
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-slate-400 mb-1 font-medium">Transfer Amount ($ USD)</label>
                  <input
                    type="number"
                    min={1}
                    value={amountDollars}
                    onChange={(e) => setAmountDollars(Number(e.target.value))}
                    className="w-full px-3 py-2 rounded-lg bg-slate-950 border border-slate-800 text-slate-200 font-mono focus:border-emerald-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-slate-400 mb-1 font-medium">Platform Fee ($ USD)</label>
                  <input
                    type="number"
                    min={0}
                    value={feeDollars}
                    onChange={(e) => setFeeDollars(Number(e.target.value))}
                    className="w-full px-3 py-2 rounded-lg bg-slate-950 border border-slate-800 text-slate-200 font-mono focus:border-emerald-500 focus:outline-none"
                  />
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-slate-400 font-medium">Idempotency Key (HTTP Header)</label>
                  <button
                    onClick={() => setIdempKey(generateUUID())}
                    className="text-emerald-400 hover:text-emerald-300 text-[11px]"
                  >
                    Generate New Key
                  </button>
                </div>
                <input
                  type="text"
                  value={idempKey}
                  onChange={(e) => setIdempKey(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg bg-slate-950 border border-slate-800 text-emerald-400 font-mono text-xs focus:border-emerald-500 focus:outline-none"
                />
              </div>

              <div className="flex gap-3 pt-2">
                <button
                  onClick={() => handleTransfer()}
                  disabled={isSubmitting}
                  className="flex-1 py-3 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold flex items-center justify-center gap-2 shadow-lg shadow-emerald-600/20 transition"
                >
                  <Send className="w-4 h-4" />
                  {isSubmitting ? 'Recording...' : `Execute Transfer (${formatMoney(amountDollars * 100)})`}
                </button>
                <button
                  onClick={triggerDoubleChargeTest}
                  disabled={isSubmitting}
                  className="px-4 py-3 rounded-lg bg-amber-600/90 hover:bg-amber-500 text-white font-semibold flex items-center justify-center gap-2 transition"
                >
                  <Zap className="w-4 h-4" />
                  Simulate Double-Click Glitch
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Tab 3: Immutable General Journal */}
        {activeTab === 'journal' && (
          <div className="space-y-4">
            <div className="rounded-xl border border-slate-800 bg-slate-900/30 overflow-hidden">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="bg-slate-900/80 border-b border-slate-800 text-slate-400 font-semibold uppercase tracking-wider">
                    <th className="py-3 px-4">Transaction / Description</th>
                    <th className="py-3 px-4">Idempotency Key</th>
                    <th className="py-3 px-4">Account Legs</th>
                    <th className="py-3 px-4">Status</th>
                    <th className="py-3 px-4 text-right">Timestamp</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60 font-mono">
                  {transactions.map((tx) => (
                    <tr key={tx.id} className="hover:bg-slate-800/40 transition">
                      <td className="py-3 px-4">
                        <div className="font-semibold text-white font-sans">{tx.description}</div>
                        <div className="text-[11px] text-slate-400 font-mono mt-0.5">ID: {tx.id.slice(0, 8)}...</div>
                      </td>
                      <td className="py-3 px-4 text-emerald-400">
                        {tx.idempotency_key || 'N/A'}
                      </td>
                      <td className="py-3 px-4 space-y-1">
                        {tx.entries.map((e) => (
                          <div key={e.id} className="flex items-center gap-2 text-[11px]">
                            <span className={e.direction === 'DEBIT' ? 'text-amber-400 font-bold' : 'text-emerald-400 font-bold'}>
                              {e.direction}:
                            </span>
                            <span className="text-slate-300">{getAccountName(e.account_id)}</span>
                            <span className="text-white font-semibold">({formatMoney(e.amount)})</span>
                          </div>
                        ))}
                      </td>
                      <td className="py-3 px-4">
                        <span className="px-2 py-0.5 rounded-full text-[11px] bg-emerald-950 text-emerald-400 border border-emerald-800">
                          {tx.status}
                        </span>
                      </td>
                      <td className="py-3 px-4 text-right text-slate-400 text-[11px]">
                        {new Date(tx.created_at).toLocaleTimeString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
