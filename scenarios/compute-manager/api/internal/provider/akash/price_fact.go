package akash

// This boundary deliberately keeps an Akash bid in its chain denomination and
// per-block period until every normalization input has authoritative provenance.
// The shared spend catalogue accepts USD monthly minor units only.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	cmprovider "compute-manager/internal/provider"
)

type PriceFact struct {
	Network, ChainID, DSeq, Provider, Region, OfferID string
	GSeq, OSeq                                        uint32
	Amount, Denom, Period                             string // exact amount / denomination / "block"
	SourceURL, SourceVersion                          string
	ObservedAt, ExpiresAt                             time.Time
	// Conversion fields are facts, not defaults: interval seconds per block,
	// seconds in the admission month, and USD per whole token.
	BlockSeconds                       int64
	BlockSourceURL, BlockSourceVersion string
	BlockObservedAt, BlockExpiresAt    time.Time
	BaseUnitsPerToken                  string
	DenomSourceURL, DenomSourceVersion string
	DenomObservedAt, DenomExpiresAt    time.Time
	USDPerToken                        string
	FXSourceURL, FXSourceVersion       string
	FXObservedAt, FXExpiresAt          time.Time
}

type MonthlyUSD struct {
	Minor          int64
	Fingerprint    string
	ProvenanceJSON string
}

type provenanceEnvelope struct {
	Version      string    `json:"version"`
	RawFact      PriceFact `json:"raw_fact_and_conversion_inputs"`
	MonthSeconds int64     `json:"month_seconds"`
	Currency     string    `json:"normalized_currency"`
	MonthlyMinor int64     `json:"monthly_minor"`
	Fingerprint  string    `json:"fingerprint"`
}

func (f PriceFact) NormalizeMonthlyUSD(now time.Time, monthSeconds int64) (MonthlyUSD, error) {
	missing := []string{}
	if f.Network != "sandbox-2" || strings.TrimSpace(f.ChainID) == "" {
		missing = append(missing, "exact Akash sandbox network and chain ID")
	}
	if f.DSeq == "" || f.GSeq == 0 || f.OSeq == 0 || f.Provider == "" || f.Region == "" || f.OfferID == "" {
		missing = append(missing, "selected offer tuple and region")
	}
	if f.Period != "block" {
		missing = append(missing, "rate period exactly 'block'")
	}
	amount, ok := new(big.Rat).SetString(strings.TrimSpace(f.Amount))
	if !ok || amount.Sign() <= 0 || strings.TrimSpace(f.Denom) == "" {
		missing = append(missing, "positive exact bid amount and denomination")
	}
	if f.SourceURL == "" || f.SourceVersion == "" || !fresh(f.ObservedAt, f.ExpiresAt, now) {
		missing = append(missing, "current source URL/version and bid observation/expiry")
	}
	if f.BlockSeconds <= 0 || f.BlockSourceURL == "" || f.BlockSourceVersion == "" || !fresh(f.BlockObservedAt, f.BlockExpiresAt, now) {
		missing = append(missing, "current authoritative block interval and provenance")
	}
	scale, scaleOK := new(big.Rat).SetString(strings.TrimSpace(f.BaseUnitsPerToken))
	if !scaleOK || scale.Sign() <= 0 || f.DenomSourceURL == "" || f.DenomSourceVersion == "" || !fresh(f.DenomObservedAt, f.DenomExpiresAt, now) {
		missing = append(missing, "current authoritative denomination scale and provenance")
	}
	if monthSeconds <= 0 {
		missing = append(missing, "explicit positive admission-month duration")
	}
	fx, fxOK := new(big.Rat).SetString(strings.TrimSpace(f.USDPerToken))
	if !fxOK || fx.Sign() <= 0 || f.FXSourceURL == "" || f.FXSourceVersion == "" || !fresh(f.FXObservedAt, f.FXExpiresAt, now) {
		missing = append(missing, "current authoritative USD exchange rate and provenance")
	}
	if len(missing) > 0 {
		return MonthlyUSD{}, fmt.Errorf("Akash price is not admissible: missing or invalid %s", strings.Join(missing, "; "))
	}
	blocks := new(big.Rat).SetFrac(big.NewInt(monthSeconds), big.NewInt(f.BlockSeconds))
	value := new(big.Rat).Mul(amount, blocks)
	value.Quo(value, scale)
	value.Mul(value, fx)
	value.Mul(value, big.NewRat(100, 1))
	// Round upward to cents so the admitted ceiling cannot understate a charge.
	n := new(big.Int).Quo(value.Num(), value.Denom())
	if new(big.Int).Mod(value.Num(), value.Denom()).Sign() != 0 {
		n.Add(n, big.NewInt(1))
	}
	if !n.IsInt64() {
		return MonthlyUSD{}, errors.New("Akash monthly USD quote exceeds supported minor-unit range")
	}
	canonical, err := json.Marshal(struct {
		Version      string    `json:"version"`
		Fact         PriceFact `json:"raw_fact_and_conversion_inputs"`
		MonthSeconds int64     `json:"month_seconds"`
		Currency     string    `json:"normalized_currency"`
		MonthlyMinor int64     `json:"monthly_minor"`
	}{"akash-price-provenance/v1", f, monthSeconds, "USD", n.Int64()})
	if err != nil {
		return MonthlyUSD{}, fmt.Errorf("encode Akash price provenance: %w", err)
	}
	h := sha256.Sum256(canonical)
	fingerprint := hex.EncodeToString(h[:])
	payload, err := json.Marshal(struct {
		Version      string    `json:"version"`
		RawFact      PriceFact `json:"raw_fact_and_conversion_inputs"`
		MonthSeconds int64     `json:"month_seconds"`
		Currency     string    `json:"normalized_currency"`
		MonthlyMinor int64     `json:"monthly_minor"`
		Fingerprint  string    `json:"fingerprint"`
	}{"akash-price-provenance/v1", f, monthSeconds, "USD", n.Int64(), fingerprint})
	if err != nil {
		return MonthlyUSD{}, fmt.Errorf("encode Akash price provenance: %w", err)
	}
	return MonthlyUSD{Minor: n.Int64(), Fingerprint: fingerprint, ProvenanceJSON: string(payload)}, nil
}

// NormalizeCatalogueFact adapts a fully sourced Akash observation into the
// shared monthly catalogue contract while retaining an auditable envelope.
func (f PriceFact) NormalizeCatalogueFact(now time.Time, monthSeconds int64) (cmprovider.PriceFact, error) {
	monthly, err := f.NormalizeMonthlyUSD(now, monthSeconds)
	if err != nil {
		return cmprovider.PriceFact{}, err
	}
	return cmprovider.PriceFact{Provider: "akash", OfferID: f.OfferID, Region: f.Region, Size: "lease", Currency: "USD", ServerMinor: monthly.Minor, TaxBasis: "pre-tax", SourceURL: f.SourceURL, SourceVersion: f.SourceVersion, ObservedAt: f.ObservedAt, ExpiresAt: f.ExpiresAt, ProvenanceJSON: monthly.ProvenanceJSON, ProvenanceFingerprint: monthly.Fingerprint}, nil
}

// ValidateCatalogueFact replays the exact normalization and ensures the
// catalogue's scalar fields still match the versioned conversion evidence.
func ValidateCatalogueFact(f cmprovider.PriceFact, now time.Time) error {
	var envelope provenanceEnvelope
	if err := json.Unmarshal([]byte(f.ProvenanceJSON), &envelope); err != nil {
		return fmt.Errorf("Akash price provenance is invalid JSON: %w", err)
	}
	if envelope.Version != "akash-price-provenance/v1" || envelope.Fingerprint == "" {
		return errors.New("Akash price provenance version or fingerprint is missing")
	}
	monthly, err := envelope.RawFact.NormalizeMonthlyUSD(now, envelope.MonthSeconds)
	if err != nil {
		return err
	}
	if monthly.Fingerprint != envelope.Fingerprint || f.ProvenanceFingerprint != envelope.Fingerprint || monthly.Minor != envelope.MonthlyMinor || envelope.Currency != "USD" || f.Provider != Name || f.Currency != envelope.Currency || f.ServerMinor != monthly.Minor || f.IPv4Minor != 0 || f.OfferID != envelope.RawFact.OfferID || f.Region != envelope.RawFact.Region || f.Size != "lease" || f.SourceURL != envelope.RawFact.SourceURL || f.SourceVersion != envelope.RawFact.SourceVersion || !f.ObservedAt.Equal(envelope.RawFact.ObservedAt) || !f.ExpiresAt.Equal(envelope.RawFact.ExpiresAt) {
		return errors.New("Akash catalogue scalar facts do not match the normalized provenance envelope")
	}
	if f.ProvenanceJSON != monthly.ProvenanceJSON {
		return errors.New("Akash provenance payload does not reproduce byte-for-byte")
	}
	return nil
}

// ValidateIntentBinding prevents a validly sourced quote for a different bid
// from being admitted against the persisted selected-bid intent.
func ValidateIntentBinding(provenanceJSON string, intent map[string]any) error {
	var envelope provenanceEnvelope
	if err := json.Unmarshal([]byte(provenanceJSON), &envelope); err != nil {
		return fmt.Errorf("Akash price provenance is invalid JSON: %w", err)
	}
	selected, ok := intent["akash_lease"].(map[string]any)
	if !ok {
		return errors.New("persisted intent is missing its Akash selected bid")
	}
	value := func(k string) string {
		switch v := selected[k].(type) {
		case string:
			return v
		case float64:
			return fmt.Sprintf("%.0f", v)
		default:
			return ""
		}
	}
	raw := envelope.RawFact
	if raw.DSeq != value("dseq") || fmt.Sprint(raw.GSeq) != value("gseq") || fmt.Sprint(raw.OSeq) != value("oseq") || raw.Provider != value("provider") || raw.Amount != value("rate_amount") || raw.Denom != value("rate_denom") {
		return errors.New("Akash quote provenance does not match the immutable selected bid in the intent")
	}
	return nil
}

func fresh(observed, expires, now time.Time) bool {
	return !observed.IsZero() && !expires.IsZero() && !observed.After(now) && observed.Before(expires) && now.Before(expires)
}
