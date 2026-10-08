package securevalue

import "testing"

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	ciphertext, err := Encrypt(key, "sensitive value")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if ciphertext == "sensitive value" {
		t.Fatal("Encrypt() returned plaintext")
	}

	plaintext, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if plaintext != "sensitive value" {
		t.Fatalf("Decrypt() = %q, want %q", plaintext, "sensitive value")
	}
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	ciphertext, err := Encrypt(key, "sensitive value")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	last := len(ciphertext) - 1
	if ciphertext[last] == 'A' {
		ciphertext = ciphertext[:last] + "B"
	} else {
		ciphertext = ciphertext[:last] + "A"
	}
	if _, err := Decrypt(key, ciphertext); err == nil {
		t.Fatal("Decrypt() succeeded for tampered ciphertext")
	}
}

func TestNilKeyPreservesValue(t *testing.T) {
	const value = "development-only value"
	ciphertext, err := Encrypt(nil, value)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if ciphertext != value {
		t.Fatalf("Encrypt() = %q, want plaintext", ciphertext)
	}
	plaintext, err := Decrypt(nil, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if plaintext != value {
		t.Fatalf("Decrypt() = %q, want %q", plaintext, value)
	}
}
