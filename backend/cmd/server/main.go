package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/portfolio/fintech-ledger/internal/api"
	"github.com/portfolio/fintech-ledger/internal/idempotency"
	"github.com/portfolio/fintech-ledger/internal/ledger"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083" // running on port 8083
	}

	ledg := ledger.NewLedger()
	idemp := idempotency.NewStore()
	server := api.NewServer(ledg, idemp)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("=============================================================")
		log.Printf("💳 Financial Core Ledger & Idempotent Engine running on :%s", port)
		log.Printf("⚖️  Double-Entry Accounting: Active & Balanced")
		log.Printf("🔒 Idempotency Protection: Active (Prevents Double-Charges)")
		log.Printf("=============================================================")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Financial Ledger gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Ledger forced to shutdown: %v", err)
	}

	log.Println("Financial Ledger stopped successfully.")
}
