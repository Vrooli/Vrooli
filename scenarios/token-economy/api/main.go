package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

var (
	db    *sql.DB
	rdb   *redis.Client
	ctx   = context.Background()
	port  string
)

type Token struct {
	ID             string  `json:"id"`
	HouseholdID    string  `json:"household_id"`
	Symbol         string  `json:"symbol"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	TotalSupply    float64 `json:"total_supply"`
	MaxSupply      float64 `json:"max_supply,omitempty"`
	Decimals       int     `json:"decimals"`
	CreatorScenario string `json:"creator_scenario"`
	Metadata       map[string]interface{} `json:"metadata"`
	IconURL        string  `json:"icon_url,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

type Wallet struct {
	ID           string                 `json:"id"`
	HouseholdID  string                 `json:"household_id"`
	UserID       string                 `json:"user_id,omitempty"`
	ScenarioName string                 `json:"scenario_name,omitempty"`
	Address      string                 `json:"address"`
	Type         string                 `json:"type"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at"`
}

type Balance struct {
	TokenID    string  `json:"token_id"`
	Symbol     string  `json:"symbol"`
	Amount     float64 `json:"amount"`
	Locked     float64 `json:"locked"`
	ValueUSD   float64 `json:"value_usd,omitempty"`
}

type Transaction struct {
	ID         string                 `json:"id"`
	Hash       string                 `json:"hash"`
	FromWallet string                 `json:"from_wallet,omitempty"`
	ToWallet   string                 `json:"to_wallet,omitempty"`
	TokenID    string                 `json:"token_id"`
	Amount     float64                `json:"amount"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  string                 `json:"created_at"`
}

type Achievement struct {
	ID         string `json:"id"`
	TokenID    string `json:"token_id"`
	Title      string `json:"title"`
	Scenario   string `json:"scenario"`
	Rarity     string `json:"rarity"`
	EarnedAt   string `json:"earned_at"`
}

type CreateTokenRequest struct {
	Symbol        string                 `json:"symbol"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	InitialSupply float64                `json:"initial_supply,omitempty"`
	MaxSupply     float64                `json:"max_supply,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type MintTokenRequest struct {
	TokenID  string                 `json:"token_id"`
	ToWallet string                 `json:"to_wallet"`
	Amount   float64                `json:"amount"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type TransferRequest struct {
	FromWallet string  `json:"from_wallet"`
	ToWallet   string  `json:"to_wallet"`
	TokenID    string  `json:"token_id"`
	Amount     float64 `json:"amount"`
	Memo       string  `json:"memo,omitempty"`
}

type SwapRequest struct {
	FromToken        string  `json:"from_token"`
	ToToken          string  `json:"to_token"`
	Amount           float64 `json:"amount"`
	SlippageTolerance float64 `json:"slippage_tolerance,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

func init() {
	godotenv.Load()
	
	port = os.Getenv("PORT_TOKEN_ECONOMY_API")
	if port == "" {
		port = "11080"
	}
	
	// Initialize PostgreSQL
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "token_economy"
	}
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Initialize Redis
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	
	// Test connections
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
		rdb = nil
	}
	
	log.Println("Token Economy API initialized successfully")
}

func main() {
	router := mux.NewRouter()
	
	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
	
	// Token endpoints
	router.HandleFunc("/api/v1/tokens/create", createTokenHandler).Methods("POST")
	router.HandleFunc("/api/v1/tokens/mint", mintTokenHandler).Methods("POST")
	router.HandleFunc("/api/v1/tokens/transfer", transferTokenHandler).Methods("POST")
	router.HandleFunc("/api/v1/tokens/swap", swapTokenHandler).Methods("POST")
	router.HandleFunc("/api/v1/tokens", listTokensHandler).Methods("GET")
	router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler).Methods("GET")
	
	// Wallet endpoints
	router.HandleFunc("/api/v1/wallets/create", createWalletHandler).Methods("POST")
	router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler).Methods("GET")
	router.HandleFunc("/api/v1/wallets/{wallet_id}", getWalletHandler).Methods("GET")
	
	// Achievement endpoints
	router.HandleFunc("/api/v1/achievements/{user_id}", getAchievementsHandler).Methods("GET")
	router.HandleFunc("/api/v1/achievements/award", awardAchievementHandler).Methods("POST")
	
	// Transaction endpoints
	router.HandleFunc("/api/v1/transactions", getTransactionsHandler).Methods("GET")
	router.HandleFunc("/api/v1/transactions/{hash}", getTransactionHandler).Methods("GET")
	
	// Admin endpoints
	router.HandleFunc("/api/v1/admin/economy/rules", setEconomyRulesHandler).Methods("POST")
	router.HandleFunc("/api/v1/admin/analytics", getAnalyticsHandler).Methods("GET")
	
	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(router)
	
	log.Printf("Token Economy API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]bool{
			"database": db.Ping() == nil,
			"redis": rdb != nil && rdb.Ping(ctx).Err() == nil,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func createTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.Symbol == "" || req.Name == "" || req.Type == "" {
		sendError(w, "Symbol, name, and type are required", http.StatusBadRequest)
		return
	}
	
	// Get household ID from context (would come from auth in production)
	householdID := "00000000-0000-0000-0000-000000000001"
	
	tokenID := uuid.New().String()
	
	// Create token in database
	query := `
		INSERT INTO tokens (id, household_id, symbol, name, type, total_supply, max_supply, 
							decimals, creator_scenario, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at`
	
	var createdAt time.Time
	err := db.QueryRow(query, tokenID, householdID, req.Symbol, req.Name, req.Type,
		req.InitialSupply, req.MaxSupply, 18, "token-economy", req.Metadata).Scan(&createdAt)
	
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			sendError(w, "Token symbol already exists", http.StatusConflict)
		} else {
			sendError(w, "Failed to create token", http.StatusInternalServerError)
		}
		return
	}
	
	response := map[string]interface{}{
		"token_id": tokenID,
		"address": fmt.Sprintf("0x%s", strings.ReplaceAll(tokenID, "-", "")),
		"created_at": createdAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func mintTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req MintTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Call mint function
	var transactionID string
	err := db.QueryRow(`SELECT mint_tokens($1, $2, $3, $4, $5)`,
		req.TokenID, req.ToWallet, req.Amount, "token-economy", req.Metadata).Scan(&transactionID)
	
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to mint tokens: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Get updated supply
	var newSupply float64
	err = db.QueryRow("SELECT total_supply FROM tokens WHERE id = $1", req.TokenID).Scan(&newSupply)
	if err != nil {
		log.Printf("Failed to get new supply: %v", err)
	}
	
	// Invalidate cache
	if rdb != nil {
		cacheKey := fmt.Sprintf("balance:%s:%s", req.ToWallet, req.TokenID)
		rdb.Del(ctx, cacheKey)
	}
	
	response := map[string]interface{}{
		"transaction_hash": transactionID,
		"new_supply": newSupply,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func transferTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Resolve wallet addresses if they're aliases
	fromWallet := resolveWalletAddress(req.FromWallet)
	toWallet := resolveWalletAddress(req.ToWallet)
	
	// Call transfer function
	var transactionID string
	metadata := map[string]interface{}{"memo": req.Memo}
	err := db.QueryRow(`SELECT transfer_tokens($1, $2, $3, $4, $5)`,
		fromWallet, toWallet, req.TokenID, req.Amount, metadata).Scan(&transactionID)
	
	if err != nil {
		if strings.Contains(err.Error(), "Insufficient balance") {
			sendError(w, "Insufficient balance", http.StatusBadRequest)
		} else {
			sendError(w, fmt.Sprintf("Transfer failed: %v", err), http.StatusInternalServerError)
		}
		return
	}
	
	// Get new balance
	var newBalance float64
	err = db.QueryRow(`SELECT amount FROM balances WHERE wallet_id = $1 AND token_id = $2`,
		fromWallet, req.TokenID).Scan(&newBalance)
	
	// Invalidate cache for both wallets
	if rdb != nil {
		rdb.Del(ctx, fmt.Sprintf("balance:%s:%s", fromWallet, req.TokenID))
		rdb.Del(ctx, fmt.Sprintf("balance:%s:%s", toWallet, req.TokenID))
	}
	
	response := map[string]interface{}{
		"transaction_hash": transactionID,
		"new_balance": newBalance,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func swapTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req SwapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Get exchange rate
	var rate float64
	err := db.QueryRow(`
		SELECT rate FROM exchange_rates 
		WHERE from_token = $1 AND to_token = $2`,
		req.FromToken, req.ToToken).Scan(&rate)
	
	if err != nil {
		sendError(w, "No exchange rate available", http.StatusNotFound)
		return
	}
	
	receivedAmount := req.Amount * rate
	
	// TODO: Implement actual swap logic
	transactionHash := uuid.New().String()
	
	response := map[string]interface{}{
		"transaction_hash": transactionHash,
		"received_amount": receivedAmount,
		"exchange_rate": rate,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type         string `json:"type"`
		UserID       string `json:"user_id,omitempty"`
		ScenarioName string `json:"scenario_name,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	householdID := "00000000-0000-0000-0000-000000000001"
	walletID := uuid.New().String()
	address := fmt.Sprintf("0x%s", strings.ReplaceAll(walletID, "-", ""))
	
	query := `
		INSERT INTO wallets (id, household_id, user_id, scenario_name, address, type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`
	
	var createdAt time.Time
	err := db.QueryRow(query, walletID, householdID, req.UserID, req.ScenarioName, 
		address, req.Type).Scan(&createdAt)
	
	if err != nil {
		sendError(w, "Failed to create wallet", http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"wallet_id": walletID,
		"address": address,
		"created_at": createdAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	walletID := resolveWalletAddress(vars["wallet_id"])
	
	// Try cache first
	if rdb != nil {
		cached, err := rdb.Get(ctx, fmt.Sprintf("balances:%s", walletID)).Result()
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(cached))
			return
		}
	}
	
	// Query database
	rows, err := db.Query(`
		SELECT b.token_id, t.symbol, b.amount, b.locked_amount
		FROM balances b
		JOIN tokens t ON b.token_id = t.id
		WHERE b.wallet_id = $1`, walletID)
	
	if err != nil {
		sendError(w, "Failed to get balances", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	balances := []Balance{}
	for rows.Next() {
		var b Balance
		err := rows.Scan(&b.TokenID, &b.Symbol, &b.Amount, &b.Locked)
		if err != nil {
			continue
		}
		balances = append(balances, b)
	}
	
	response := map[string]interface{}{
		"balances": balances,
	}
	
	// Cache result
	if rdb != nil {
		data, _ := json.Marshal(response)
		rdb.Set(ctx, fmt.Sprintf("balances:%s", walletID), data, 60*time.Second)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getAchievementsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]
	
	// Get user's wallet
	var walletID string
	err := db.QueryRow(`SELECT id FROM wallets WHERE user_id = $1 AND type = 'user'`, 
		userID).Scan(&walletID)
	
	if err != nil {
		sendError(w, "Wallet not found", http.StatusNotFound)
		return
	}
	
	// Get achievements
	rows, err := db.Query(`
		SELECT id, token_id, title, scenario, rarity, earned_at
		FROM achievements
		WHERE wallet_id = $1
		ORDER BY earned_at DESC`, walletID)
	
	if err != nil {
		sendError(w, "Failed to get achievements", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	achievements := []Achievement{}
	for rows.Next() {
		var a Achievement
		err := rows.Scan(&a.ID, &a.TokenID, &a.Title, &a.Scenario, &a.Rarity, &a.EarnedAt)
		if err != nil {
			continue
		}
		achievements = append(achievements, a)
	}
	
	response := map[string]interface{}{
		"achievements": achievements,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func awardAchievementHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement achievement awarding
	sendError(w, "Not implemented", http.StatusNotImplemented)
}

func getTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	walletID := r.URL.Query().Get("wallet")
	tokenID := r.URL.Query().Get("token")
	limit := r.URL.Query().Get("limit")
	
	if limit == "" {
		limit = "100"
	}
	
	query := `
		SELECT id, hash, from_wallet, to_wallet, token_id, amount, type, status, created_at
		FROM transactions
		WHERE 1=1`
	
	args := []interface{}{}
	argCount := 0
	
	if walletID != "" {
		argCount++
		query += fmt.Sprintf(" AND (from_wallet = $%d OR to_wallet = $%d)", argCount, argCount)
		args = append(args, resolveWalletAddress(walletID))
	}
	
	if tokenID != "" {
		argCount++
		query += fmt.Sprintf(" AND token_id = $%d", argCount)
		args = append(args, tokenID)
	}
	
	query += " ORDER BY created_at DESC LIMIT " + limit
	
	rows, err := db.Query(query, args...)
	if err != nil {
		sendError(w, "Failed to get transactions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	transactions := []Transaction{}
	for rows.Next() {
		var t Transaction
		var fromWallet, toWallet sql.NullString
		err := rows.Scan(&t.ID, &t.Hash, &fromWallet, &toWallet, &t.TokenID, 
			&t.Amount, &t.Type, &t.Status, &t.CreatedAt)
		if err != nil {
			continue
		}
		if fromWallet.Valid {
			t.FromWallet = fromWallet.String
		}
		if toWallet.Valid {
			t.ToWallet = toWallet.String
		}
		transactions = append(transactions, t)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

func getTransactionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]
	
	var t Transaction
	var fromWallet, toWallet sql.NullString
	
	err := db.QueryRow(`
		SELECT id, hash, from_wallet, to_wallet, token_id, amount, type, status, created_at
		FROM transactions
		WHERE hash = $1`, hash).Scan(&t.ID, &t.Hash, &fromWallet, &toWallet, 
		&t.TokenID, &t.Amount, &t.Type, &t.Status, &t.CreatedAt)
	
	if err != nil {
		sendError(w, "Transaction not found", http.StatusNotFound)
		return
	}
	
	if fromWallet.Valid {
		t.FromWallet = fromWallet.String
	}
	if toWallet.Valid {
		t.ToWallet = toWallet.String
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func listTokensHandler(w http.ResponseWriter, r *http.Request) {
	householdID := "00000000-0000-0000-0000-000000000001"
	
	rows, err := db.Query(`
		SELECT id, symbol, name, type, total_supply, created_at
		FROM tokens
		WHERE household_id = $1 AND is_active = true
		ORDER BY created_at DESC`, householdID)
	
	if err != nil {
		sendError(w, "Failed to list tokens", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	tokens := []Token{}
	for rows.Next() {
		var t Token
		err := rows.Scan(&t.ID, &t.Symbol, &t.Name, &t.Type, &t.TotalSupply, &t.CreatedAt)
		if err != nil {
			continue
		}
		tokens = append(tokens, t)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

func getTokenHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tokenID := vars["id"]
	
	var t Token
	err := db.QueryRow(`
		SELECT id, symbol, name, type, total_supply, max_supply, decimals, created_at
		FROM tokens
		WHERE id = $1`, tokenID).Scan(&t.ID, &t.Symbol, &t.Name, &t.Type, 
		&t.TotalSupply, &t.MaxSupply, &t.Decimals, &t.CreatedAt)
	
	if err != nil {
		sendError(w, "Token not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func getWalletHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	walletID := resolveWalletAddress(vars["wallet_id"])
	
	var w_data Wallet
	var userID, scenarioName sql.NullString
	
	err := db.QueryRow(`
		SELECT id, household_id, user_id, scenario_name, address, type, created_at
		FROM wallets
		WHERE id = $1 OR address = $1`, walletID).Scan(&w_data.ID, &w_data.HouseholdID, 
		&userID, &scenarioName, &w_data.Address, &w_data.Type, &w_data.CreatedAt)
	
	if err != nil {
		sendError(w, "Wallet not found", http.StatusNotFound)
		return
	}
	
	if userID.Valid {
		w_data.UserID = userID.String
	}
	if scenarioName.Valid {
		w_data.ScenarioName = scenarioName.String
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(w_data)
}

func setEconomyRulesHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement economy rules setting (parent controls)
	sendError(w, "Not implemented", http.StatusNotImplemented)
}

func getAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	householdID := "00000000-0000-0000-0000-000000000001"
	
	// Get basic stats
	var totalTransactions int
	var totalVolume float64
	var activeWallets int
	
	db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE household_id = $1 AND status = 'confirmed'`, householdID).Scan(&totalTransactions, &totalVolume)
	
	db.QueryRow(`
		SELECT COUNT(DISTINCT wallet_id)
		FROM balances
		WHERE amount > 0`, ).Scan(&activeWallets)
	
	// Get top earners
	rows, _ := db.Query(`
		SELECT w.address, SUM(b.amount) as total
		FROM balances b
		JOIN wallets w ON b.wallet_id = w.id
		WHERE w.household_id = $1
		GROUP BY w.address
		ORDER BY total DESC
		LIMIT 10`, householdID)
	defer rows.Close()
	
	topEarners := []map[string]interface{}{}
	for rows.Next() {
		var address string
		var total float64
		rows.Scan(&address, &total)
		topEarners = append(topEarners, map[string]interface{}{
			"address": address,
			"total": total,
		})
	}
	
	analytics := map[string]interface{}{
		"total_transactions": totalTransactions,
		"total_volume": totalVolume,
		"active_wallets": activeWallets,
		"top_earners": topEarners,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

func resolveWalletAddress(input string) string {
	// If it's an alias like @alice, resolve it
	if strings.HasPrefix(input, "@") {
		// TODO: Implement alias resolution
		return input
	}
	
	// If it's a short address, try to find the full one
	if !strings.HasPrefix(input, "0x") && len(input) < 36 {
		var fullAddress string
		err := db.QueryRow(`SELECT id FROM wallets WHERE address LIKE $1 || '%'`, input).Scan(&fullAddress)
		if err == nil {
			return fullAddress
		}
	}
	
	return input
}

func sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SuccessResponse{Success: true, Data: data})
}