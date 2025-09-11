package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"scenario-authenticator/models"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

// LoadJWTKeys loads or generates JWT signing keys
func LoadJWTKeys() error {
	// Try to load existing keys
	privateKeyPath := "../data/keys/private.pem"
	publicKeyPath := "../data/keys/public.pem"
	
	// Try to load private key
	privateKeyData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Println("No existing JWT keys found, generating new ones...")
		return GenerateJWTKeys()
	}
	
	// Parse private key
	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return fmt.Errorf("failed to parse private key PEM block")
	}
	
	privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("not an RSA private key")
		}
	}
	
	// Load public key
	publicKeyData, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		// Generate public key from private key
		publicKey = &privateKey.PublicKey
		log.Println("Generated public key from private key")
	} else {
		// Parse public key
		block, _ := pem.Decode(publicKeyData)
		if block == nil {
			return fmt.Errorf("failed to parse public key PEM block")
		}
		
		pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
		
		var ok bool
		publicKey, ok = pubInterface.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("not an RSA public key")
		}
	}
	
	log.Println("JWT keys loaded successfully")
	return nil
}

// GenerateJWTKeys generates new RSA key pair for JWT signing
func GenerateJWTKeys() error {
	// Generate RSA key pair
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}
	
	publicKey = &privateKey.PublicKey
	
	// Save keys to files (optional, for persistence)
	// In production, you might want to store these securely
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	
	// Try to save keys (non-critical if it fails)
	if err := ioutil.WriteFile("../data/keys/private.pem", pem.EncodeToMemory(privateKeyPEM), 0600); err != nil {
		log.Printf("Warning: Could not save private key: %v", err)
	}
	
	if err := ioutil.WriteFile("../data/keys/public.pem", pem.EncodeToMemory(publicKeyPEM), 0644); err != nil {
		log.Printf("Warning: Could not save public key: %v", err)
	}
	
	log.Println("Generated new JWT keys successfully")
	return nil
}

// GenerateToken generates a JWT token for a user
func GenerateToken(user *models.User) (string, error) {
	claims := &models.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  user.Roles,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "scenario-authenticator",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*models.Claims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token")
}

// GenerateRefreshToken generates a secure random refresh token
func GenerateRefreshToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// HashToken creates a SHA256 hash of a token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}