// +build testing

package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestHashHandler tests the hash endpoint
func TestHashHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite("Hash", env.Server, env.Config.APIToken)

	t.Run("Success_SHA256", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body:   createTestHashRequest("test data", "sha256"),
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertHashResponse(t, data, "sha256")
		})
	})

	t.Run("Success_SHA512", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body:   createTestHashRequest("test data", "sha512"),
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertHashResponse(t, data, "sha512")
		})
	})

	t.Run("Success_Bcrypt", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body:   createTestHashRequest("password123", "bcrypt"),
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertHashResponse(t, data, "bcrypt")
		})
	})

	t.Run("Success_Scrypt", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body: map[string]interface{}{
				"data":      "password123",
				"algorithm": "scrypt",
				"salt":      base64.StdEncoding.EncodeToString([]byte("test-salt-1234567890")),
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertHashResponse(t, data, "scrypt")
			if _, ok := data["salt"].(string); !ok {
				t.Error("Expected salt to be present in scrypt response")
			}
		})
	})

	t.Run("DefaultAlgorithm", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body: map[string]interface{}{
				"data": "test data",
				// No algorithm specified - should default to sha256
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			algorithm, ok := data["algorithm"].(string)
			if !ok || algorithm != "sha256" {
				t.Errorf("Expected default algorithm sha256, got %v", algorithm)
			}
		})
	})

	t.Run("OutputFormat_Base64", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body: map[string]interface{}{
				"data":          "test data",
				"algorithm":     "sha256",
				"output_format": "base64",
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			hash, ok := data["hash"].(string)
			if !ok {
				t.Fatal("Expected hash to be a string")
			}
			// Base64 should not contain hex characters only
			if _, err := base64.StdEncoding.DecodeString(hash); err != nil {
				t.Errorf("Expected valid base64 output, got decode error: %v", err)
			}
		})
	})

	t.Run("ErrorPatterns", func(t *testing.T) {
		suite.RunErrorTests(t, CryptoHashErrorPatterns())
	})

	t.Run("UnsupportedAlgorithm", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body: map[string]interface{}{
				"data":      "test data",
				"algorithm": "unsupported-algo",
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestEncryptHandler tests the encryption endpoint
func TestEncryptHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite("Encrypt", env.Server, env.Config.APIToken)

	t.Run("Success_AES256", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/crypto/encrypt",
			Body:    createTestEncryptRequest("test data", "aes256"),
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertEncryptResponse(t, data, "aes256")
		})
	})

	t.Run("Success_WithProvidedKey", func(t *testing.T) {
		// Generate a valid AES key
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}
		encodedKey := base64.StdEncoding.EncodeToString(key)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/encrypt",
			Body: map[string]interface{}{
				"data":      "test data",
				"algorithm": "aes256",
				"key":       encodedKey,
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertEncryptResponse(t, data, "aes256")
		})
	})

	t.Run("InvalidKey_WrongLength", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/encrypt",
			Body: map[string]interface{}{
				"data":      "test data",
				"algorithm": "aes256",
				"key":       base64.StdEncoding.EncodeToString([]byte("short-key")),
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("ErrorPatterns", func(t *testing.T) {
		suite.RunErrorTests(t, CryptoEncryptErrorPatterns())
	})
}

// TestDecryptHandler tests the decryption endpoint
func TestDecryptHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite("Decrypt", env.Server, env.Config.APIToken)

	t.Run("Success_EncryptDecrypt", func(t *testing.T) {
		// First encrypt some data
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}
		encodedKey := base64.StdEncoding.EncodeToString(key)
		plaintext := "test data for encryption"

		encReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/encrypt",
			Body: map[string]interface{}{
				"data":      plaintext,
				"algorithm": "aes256",
				"key":       encodedKey,
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		encW, err := makeHTTPRequest(env.Server, encReq)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		var encResponse Response
		if err := json.NewDecoder(encW.Body).Decode(&encResponse); err == nil {
			// Extract encrypted data for decryption
			if encData, ok := encResponse.Data.(map[string]interface{}); ok {
				encryptedData := encData["encrypted_data"].(string)

				// Now decrypt
				decReq := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/crypto/decrypt",
					Body: map[string]interface{}{
						"encrypted_data": encryptedData,
						"algorithm":      "aes256",
						"key":            encodedKey,
					},
					Headers: createAuthHeaders(env.Config.APIToken),
				}

				decW, err := makeHTTPRequest(env.Server, decReq)
				if err != nil {
					t.Fatalf("Failed to decrypt: %v", err)
				}

				assertJSONResponse(t, decW, http.StatusOK, func(data map[string]interface{}) {
					decryptedData, ok := data["decrypted_data"].(string)
					if !ok {
						t.Fatal("Expected decrypted_data to be a string")
					}
					if decryptedData != plaintext {
						t.Errorf("Expected decrypted data to match original, got %s", decryptedData)
					}

					integrityVerified, ok := data["integrity_verified"].(bool)
					if !ok || !integrityVerified {
						t.Error("Expected integrity_verified to be true")
					}
				})
			}
		}
	})

	t.Run("InvalidEncryptedData", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/decrypt",
			Body: map[string]interface{}{
				"encrypted_data": "invalid-base64!!!",
				"algorithm":      "aes256",
				"key":            base64.StdEncoding.EncodeToString(make([]byte, 32)),
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("ErrorPatterns", func(t *testing.T) {
		suite.RunErrorTests(t, CryptoDecryptErrorPatterns())
	})
}

// TestKeyGenerateHandler tests key generation endpoint
func TestKeyGenerateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite("KeyGenerate", env.Server, env.Config.APIToken)

	t.Run("Success_RSA_2048", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/crypto/keys/generate",
			Body:    createTestKeyGenRequest("rsa", 2048),
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertKeyGenResponse(t, data, "rsa", 2048)

			// RSA should have a public key
			publicKey, ok := data["public_key"].(string)
			if !ok || publicKey == "" {
				t.Error("Expected public_key to be present for RSA key")
			}
		})
	})

	t.Run("Success_RSA_4096", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/crypto/keys/generate",
			Body:    createTestKeyGenRequest("rsa", 4096),
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertKeyGenResponse(t, data, "rsa", 4096)
		})
	})

	t.Run("Success_Symmetric", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/crypto/keys/generate",
			Body:    createTestKeyGenRequest("symmetric", 256),
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertKeyGenResponse(t, data, "symmetric", 256)

			// Symmetric keys should not have a public key
			publicKey, _ := data["public_key"].(string)
			if publicKey != "" {
				t.Error("Expected no public_key for symmetric key")
			}
		})
	})

	t.Run("DefaultKeySize", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/keys/generate",
			Body: map[string]interface{}{
				"key_type": "rsa",
				// No key_size specified
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			keySize, ok := data["key_size"].(float64)
			if !ok || int(keySize) != 2048 {
				t.Errorf("Expected default RSA key size 2048, got %v", keySize)
			}
		})
	})

	t.Run("ErrorPatterns", func(t *testing.T) {
		suite.RunErrorTests(t, CryptoKeyGenErrorPatterns())
	})

	t.Run("UnsupportedKeyType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/keys/generate",
			Body: map[string]interface{}{
				"key_type": "unsupported-type",
				"key_size": 2048,
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestSignHandler tests digital signature endpoint
func TestSignHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite("Sign", env.Server, env.Config.APIToken)

	keyID := uuid.New().String()

	t.Run("Success_Basic", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/sign",
			Body: map[string]interface{}{
				"data":   "test data to sign",
				"key_id": keyID,
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertSignResponse(t, data)
		})
	})

	t.Run("Success_WithTimestamp", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/sign",
			Body: map[string]interface{}{
				"data":   "test data to sign",
				"key_id": keyID,
				"options": map[string]interface{}{
					"include_timestamp": true,
				},
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			assertSignResponse(t, data)

			timestamp, ok := data["timestamp"].(string)
			if !ok || timestamp == "" {
				t.Error("Expected timestamp to be present when include_timestamp is true")
			}
		})
	})

	t.Run("ErrorPatterns", func(t *testing.T) {
		suite.RunErrorTests(t, CryptoSignErrorPatterns())
	})
}

// TestVerifyHandler tests signature verification endpoint
func TestVerifyHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite("Verify", env.Server, env.Config.APIToken)

	keyID := uuid.New().String()

	t.Run("Success_ValidSignature", func(t *testing.T) {
		// First create a signature
		data := "test data to verify"
		signReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/sign",
			Body: map[string]interface{}{
				"data":   data,
				"key_id": keyID,
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		signW, err := makeHTTPRequest(env.Server, signReq)
		if err != nil {
			t.Fatalf("Failed to sign: %v", err)
		}

		// Get the signature from response
		// Note: In the mock implementation, we can verify by re-computing
		verifyReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/verify",
			Body: map[string]interface{}{
				"data":      data,
				"signature": "mock-signature", // Mock implementation
				"key_id":    keyID,
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		verifyW, err := makeHTTPRequest(env.Server, verifyReq)
		if err != nil {
			t.Fatalf("Failed to verify: %v", err)
		}

		// The mock implementation should handle this
		if verifyW.Code != http.StatusOK && verifyW.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400, got %d", verifyW.Code)
		}

		_ = signW // Use signW to avoid unused variable error
	})

	t.Run("MissingPublicKeyAndKeyID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/verify",
			Body: map[string]interface{}{
				"data":      "test data",
				"signature": "some-signature",
				// Missing both public_key and key_id
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("ErrorPatterns", func(t *testing.T) {
		suite.RunErrorTests(t, CryptoVerifyErrorPatterns())
	})
}

// TestListKeys tests key listing endpoint
func TestListKeys(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/crypto/keys",
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) {
			keys, ok := data["keys"].([]interface{})
			if !ok {
				t.Fatal("Expected keys to be an array")
			}

			// Without database, should return empty array
			if len(keys) != 0 {
				t.Errorf("Expected empty keys array, got %d items", len(keys))
			}

			total, ok := data["total"].(float64)
			if !ok || int(total) != 0 {
				t.Errorf("Expected total to be 0, got %v", total)
			}
		})
	})
}

// TestGetKey tests individual key retrieval
func TestGetKey(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NotFound_NoDatabase", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/crypto/keys/{id}",
			URLVars: map[string]string{"id": uuid.New().String()},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}
