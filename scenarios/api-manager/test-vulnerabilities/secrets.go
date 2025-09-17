package testvulns

import (
	"crypto/md5"
	"fmt"
	"net/http"
)

const (
	// VULNERABILITY: Hardcoded AWS credentials (CWE-798)
	AWSAccessKey = "AKIAIOSFODNN7EXAMPLE"
	AWSSecretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	
	// VULNERABILITY: Hardcoded API key
	APIKey = "sk_live_4eC39HqLyjWDarjtT1zdp7dc"
	
	// VULNERABILITY: Hardcoded password
	DatabasePassword = "admin123!@#"
	
	// VULNERABILITY: JWT Secret
	JWTSecret = "my-super-secret-jwt-key-123"
)

// WeakCryptoExample demonstrates weak cryptography
func WeakCryptoExample() {
	// VULNERABILITY: Use of MD5 (CWE-328)
	hash := md5.New()
	hash.Write([]byte("password"))
	fmt.Printf("%x", hash.Sum(nil))
}

// HardcodedCredsExample shows hardcoded credentials in code
func HardcodedCredsExample() {
	// VULNERABILITY: Hardcoded database connection string
	connectionString := "postgres://admin:password123@localhost:5432/mydb"
	
	// VULNERABILITY: Hardcoded GitHub token
	githubToken := "ghp_16CharactersSecretsTokenGithub12345678"
	
	// VULNERABILITY: Private key in code
	privateKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAz8XKuLscVBMsjqMUn6kDDanNxc5mMWLSaUl5LFrJbGkd0Mau
fake-private-key-content-for-testing
-----END RSA PRIVATE KEY-----`
	
	_ = connectionString
	_ = githubToken
	_ = privateKey
}

// InsecureRandomExample shows weak random number generation
func InsecureRandomExample() {
	// VULNERABILITY: math/rand instead of crypto/rand (CWE-338)
	// math.rand.Intn(100) // Would be flagged by gosec
}

// ErrorHandlingExample shows poor error handling
func ErrorHandlingExample() error {
	// VULNERABILITY: Ignored error (CWE-391)
	resp, _ := http.Get("https://example.com")
	// VULNERABILITY: Missing defer resp.Body.Close() - resource leak
	_ = resp
	return nil
}