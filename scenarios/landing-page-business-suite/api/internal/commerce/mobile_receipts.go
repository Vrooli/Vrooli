package commerce

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// AppleSignedTransactionValidator verifies StoreKit's ES256 JWS transaction
// with the configured Apple root certificate before it enters commerce. The
// account-token resolver binds the signed transaction to the Vrooli identity;
// a caller-supplied email is never treated as proof of ownership.
type AppleSignedTransactionValidator struct {
	RootCertificate  *x509.Certificate
	ExpectedBundleID string
	ProductPlans     map[string]string
	ResolveIdentity  func(context.Context, string) (string, error)
	Now              func() time.Time
}

type appleJWSHeader struct {
	Algorithm        string   `json:"alg"`
	CertificateChain []string `json:"x5c"`
}

type appleSignedTransaction struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	BundleID              string `json:"bundleId"`
	ProductID             string `json:"productId"`
	AppAccountToken       string `json:"appAccountToken"`
	ExpiresDate           int64  `json:"expiresDate"`
	RevocationDate        int64  `json:"revocationDate"`
}

// Validate verifies the signed transaction, product mapping, expiry, and
// account binding. Apple certificate material is intentionally configuration,
// not a client-provided value.
func (v AppleSignedTransactionValidator) Validate(ctx context.Context, receipt Receipt) (NormalizedSubscription, error) {
	if strings.ToLower(strings.TrimSpace(receipt.Source)) != "apple" || strings.TrimSpace(receipt.Token) == "" || v.RootCertificate == nil || v.ResolveIdentity == nil {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	header, payload, err := verifyAppleJWS(receipt.Token, v.RootCertificate)
	if err != nil || header.Algorithm != "ES256" {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	var transaction appleSignedTransaction
	if json.Unmarshal(payload, &transaction) != nil || transaction.TransactionID == "" || transaction.OriginalTransactionID == "" || transaction.ProductID == "" || transaction.AppAccountToken == "" {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	if v.ExpectedBundleID != "" && transaction.BundleID != v.ExpectedBundleID {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	plan, ok := v.ProductPlans[transaction.ProductID]
	if !ok {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	identity, err := v.ResolveIdentity(ctx, transaction.AppAccountToken)
	if err != nil || !strings.EqualFold(strings.TrimSpace(identity), strings.TrimSpace(receipt.UserIdentity)) {
		return NormalizedSubscription{}, ErrReceiptBound
	}
	now := time.Now().UTC()
	if v.Now != nil {
		now = v.Now().UTC()
	}
	if transaction.ExpiresDate <= now.UnixMilli() || transaction.RevocationDate > 0 {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	return NormalizedSubscription{
		SubscriptionID:       transaction.OriginalTransactionID,
		ExternalSubscription: transaction.TransactionID,
		UserIdentity:         receipt.UserIdentity,
		Status:               "active",
		PlanTier:             plan,
		PriceID:              transaction.ProductID,
		BundleKey:            "business_suite",
	}, nil
}

func verifyAppleJWS(raw string, root *x509.Certificate) (appleJWSHeader, []byte, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return appleJWSHeader{}, nil, ErrReceiptInvalid
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return appleJWSHeader{}, nil, err
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return appleJWSHeader{}, nil, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != 64 {
		return appleJWSHeader{}, nil, ErrReceiptInvalid
	}
	var header appleJWSHeader
	if json.Unmarshal(headerBytes, &header) != nil || len(header.CertificateChain) == 0 {
		return appleJWSHeader{}, nil, ErrReceiptInvalid
	}
	certs := make([]*x509.Certificate, 0, len(header.CertificateChain))
	for _, encoded := range header.CertificateChain {
		der, decodeErr := base64.StdEncoding.DecodeString(encoded)
		if decodeErr != nil {
			return appleJWSHeader{}, nil, ErrReceiptInvalid
		}
		cert, parseErr := x509.ParseCertificate(der)
		if parseErr != nil {
			return appleJWSHeader{}, nil, ErrReceiptInvalid
		}
		certs = append(certs, cert)
	}
	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}
	if _, err := certs[0].Verify(x509.VerifyOptions{Roots: certPool(root), Intermediates: intermediates, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}); err != nil {
		return appleJWSHeader{}, nil, ErrReceiptInvalid
	}
	publicKey, ok := certs[0].PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return appleJWSHeader{}, nil, ErrReceiptInvalid
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.VerifyASN1(publicKey, digest[:], joseECDSASignatureToASN1(signature)) {
		return appleJWSHeader{}, nil, ErrReceiptInvalid
	}
	return header, payload, nil
}

func certPool(root *x509.Certificate) *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(root)
	return pool
}

// joseECDSASignatureToASN1 converts the fixed-width JWS R||S form to the
// ASN.1 form accepted by crypto/ecdsa. The implementation uses crypto.Signer
// verification helpers only; no private key or platform secret is involved.
func joseECDSASignatureToASN1(raw []byte) []byte {
	// Apple JWS uses P-256 and the standard library's ASN.1 parser is kept
	// private, so use a minimal DER encoder for two positive INTEGERs.
	encodeInt := func(value []byte) []byte {
		for len(value) > 1 && value[0] == 0 {
			value = value[1:]
		}
		if value[0]&0x80 != 0 {
			value = append([]byte{0}, value...)
		}
		return append([]byte{0x02, byte(len(value))}, value...)
	}
	r, s := encodeInt(raw[:32]), encodeInt(raw[32:])
	return append(append([]byte{0x30, byte(len(r) + len(s))}, r...), s...)
}

// GooglePlayDeveloperValidator validates a Play purchase token with Google's
// Android Publisher API. OAuth token acquisition is injected so service
// account credentials remain server-only and never enter a scenario client.
type GooglePlayDeveloperValidator struct {
	PackageName     string
	ProductID       string
	OAuthToken      func(context.Context) (string, error)
	ResolveIdentity func(context.Context, string) (string, error)
	Client          *http.Client
	Now             func() time.Time
}

type googleProductPurchase struct {
	PurchaseState               int    `json:"purchaseState"`
	AcknowledgementState        int    `json:"acknowledgementState"`
	OrderID                     string `json:"orderId"`
	ProductID                   string `json:"productId"`
	PurchaseToken               string `json:"purchaseToken"`
	ObfuscatedExternalAccountID string `json:"obfuscatedExternalAccountId"`
	ExpiryTimeMillis            string `json:"expiryTimeMillis"`
}

// Validate performs the server-to-server purchase lookup and normalizes the
// verified result. A pending, refunded, expired, or unbound purchase fails.
func (v GooglePlayDeveloperValidator) Validate(ctx context.Context, receipt Receipt) (NormalizedSubscription, error) {
	if strings.ToLower(strings.TrimSpace(receipt.Source)) != "google" || strings.TrimSpace(receipt.Token) == "" || v.PackageName == "" || v.ProductID == "" || v.OAuthToken == nil || v.ResolveIdentity == nil {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	token, err := v.OAuthToken(ctx)
	if err != nil {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	endpoint := "https://androidpublisher.googleapis.com/androidpublisher/v3/applications/" + url.PathEscape(v.PackageName) + "/purchases/subscriptions/" + url.PathEscape(v.ProductID) + "/tokens/" + url.PathEscape(receipt.Token)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	request.Header.Set("Authorization", "Bearer "+token)
	client := v.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	var purchase googleProductPurchase
	if json.Unmarshal(body, &purchase) != nil || purchase.PurchaseState != 0 || purchase.OrderID == "" || purchase.PurchaseToken != receipt.Token || purchase.ProductID != "" && purchase.ProductID != v.ProductID || purchase.ObfuscatedExternalAccountID == "" {
		return NormalizedSubscription{}, ErrReceiptInvalid
	}
	identity, err := v.ResolveIdentity(ctx, purchase.ObfuscatedExternalAccountID)
	if err != nil || !strings.EqualFold(strings.TrimSpace(identity), strings.TrimSpace(receipt.UserIdentity)) {
		return NormalizedSubscription{}, ErrReceiptBound
	}
	if purchase.ExpiryTimeMillis != "" {
		expires, parseErr := strconv.ParseInt(purchase.ExpiryTimeMillis, 10, 64)
		if parseErr != nil || !time.UnixMilli(expires).After(v.now()) {
			return NormalizedSubscription{}, ErrReceiptInvalid
		}
	}
	return NormalizedSubscription{SubscriptionID: purchase.OrderID, ExternalSubscription: purchase.PurchaseToken, UserIdentity: receipt.UserIdentity, Status: "active", PlanTier: "pro", PriceID: v.ProductID, BundleKey: "business_suite"}, nil
}

func (v GooglePlayDeveloperValidator) now() time.Time {
	if v.Now != nil {
		return v.Now().UTC()
	}
	return time.Now().UTC()
}
