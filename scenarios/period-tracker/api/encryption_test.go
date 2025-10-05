package main

import (
	"testing"
)

// TestDeriveKey tests key derivation function
func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name     string
		password string
		salt     []byte
		wantLen  int
	}{
		{
			name:     "Standard key derivation",
			password: "test-password",
			salt:     []byte("test-salt-12345678901234"),
			wantLen:  32,
		},
		{
			name:     "Empty password",
			password: "",
			salt:     []byte("test-salt-12345678901234"),
			wantLen:  32,
		},
		{
			name:     "Long password",
			password: "this-is-a-very-long-password-for-encryption-key-derivation",
			salt:     []byte("test-salt-12345678901234"),
			wantLen:  32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := deriveKey(tt.password, tt.salt)
			if len(key) != tt.wantLen {
				t.Errorf("deriveKey() key length = %d, want %d", len(key), tt.wantLen)
			}
		})
	}
}

// TestEncryptDecrypt tests encryption and decryption
func TestEncryptDecrypt(t *testing.T) {
	// Setup encryption key
	salt := []byte("period-tracker-salt-2024")
	encryptionKey = deriveKey("test-key", salt)

	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "Simple text",
			plaintext: "Hello, World!",
			wantErr:   false,
		},
		{
			name:      "Empty string",
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "Long text",
			plaintext: "This is a very long piece of sensitive health information that needs to be encrypted securely for privacy protection",
			wantErr:   false,
		},
		{
			name:      "Special characters",
			plaintext: "Test with sp3c!@l ch@r@ct3rs & émojis 🔒",
			wantErr:   false,
		},
		{
			name:      "JSON data",
			plaintext: `{"symptoms": ["headache", "cramps"], "intensity": 7}`,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryptString
			encrypted, err := encryptString(tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("encryptString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.plaintext == "" {
				if encrypted != "" {
					t.Error("Empty string should result in empty encrypted string")
				}
				return
			}

			if encrypted == "" && tt.plaintext != "" {
				t.Error("Encrypted string should not be empty for non-empty input")
				return
			}

			if encrypted == tt.plaintext {
				t.Error("Encrypted string should differ from plaintext")
				return
			}

			// Test decryptString
			decrypted, err := decryptString(encrypted)
			if err != nil {
				t.Errorf("decryptString() error = %v", err)
				return
			}

			if decrypted != tt.plaintext {
				t.Errorf("decryptString() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

// TestEncryptDecryptBytes tests byte-level encryption
func TestEncryptDecryptBytes(t *testing.T) {
	salt := []byte("period-tracker-salt-2024")
	key := deriveKey("test-key", salt)

	tests := []struct {
		name      string
		plaintext []byte
		wantErr   bool
	}{
		{
			name:      "Simple bytes",
			plaintext: []byte("test data"),
			wantErr:   false,
		},
		{
			name:      "Binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encrypt(tt.plaintext, key)
			if (err != nil) != tt.wantErr {
				t.Errorf("encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(encrypted) == 0 {
				t.Error("Encrypted data should not be empty")
				return
			}

			decrypted, err := decrypt(encrypted, key)
			if err != nil {
				t.Errorf("decrypt() error = %v", err)
				return
			}

			if string(decrypted) != string(tt.plaintext) {
				t.Errorf("decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

// TestDecryptInvalidData tests decryption error handling
func TestDecryptInvalidData(t *testing.T) {
	salt := []byte("period-tracker-salt-2024")
	encryptionKey = deriveKey("test-key", salt)

	tests := []struct {
		name       string
		ciphertext string
		wantErr    bool
	}{
		{
			name:       "Invalid base64",
			ciphertext: "not-valid-base64-!@#$%",
			wantErr:    true,
		},
		{
			name:       "Empty ciphertext",
			ciphertext: "",
			wantErr:    false, // Empty is handled specially
		},
		{
			name:       "Short ciphertext",
			ciphertext: "dGVzdA==", // "test" in base64, too short for GCM
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decryptString(tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("decryptString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEncryptionConsistency verifies multiple encryptions produce different results
func TestEncryptionConsistency(t *testing.T) {
	salt := []byte("period-tracker-salt-2024")
	encryptionKey = deriveKey("test-key", salt)

	plaintext := "test data for consistency"

	encrypted1, err1 := encryptString(plaintext)
	encrypted2, err2 := encryptString(plaintext)

	if err1 != nil || err2 != nil {
		t.Fatalf("Encryption failed: err1=%v, err2=%v", err1, err2)
	}

	// Encrypted values should differ due to random nonce
	if encrypted1 == encrypted2 {
		t.Error("Multiple encryptions of same data should produce different ciphertexts (due to random nonce)")
	}

	// But both should decrypt to same value
	decrypted1, err1 := decryptString(encrypted1)
	decrypted2, err2 := decryptString(encrypted2)

	if err1 != nil || err2 != nil {
		t.Fatalf("Decryption failed: err1=%v, err2=%v", err1, err2)
	}

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both encrypted values should decrypt to original plaintext")
	}
}

// BenchmarkEncryption benchmarks encryption performance
func BenchmarkEncryption(b *testing.B) {
	salt := []byte("period-tracker-salt-2024")
	encryptionKey = deriveKey("test-key", salt)
	testData := "Sensitive health data that needs encryption"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encryptString(testData)
		if err != nil {
			b.Fatalf("Encryption failed: %v", err)
		}
	}
}

// BenchmarkDecryption benchmarks decryption performance
func BenchmarkDecryption(b *testing.B) {
	salt := []byte("period-tracker-salt-2024")
	encryptionKey = deriveKey("test-key", salt)
	testData := "Sensitive health data that needs encryption"

	encrypted, err := encryptString(testData)
	if err != nil {
		b.Fatalf("Setup encryption failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decryptString(encrypted)
		if err != nil {
			b.Fatalf("Decryption failed: %v", err)
		}
	}
}

// BenchmarkKeyDerivation benchmarks key derivation performance
func BenchmarkKeyDerivation(b *testing.B) {
	password := "test-password"
	salt := []byte("period-tracker-salt-2024")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deriveKey(password, salt)
	}
}
