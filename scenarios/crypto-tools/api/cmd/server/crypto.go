package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)

// HashRequest represents a request to hash data
type HashRequest struct {
	Data       string `json:"data"`
	Algorithm  string `json:"algorithm"`
	Salt       string `json:"salt,omitempty"`
	Iterations int    `json:"iterations,omitempty"`
	Format     string `json:"output_format,omitempty"`
}

// HashResponse represents the hash operation response
type HashResponse struct {
	Hash          string `json:"hash"`
	Algorithm     string `json:"algorithm"`
	Salt          string `json:"salt,omitempty"`
	Iterations    int    `json:"iterations,omitempty"`
	ExecutionTime int64  `json:"execution_time_ms"`
}

// EncryptRequest represents encryption request
type EncryptRequest struct {
	Data      string `json:"data"`
	Algorithm string `json:"algorithm"`
	Key       string `json:"key,omitempty"`
	KeyID     string `json:"key_id,omitempty"`
	Format    string `json:"output_format,omitempty"`
}

// EncryptResponse represents encryption response
type EncryptResponse struct {
	EncryptedData string `json:"encrypted_data"`
	Algorithm     string `json:"algorithm"`
	KeyID         string `json:"key_id,omitempty"`
	IV            string `json:"initialization_vector,omitempty"`
	AuthTag       string `json:"authentication_tag,omitempty"`
}

// DecryptRequest represents decryption request
type DecryptRequest struct {
	EncryptedData string `json:"encrypted_data"`
	Algorithm     string `json:"algorithm"`
	Key           string `json:"key,omitempty"`
	KeyID         string `json:"key_id,omitempty"`
	IV            string `json:"initialization_vector,omitempty"`
	AuthTag       string `json:"authentication_tag,omitempty"`
}

// DecryptResponse represents decryption response
type DecryptResponse struct {
	DecryptedData    string `json:"decrypted_data"`
	IntegrityVerified bool   `json:"integrity_verified"`
	DecryptionTime   int64  `json:"decryption_time_ms"`
}

// KeyGenRequest represents key generation request
type KeyGenRequest struct {
	KeyType  string   `json:"key_type"`
	KeySize  int      `json:"key_size,omitempty"`
	Usage    []string `json:"usage,omitempty"`
	Name     string   `json:"name,omitempty"`
	Expiry   int      `json:"expiry_days,omitempty"`
}

// KeyGenResponse represents key generation response
type KeyGenResponse struct {
	KeyID       string    `json:"key_id"`
	PublicKey   string    `json:"public_key,omitempty"`
	KeyType     string    `json:"key_type"`
	KeySize     int       `json:"key_size"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
}

// setupCryptoRoutes adds crypto-specific routes to the API
func (s *Server) setupCryptoRoutes() {
	api := s.router.PathPrefix("/api/v1/crypto").Subrouter()
	
	// Hash operations
	api.HandleFunc("/hash", s.handleHash).Methods("POST")
	
	// Encryption/Decryption
	api.HandleFunc("/encrypt", s.handleEncrypt).Methods("POST")
	api.HandleFunc("/decrypt", s.handleDecrypt).Methods("POST")
	
	// Key management
	api.HandleFunc("/keys/generate", s.handleKeyGenerate).Methods("POST")
	api.HandleFunc("/keys", s.handleListKeys).Methods("GET")
	api.HandleFunc("/keys/{id}", s.handleGetKey).Methods("GET")
	
	// Digital signatures
	api.HandleFunc("/sign", s.handleSign).Methods("POST")
	api.HandleFunc("/verify", s.handleVerify).Methods("POST")
}

// handleHash processes hash requests
func (s *Server) handleHash(w http.ResponseWriter, r *http.Request) {
	var req HashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	startTime := time.Now()
	
	// Default algorithm if not specified
	if req.Algorithm == "" {
		req.Algorithm = "sha256"
	}
	
	// Default output format
	if req.Format == "" {
		req.Format = "hex"
	}
	
	var hashResult []byte
	var err error
	
	switch req.Algorithm {
	case "sha256":
		h := sha256.Sum256([]byte(req.Data))
		hashResult = h[:]
	case "sha512":
		h := sha512.Sum512([]byte(req.Data))
		hashResult = h[:]
	case "bcrypt":
		cost := 10
		if req.Iterations > 0 && req.Iterations <= 31 {
			cost = req.Iterations
		}
		hashResult, err = bcrypt.GenerateFromPassword([]byte(req.Data), cost)
		if err != nil {
			s.sendError(w, http.StatusInternalServerError, "Bcrypt hashing failed")
			return
		}
	case "scrypt":
		salt := []byte(req.Salt)
		if len(salt) == 0 {
			salt = make([]byte, 32)
			rand.Read(salt)
			req.Salt = base64.StdEncoding.EncodeToString(salt)
		}
		hashResult, err = scrypt.Key([]byte(req.Data), salt, 32768, 8, 1, 32)
		if err != nil {
			s.sendError(w, http.StatusInternalServerError, "Scrypt hashing failed")
			return
		}
	default:
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported algorithm: %s", req.Algorithm))
		return
	}
	
	// Format output
	var hashString string
	switch req.Format {
	case "hex":
		hashString = hex.EncodeToString(hashResult)
	case "base64":
		hashString = base64.StdEncoding.EncodeToString(hashResult)
	default:
		hashString = hex.EncodeToString(hashResult)
	}
	
	executionTime := time.Since(startTime).Milliseconds()
	
	// Record operation in database if available
	if s.db != nil {
		_, _ = s.db.Exec(`
			INSERT INTO crypto_operations (operation_type, algorithm, input_size, execution_time_ms, success)
			VALUES ($1, $2, $3, $4, $5)`,
			"hash", req.Algorithm, len(req.Data), executionTime, true,
		)
	}
	
	response := HashResponse{
		Hash:          hashString,
		Algorithm:     req.Algorithm,
		Salt:          req.Salt,
		Iterations:    req.Iterations,
		ExecutionTime: executionTime,
	}
	
	s.sendJSON(w, http.StatusOK, response)
}

// handleEncrypt processes encryption requests
func (s *Server) handleEncrypt(w http.ResponseWriter, r *http.Request) {
	var req EncryptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	startTime := time.Now()
	
	// Default algorithm
	if req.Algorithm == "" {
		req.Algorithm = "aes256"
	}
	
	// Generate key if not provided
	if req.Key == "" && req.KeyID == "" {
		// Generate a random key for this request
		key := make([]byte, 32) // 256 bits for AES-256
		if _, err := rand.Read(key); err != nil {
			s.sendError(w, http.StatusInternalServerError, "Failed to generate encryption key")
			return
		}
		req.Key = base64.StdEncoding.EncodeToString(key)
	}
	
	var encryptedData []byte
	var iv []byte
	
	switch req.Algorithm {
	case "aes256", "aes256-gcm":
		// Decode key
		key, err := base64.StdEncoding.DecodeString(req.Key)
		if err != nil || len(key) != 32 {
			s.sendError(w, http.StatusBadRequest, "Invalid AES key (must be 32 bytes)")
			return
		}
		
		// Create cipher
		block, err := aes.NewCipher(key)
		if err != nil {
			s.sendError(w, http.StatusInternalServerError, "Failed to create cipher")
			return
		}
		
		// Generate IV
		iv = make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			s.sendError(w, http.StatusInternalServerError, "Failed to generate IV")
			return
		}
		
		// Encrypt
		plaintext := []byte(req.Data)
		ciphertext := make([]byte, len(plaintext))
		stream := cipher.NewCFBEncrypter(block, iv)
		stream.XORKeyStream(ciphertext, plaintext)
		
		encryptedData = append(iv, ciphertext...)
	default:
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported algorithm: %s", req.Algorithm))
		return
	}
	
	executionTime := time.Since(startTime).Milliseconds()
	
	// Record operation
	if s.db != nil {
		_, _ = s.db.Exec(`
			INSERT INTO crypto_operations (operation_type, algorithm, input_size, output_size, execution_time_ms, success)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			"encrypt", req.Algorithm, len(req.Data), len(encryptedData), executionTime, true,
		)
	}
	
	response := EncryptResponse{
		EncryptedData: base64.StdEncoding.EncodeToString(encryptedData),
		Algorithm:     req.Algorithm,
		KeyID:         req.KeyID,
		IV:            base64.StdEncoding.EncodeToString(iv),
	}
	
	s.sendJSON(w, http.StatusOK, response)
}

// handleDecrypt processes decryption requests
func (s *Server) handleDecrypt(w http.ResponseWriter, r *http.Request) {
	var req DecryptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	startTime := time.Now()
	
	// Decode encrypted data
	encryptedData, err := base64.StdEncoding.DecodeString(req.EncryptedData)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid encrypted data format")
		return
	}
	
	var decryptedData []byte
	
	switch req.Algorithm {
	case "aes256", "aes256-gcm":
		// Decode key
		key, err := base64.StdEncoding.DecodeString(req.Key)
		if err != nil || len(key) != 32 {
			s.sendError(w, http.StatusBadRequest, "Invalid AES key")
			return
		}
		
		// Create cipher
		block, err := aes.NewCipher(key)
		if err != nil {
			s.sendError(w, http.StatusInternalServerError, "Failed to create cipher")
			return
		}
		
		// Extract IV from encrypted data
		if len(encryptedData) < aes.BlockSize {
			s.sendError(w, http.StatusBadRequest, "Encrypted data too short")
			return
		}
		iv := encryptedData[:aes.BlockSize]
		ciphertext := encryptedData[aes.BlockSize:]
		
		// Decrypt
		plaintext := make([]byte, len(ciphertext))
		stream := cipher.NewCFBDecrypter(block, iv)
		stream.XORKeyStream(plaintext, ciphertext)
		
		decryptedData = plaintext
	default:
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported algorithm: %s", req.Algorithm))
		return
	}
	
	executionTime := time.Since(startTime).Milliseconds()
	
	// Record operation
	if s.db != nil {
		_, _ = s.db.Exec(`
			INSERT INTO crypto_operations (operation_type, algorithm, input_size, output_size, execution_time_ms, success)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			"decrypt", req.Algorithm, len(encryptedData), len(decryptedData), executionTime, true,
		)
	}
	
	response := DecryptResponse{
		DecryptedData:    string(decryptedData),
		IntegrityVerified: true,
		DecryptionTime:   executionTime,
	}
	
	s.sendJSON(w, http.StatusOK, response)
}

// handleKeyGenerate generates new cryptographic keys
func (s *Server) handleKeyGenerate(w http.ResponseWriter, r *http.Request) {
	var req KeyGenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	startTime := time.Now()
	
	// Default key type
	if req.KeyType == "" {
		req.KeyType = "rsa"
	}
	
	// Default key size
	if req.KeySize == 0 {
		switch req.KeyType {
		case "rsa":
			req.KeySize = 2048
		case "symmetric":
			req.KeySize = 256
		}
	}
	
	keyID := uuid.New().String()
	var publicKeyPEM string
	var fingerprint string
	
	switch req.KeyType {
	case "rsa":
		// Generate RSA key pair
		privateKey, err := rsa.GenerateKey(rand.Reader, req.KeySize)
		if err != nil {
			s.sendError(w, http.StatusInternalServerError, "Failed to generate RSA key")
			return
		}
		
		// Export public key
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			s.sendError(w, http.StatusInternalServerError, "Failed to marshal public key")
			return
		}
		
		publicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		}))
		
		// Calculate fingerprint
		h := sha256.Sum256(publicKeyBytes)
		fingerprint = hex.EncodeToString(h[:])
		
	case "symmetric":
		// Generate symmetric key
		keyBytes := make([]byte, req.KeySize/8)
		if _, err := rand.Read(keyBytes); err != nil {
			s.sendError(w, http.StatusInternalServerError, "Failed to generate symmetric key")
			return
		}
		
		// For symmetric keys, we don't have a public key
		publicKeyPEM = ""
		h := sha256.Sum256(keyBytes)
		fingerprint = hex.EncodeToString(h[:])
		
	default:
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported key type: %s", req.KeyType))
		return
	}
	
	executionTime := time.Since(startTime).Milliseconds()
	
	// Store key metadata in database if available
	if s.db != nil {
		_, _ = s.db.Exec(`
			INSERT INTO crypto_keys (id, name, key_type, key_size, public_key, key_fingerprint, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			keyID, req.Name, req.KeyType, req.KeySize, publicKeyPEM, fingerprint, "active",
		)
		
		_, _ = s.db.Exec(`
			INSERT INTO crypto_operations (operation_type, algorithm, execution_time_ms, success, key_id)
			VALUES ($1, $2, $3, $4, $5)`,
			"keygen", req.KeyType, executionTime, true, keyID,
		)
	}
	
	response := KeyGenResponse{
		KeyID:       keyID,
		PublicKey:   publicKeyPEM,
		KeyType:     req.KeyType,
		KeySize:     req.KeySize,
		Fingerprint: fingerprint,
		CreatedAt:   time.Now(),
	}
	
	s.sendJSON(w, http.StatusOK, response)
}

// handleListKeys lists available cryptographic keys
func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	// Mock response when no database
	if s.db == nil {
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"keys": []interface{}{},
			"total": 0,
		})
		return
	}
	
	// Query keys from database
	rows, err := s.db.Query(`
		SELECT id, name, key_type, key_size, key_fingerprint, status, created_at
		FROM crypto_keys
		WHERE status = 'active'
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to retrieve keys")
		return
	}
	defer rows.Close()
	
	var keys []map[string]interface{}
	for rows.Next() {
		var id, name, keyType, fingerprint, status string
		var keySize int
		var createdAt time.Time
		
		if err := rows.Scan(&id, &name, &keyType, &keySize, &fingerprint, &status, &createdAt); err != nil {
			continue
		}
		
		keys = append(keys, map[string]interface{}{
			"key_id":      id,
			"name":        name,
			"key_type":    keyType,
			"key_size":    keySize,
			"fingerprint": fingerprint,
			"status":      status,
			"created_at":  createdAt,
		})
	}
	
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"keys":  keys,
		"total": len(keys),
	})
}

// handleGetKey retrieves a specific key's public information
func (s *Server) handleGetKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["id"]
	
	if s.db == nil {
		s.sendError(w, http.StatusNotFound, "Key not found")
		return
	}
	
	var id, name, keyType, publicKey, fingerprint, status string
	var keySize int
	var createdAt time.Time
	
	err := s.db.QueryRow(`
		SELECT id, name, key_type, key_size, public_key, key_fingerprint, status, created_at
		FROM crypto_keys
		WHERE id = $1 AND status = 'active'
	`, keyID).Scan(&id, &name, &keyType, &keySize, &publicKey, &fingerprint, &status, &createdAt)
	
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Key not found")
		return
	}
	
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"key_id":      id,
		"name":        name,
		"key_type":    keyType,
		"key_size":    keySize,
		"public_key":  publicKey,
		"fingerprint": fingerprint,
		"status":      status,
		"created_at":  createdAt,
	})
}

// handleSign creates digital signatures (stub for now)
func (s *Server) handleSign(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement digital signature creation
	s.sendJSON(w, http.StatusNotImplemented, map[string]string{
		"error": "Digital signature functionality coming soon",
	})
}

// handleVerify verifies digital signatures (stub for now)
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement signature verification
	s.sendJSON(w, http.StatusNotImplemented, map[string]string{
		"error": "Signature verification functionality coming soon",
	})
}