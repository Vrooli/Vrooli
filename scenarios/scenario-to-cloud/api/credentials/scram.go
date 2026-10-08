package credentials

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

// scramIterations matches PostgreSQL's default for SCRAM-SHA-256.
const scramIterations = 4096

// ScramSHA256Verifier renders the PostgreSQL SCRAM-SHA-256 password verifier
// for a plaintext value. The database role is prepared with the verifier, so
// the plaintext never travels to the database host, never enters its
// statement log and never appears in the argv of the owner command.
func ScramSHA256Verifier(password string, salt []byte) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password is required")
	}
	if len(salt) == 0 {
		salt = make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return "", err
		}
	}
	salted := pbkdf2.Key([]byte(password), salt, scramIterations, sha256.Size, sha256.New)
	clientKey := hmacSHA256(salted, []byte("Client Key"))
	storedKey := sha256.Sum256(clientKey)
	serverKey := hmacSHA256(salted, []byte("Server Key"))
	return fmt.Sprintf("SCRAM-SHA-256$%d:%s$%s:%s", scramIterations,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(storedKey[:]),
		base64.StdEncoding.EncodeToString(serverKey)), nil
}

func hmacSHA256(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}
