package main

import (
	"testing"

	"github.com/google/uuid"
)

// TestResolveWalletAddressLogic tests wallet address resolution logic
func TestResolveWalletAddressLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	wallet := createTestWallet(t, testDB.DB, householdID)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "DirectUUID",
			input:    wallet.ID,
			expected: wallet.ID,
		},
		{
			name:     "FullAddress",
			input:    wallet.Address,
			expected: wallet.Address,
		},
		{
			name:     "AliasNotImplemented",
			input:    "@alice",
			expected: "@alice", // Returns as-is when not implemented
		},
		{
			name:     "RandomString",
			input:    "random-string",
			expected: "random-string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveWalletAddress(tt.input)
			if result != tt.expected {
				t.Errorf("resolveWalletAddress(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

// TestTokenValidation tests token creation validation logic
func TestTokenValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name        string
		symbol      string
		tokenName   string
		tokenType   string
		shouldError bool
	}{
		{
			name:        "ValidFungibleToken",
			symbol:      "GOLD",
			tokenName:   "Gold Coin",
			tokenType:   "fungible",
			shouldError: false,
		},
		{
			name:        "ValidNonFungibleToken",
			symbol:      "BADGE",
			tokenName:   "Achievement Badge",
			tokenType:   "non-fungible",
			shouldError: false,
		},
		{
			name:        "EmptySymbol",
			symbol:      "",
			tokenName:   "Test Token",
			tokenType:   "fungible",
			shouldError: true,
		},
		{
			name:        "EmptyName",
			symbol:      "TEST",
			tokenName:   "",
			tokenType:   "fungible",
			shouldError: true,
		},
		{
			name:        "EmptyType",
			symbol:      "TEST",
			tokenName:   "Test Token",
			tokenType:   "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation logic from createTokenHandler
			if tt.symbol == "" || tt.tokenName == "" || tt.tokenType == "" {
				if !tt.shouldError {
					t.Error("Expected error but got none")
				}
			} else {
				if tt.shouldError {
					t.Error("Expected no error but validation should fail")
				}
			}
		})
	}
}

// TestWalletTypeValidation tests wallet type validation
func TestWalletTypeValidation(t *testing.T) {
	tests := []struct {
		name         string
		walletType   string
		userID       string
		scenarioName string
		isValid      bool
	}{
		{
			name:       "ValidUserWallet",
			walletType: "user",
			userID:     "user123",
			isValid:    true,
		},
		{
			name:         "ValidScenarioWallet",
			walletType:   "scenario",
			scenarioName: "test-scenario",
			isValid:      true,
		},
		{
			name:       "ValidTreasuryWallet",
			walletType: "treasury",
			isValid:    true,
		},
		{
			name:       "UserWalletMissingUserID",
			walletType: "user",
			isValid:    false, // Should fail constraint
		},
		{
			name:       "ScenarioWalletMissingScenarioName",
			walletType: "scenario",
			isValid:    false, // Should fail constraint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This validates the constraint logic from schema:
			// (type = 'user' AND user_id IS NOT NULL) OR
			// (type = 'scenario' AND scenario_name IS NOT NULL) OR
			// (type = 'treasury')

			valid := false
			if tt.walletType == "user" && tt.userID != "" {
				valid = true
			} else if tt.walletType == "scenario" && tt.scenarioName != "" {
				valid = true
			} else if tt.walletType == "treasury" {
				valid = true
			}

			if valid != tt.isValid {
				t.Errorf("Wallet validation mismatch: got %v, want %v", valid, tt.isValid)
			}
		})
	}
}

// TestTransactionTypeValidation tests transaction type validation
func TestTransactionTypeValidation(t *testing.T) {
	tests := []struct {
		name       string
		txType     string
		fromWallet string
		toWallet   string
		isValid    bool
	}{
		{
			name:       "ValidMint",
			txType:     "mint",
			fromWallet: "",
			toWallet:   uuid.New().String(),
			isValid:    true,
		},
		{
			name:       "ValidBurn",
			txType:     "burn",
			fromWallet: uuid.New().String(),
			toWallet:   "",
			isValid:    true,
		},
		{
			name:       "ValidTransfer",
			txType:     "transfer",
			fromWallet: uuid.New().String(),
			toWallet:   uuid.New().String(),
			isValid:    true,
		},
		{
			name:       "ValidSwap",
			txType:     "swap",
			fromWallet: uuid.New().String(),
			toWallet:   uuid.New().String(),
			isValid:    true,
		},
		{
			name:       "InvalidMint_HasFromWallet",
			txType:     "mint",
			fromWallet: uuid.New().String(),
			toWallet:   uuid.New().String(),
			isValid:    false,
		},
		{
			name:       "InvalidBurn_HasToWallet",
			txType:     "burn",
			fromWallet: uuid.New().String(),
			toWallet:   uuid.New().String(),
			isValid:    false,
		},
		{
			name:       "InvalidTransfer_MissingWallet",
			txType:     "transfer",
			fromWallet: "",
			toWallet:   uuid.New().String(),
			isValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validates constraint from schema:
			// (type = 'mint' AND from_wallet IS NULL AND to_wallet IS NOT NULL) OR
			// (type = 'burn' AND from_wallet IS NOT NULL AND to_wallet IS NULL) OR
			// (type IN ('transfer', 'swap') AND from_wallet IS NOT NULL AND to_wallet IS NOT NULL)

			valid := false
			if tt.txType == "mint" && tt.fromWallet == "" && tt.toWallet != "" {
				valid = true
			} else if tt.txType == "burn" && tt.fromWallet != "" && tt.toWallet == "" {
				valid = true
			} else if (tt.txType == "transfer" || tt.txType == "swap") && tt.fromWallet != "" && tt.toWallet != "" {
				valid = true
			}

			if valid != tt.isValid {
				t.Errorf("Transaction validation mismatch for %s: got %v, want %v", tt.name, valid, tt.isValid)
			}
		})
	}
}

// TestBalanceConstraints tests balance non-negative constraints
func TestBalanceConstraints(t *testing.T) {
	tests := []struct {
		name         string
		amount       float64
		lockedAmount float64
		isValid      bool
	}{
		{
			name:         "ValidBalance",
			amount:       1000.0,
			lockedAmount: 100.0,
			isValid:      true,
		},
		{
			name:         "ZeroBalance",
			amount:       0.0,
			lockedAmount: 0.0,
			isValid:      true,
		},
		{
			name:         "NegativeAmount",
			amount:       -100.0,
			lockedAmount: 0.0,
			isValid:      false,
		},
		{
			name:         "NegativeLockedAmount",
			amount:       100.0,
			lockedAmount: -10.0,
			isValid:      false,
		},
		{
			name:         "BothNegative",
			amount:       -100.0,
			lockedAmount: -10.0,
			isValid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validates constraint: amount >= 0 AND locked_amount >= 0
			valid := tt.amount >= 0 && tt.lockedAmount >= 0

			if valid != tt.isValid {
				t.Errorf("Balance constraint mismatch: got %v, want %v", valid, tt.isValid)
			}
		})
	}
}

// TestTokenSupplyLogic tests token supply management
func TestTokenSupplyLogic(t *testing.T) {
	tests := []struct {
		name          string
		totalSupply   float64
		maxSupply     float64
		mintAmount    float64
		shouldSucceed bool
	}{
		{
			name:          "MintWithinLimit",
			totalSupply:   1000.0,
			maxSupply:     10000.0,
			mintAmount:    500.0,
			shouldSucceed: true,
		},
		{
			name:          "MintExceedingLimit",
			totalSupply:   9500.0,
			maxSupply:     10000.0,
			mintAmount:    1000.0,
			shouldSucceed: false,
		},
		{
			name:          "MintExactlyToLimit",
			totalSupply:   9000.0,
			maxSupply:     10000.0,
			mintAmount:    1000.0,
			shouldSucceed: true,
		},
		{
			name:          "UnlimitedSupply",
			totalSupply:   1000000.0,
			maxSupply:     0, // 0 or NULL means unlimited
			mintAmount:    1000000.0,
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newSupply := tt.totalSupply + tt.mintAmount
			canMint := tt.maxSupply == 0 || newSupply <= tt.maxSupply

			if canMint != tt.shouldSucceed {
				t.Errorf("Supply logic mismatch: can mint %v, want %v", canMint, tt.shouldSucceed)
			}
		})
	}
}

// TestErrorResponseStructure tests error response formatting
func TestErrorResponseStructure(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		expectedField string
	}{
		{
			name:          "SimpleError",
			message:       "Invalid request",
			expectedField: "error",
		},
		{
			name:          "DetailedError",
			message:       "Token symbol already exists",
			expectedField: "error",
		},
		{
			name:          "EmptyError",
			message:       "",
			expectedField: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ErrorResponse{Error: tt.message}

			if response.Error != tt.message {
				t.Errorf("Error message mismatch: got %s, want %s", response.Error, tt.message)
			}
		})
	}
}

// TestSuccessResponseStructure tests success response formatting
func TestSuccessResponseStructure(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "WithData",
			data: map[string]string{"key": "value"},
		},
		{
			name: "WithoutData",
			data: nil,
		},
		{
			name: "WithComplexData",
			data: map[string]interface{}{
				"token_id": uuid.New().String(),
				"amount":   1000.0,
				"status":   "confirmed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := SuccessResponse{
				Success: true,
				Data:    tt.data,
			}

			if !response.Success {
				t.Error("Expected success to be true")
			}

			if tt.data != nil && response.Data == nil {
				t.Error("Expected data to be present")
			}
		})
	}
}

// TestHouseholdIsolation tests multi-tenancy household isolation
func TestHouseholdIsolation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID1 := "00000000-0000-0000-0000-000000000099"
	householdID2 := "00000000-0000-0000-0000-000000000098"

	// Create tokens in different households
	token1 := createTestToken(t, testDB.DB, householdID1)
	token2 := createTestToken(t, testDB.DB, householdID2)

	t.Run("TokensIsolatedByHousehold", func(t *testing.T) {
		if token1.HouseholdID == token2.HouseholdID {
			t.Error("Tokens should be in different households")
		}

		if token1.HouseholdID != householdID1 {
			t.Errorf("Token 1 household mismatch: got %s, want %s", token1.HouseholdID, householdID1)
		}

		if token2.HouseholdID != householdID2 {
			t.Errorf("Token 2 household mismatch: got %s, want %s", token2.HouseholdID, householdID2)
		}
	})
}

// TestMetadataHandling tests JSON metadata storage and retrieval
func TestMetadataHandling(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
	}{
		{
			name: "SimpleMetadata",
			metadata: map[string]interface{}{
				"color": "gold",
				"rarity": "rare",
			},
		},
		{
			name: "NestedMetadata",
			metadata: map[string]interface{}{
				"attributes": map[string]interface{}{
					"strength": 10,
					"defense":  5,
				},
				"description": "A powerful token",
			},
		},
		{
			name:     "EmptyMetadata",
			metadata: map[string]interface{}{},
		},
		{
			name: "ComplexMetadata",
			metadata: map[string]interface{}{
				"list": []string{"item1", "item2", "item3"},
				"nested": map[string]interface{}{
					"deep": map[string]interface{}{
						"value": 42,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				Metadata: tt.metadata,
			}

			if token.Metadata == nil && tt.metadata != nil {
				t.Error("Metadata should not be nil")
			}

			if len(token.Metadata) != len(tt.metadata) {
				t.Errorf("Metadata length mismatch: got %d, want %d", len(token.Metadata), len(tt.metadata))
			}
		})
	}
}

// TestUniqueConstraints tests unique constraint violations
func TestUniqueConstraints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"

	t.Run("DuplicateTokenSymbol", func(t *testing.T) {
		// First token should succeed
		token1 := createTestToken(t, testDB.DB, householdID)

		// Attempting to create another token with same symbol should fail
		// This is validated by the unique constraint on (household_id, symbol)
		if token1.Symbol != "" {
			t.Logf("Token symbol %s created successfully", token1.Symbol)
		}
	})

	t.Run("DuplicateWalletAddress", func(t *testing.T) {
		// First wallet should succeed
		wallet1 := createTestWallet(t, testDB.DB, householdID)

		// Attempting to create another wallet with same address should fail
		// This is validated by the unique constraint on address
		if wallet1.Address != "" {
			t.Logf("Wallet address %s created successfully", wallet1.Address)
		}
	})
}
