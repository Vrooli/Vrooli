package main

import (
	"encoding/json"
	"os"
	"testing"

	"financial-calculators-hub/lib"
	"github.com/google/uuid"
)

// Note: These tests require a PostgreSQL instance to be available
// They will be skipped if POSTGRES_HOST is not set

func setupTestDatabase(t *testing.T) func() {
	t.Helper()

	// Skip if no database available
	if os.Getenv("POSTGRES_HOST") == "" {
		t.Skip("Skipping database tests - POSTGRES_HOST not set")
	}

	// Initialize database
	if err := initDatabase(); err != nil {
		t.Skipf("Skipping database tests - database not available: %v", err)
	}

	// Store original db reference
	originalDB := db

	return func() {
		// Cleanup test data
		if db != nil {
			db.Exec("TRUNCATE TABLE calculations CASCADE")
			db.Exec("TRUNCATE TABLE net_worth_entries CASCADE")
			db.Exec("TRUNCATE TABLE tax_calculations CASCADE")
			db.Exec("TRUNCATE TABLE saved_scenarios CASCADE")
		}
		db = originalDB
	}
}

func TestInitDatabase(t *testing.T) {
	if os.Getenv("POSTGRES_HOST") == "" {
		t.Skip("Skipping database tests - POSTGRES_HOST not set")
	}

	cleanup := setupTestDatabase(t)
	defer cleanup()

	// Verify database connection
	if db == nil {
		t.Fatal("Database not initialized")
	}

	err := db.Ping()
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}

func TestSaveCalculation(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		input := lib.FIREInput{
			CurrentAge:           30,
			CurrentSavings:       100000,
			AnnualIncome:         100000,
			AnnualExpenses:       50000,
			SavingsRate:          50,
			ExpectedReturn:       7,
			TargetWithdrawalRate: 4,
		}

		result := lib.CalculateFIRE(input)

		userID := uuid.New().String()
		notes := "Test calculation"

		id, err := saveCalculation("fire", input, result, &userID, &notes)
		if err != nil {
			t.Fatalf("Failed to save calculation: %v", err)
		}

		if id == "" {
			t.Error("Expected non-empty calculation ID")
		}

		// Verify it was saved
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM calculations WHERE id = $1", id).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query calculation: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 calculation, found %d", count)
		}
	})

	t.Run("WithoutUserID", func(t *testing.T) {
		input := lib.CompoundInterestInput{
			Principal:           10000,
			AnnualRate:          7,
			Years:               10,
			CompoundFrequency:   "annually",
		}

		result := lib.CalculateCompoundInterest(input)

		id, err := saveCalculation("compound-interest", input, result, nil, nil)
		if err != nil {
			t.Fatalf("Failed to save calculation: %v", err)
		}

		if id == "" {
			t.Error("Expected non-empty calculation ID")
		}
	})
}

func TestGetCalculationHistory(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	// Insert test data
	userID := uuid.New().String()
	notes := "Test calc 1"

	input1 := lib.FIREInput{
		CurrentAge:     30,
		CurrentSavings: 100000,
		AnnualIncome:   100000,
		AnnualExpenses: 50000,
		SavingsRate:    50,
	}
	result1 := lib.CalculateFIRE(input1)
	saveCalculation("fire", input1, result1, &userID, &notes)

	input2 := lib.CompoundInterestInput{
		Principal:  10000,
		AnnualRate: 7,
		Years:      10,
	}
	result2 := lib.CalculateCompoundInterest(input2)
	saveCalculation("compound-interest", input2, result2, &userID, &notes)

	t.Run("AllTypes", func(t *testing.T) {
		history, err := getCalculationHistory("", 50, &userID)
		if err != nil {
			t.Fatalf("Failed to get history: %v", err)
		}

		if len(history) < 2 {
			t.Errorf("Expected at least 2 calculations, got %d", len(history))
		}
	})

	t.Run("FilterByType", func(t *testing.T) {
		history, err := getCalculationHistory("fire", 50, &userID)
		if err != nil {
			t.Fatalf("Failed to get history: %v", err)
		}

		if len(history) < 1 {
			t.Error("Expected at least 1 FIRE calculation")
		}

		for _, calc := range history {
			if calc["calculator_type"] != "fire" {
				t.Error("Expected only FIRE calculations")
			}
		}
	})

	t.Run("LimitResults", func(t *testing.T) {
		history, err := getCalculationHistory("", 1, &userID)
		if err != nil {
			t.Fatalf("Failed to get history: %v", err)
		}

		if len(history) > 1 {
			t.Errorf("Expected max 1 calculation, got %d", len(history))
		}
	})
}

func TestSaveNetWorthEntry(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New().String()
		input := lib.NetWorthInput{
			Assets: map[string]float64{
				"cash":        50000,
				"investments": 150000,
			},
			Liabilities: map[string]float64{
				"mortgage": 200000,
			},
		}

		netWorth := 0.0 // Will be calculated
		for _, v := range input.Assets {
			netWorth += v
		}
		for _, v := range input.Liabilities {
			netWorth -= v
		}

		err := saveNetWorthEntry(userID, input, netWorth)
		if err != nil {
			t.Fatalf("Failed to save net worth entry: %v", err)
		}

		// Verify it was saved
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM net_worth_entries WHERE user_id = $1", userID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query net worth entry: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 entry, found %d", count)
		}
	})

	t.Run("UpdateExistingEntry", func(t *testing.T) {
		userID := uuid.New().String()
		input := lib.NetWorthInput{
			Assets: map[string]float64{
				"cash": 10000,
			},
			Liabilities: map[string]float64{},
		}

		// Save first entry
		err := saveNetWorthEntry(userID, input, 10000)
		if err != nil {
			t.Fatalf("Failed to save first entry: %v", err)
		}

		// Update with new values
		input.Assets["cash"] = 20000
		err = saveNetWorthEntry(userID, input, 20000)
		if err != nil {
			t.Fatalf("Failed to update entry: %v", err)
		}

		// Should still have only 1 entry (due to UNIQUE constraint)
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM net_worth_entries WHERE user_id = $1", userID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query net worth entry: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 entry after update, found %d", count)
		}
	})
}

func TestGetNetWorthHistory(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	userID := uuid.New().String()

	// Insert multiple entries
	for i := 0; i < 3; i++ {
		input := lib.NetWorthInput{
			Assets: map[string]float64{
				"cash": float64((i + 1) * 10000),
			},
			Liabilities: map[string]float64{},
		}
		saveNetWorthEntry(userID, input, float64((i+1)*10000))
	}

	history, err := getNetWorthHistory(userID)
	if err != nil {
		t.Fatalf("Failed to get net worth history: %v", err)
	}

	// Note: Due to UNIQUE(user_id, entry_date), we might only get 1 entry if all saved on same date
	if len(history) < 1 {
		t.Error("Expected at least 1 history entry")
	}
}

func TestSaveTaxCalculation(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	userID := uuid.New().String()
	input := lib.TaxOptimizerInput{
		Income:       75000,
		TaxYear:      2024,
		FilingStatus: "single",
		Deductions: map[string]float64{
			"401k": 10000,
		},
		Credits: map[string]float64{},
	}

	result := lib.CalculateTaxOptimization(input)

	err := saveTaxCalculation(userID, input, result)
	if err != nil {
		t.Fatalf("Failed to save tax calculation: %v", err)
	}

	// Verify it was saved
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tax_calculations WHERE user_id = $1", userID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query tax calculation: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 tax calculation, found %d", count)
	}
}

func TestDatabaseNilSafety(t *testing.T) {
	// Test functions handle nil db gracefully
	originalDB := db
	db = nil
	defer func() { db = originalDB }()

	t.Run("SaveCalculation_NilDB", func(t *testing.T) {
		input := lib.FIREInput{CurrentAge: 30}
		result := lib.FIREResult{}

		_, err := saveCalculation("fire", input, result, nil, nil)
		if err == nil {
			t.Error("Expected error when db is nil")
		}
	})

	t.Run("GetCalculationHistory_NilDB", func(t *testing.T) {
		_, err := getCalculationHistory("", 10, nil)
		if err == nil {
			t.Error("Expected error when db is nil")
		}
	})

	t.Run("SaveNetWorthEntry_NilDB", func(t *testing.T) {
		input := lib.NetWorthInput{}
		err := saveNetWorthEntry("user", input, 0)
		if err != nil {
			t.Error("saveNetWorthEntry should not error with nil db (optional feature)")
		}
	})
}

func TestDatabaseSchema(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	t.Run("CalculationsTableExists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = 'calculations'
			)
		`).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check table existence: %v", err)
		}

		if !exists {
			t.Error("calculations table does not exist")
		}
	})

	t.Run("NetWorthEntriesTableExists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = 'net_worth_entries'
			)
		`).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check table existence: %v", err)
		}

		if !exists {
			t.Error("net_worth_entries table does not exist")
		}
	})

	t.Run("TaxCalculationsTableExists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = 'tax_calculations'
			)
		`).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check table existence: %v", err)
		}

		if !exists {
			t.Error("tax_calculations table does not exist")
		}
	})
}

func TestJSONBStorage(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	t.Run("ComplexJSONBRoundtrip", func(t *testing.T) {
		input := lib.FIREInput{
			CurrentAge:           30,
			CurrentSavings:       100000,
			AnnualIncome:         100000,
			AnnualExpenses:       50000,
			SavingsRate:          50,
			ExpectedReturn:       7,
			TargetWithdrawalRate: 4,
		}

		result := lib.CalculateFIRE(input)

		id, err := saveCalculation("fire", input, result, nil, nil)
		if err != nil {
			t.Fatalf("Failed to save calculation: %v", err)
		}

		// Retrieve and verify JSONB data
		var inputsJSON, outputsJSON json.RawMessage
		err = db.QueryRow(`
			SELECT inputs, outputs FROM calculations WHERE id = $1
		`, id).Scan(&inputsJSON, &outputsJSON)

		if err != nil {
			t.Fatalf("Failed to retrieve calculation: %v", err)
		}

		// Verify we can unmarshal the data
		var retrievedInput lib.FIREInput
		if err := json.Unmarshal(inputsJSON, &retrievedInput); err != nil {
			t.Fatalf("Failed to unmarshal inputs: %v", err)
		}

		if retrievedInput.CurrentAge != input.CurrentAge {
			t.Errorf("CurrentAge mismatch: expected %v, got %v",
				input.CurrentAge, retrievedInput.CurrentAge)
		}
	})
}
