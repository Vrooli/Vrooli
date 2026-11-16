package monetization

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	sharedmonetization "github.com/vrooli/vrooli/packages/monetization-go"
)

type journeyIssuerKeyStore struct{ jwks []byte }

func (s journeyIssuerKeyStore) Load(context.Context, string) ([]byte, error) { return s.jwks, nil }
func (journeyIssuerKeyStore) Save(context.Context, string, []byte) error     { return nil }

// This is an executable attack journey through the shared paid-surface kit.
// It intentionally complements source/declaration scans: those scans cannot
// establish that a gate rejects a forged, expired, or unpaid lease.
func TestSharedKitRejectsForgedExpiredAndUnpaidLeasesAndKeepsValidLeaseOffline(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "journey-key", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	newLease := func(status string, expiry time.Time) string {
		t.Helper()
		token, signErr := entitlementclient.Sign(entitlementclient.Payload{
			UserIdentity: "buyer@example.test", Status: status,
			Features: []string{"premium"}, NotAfter: expiry,
		}, "journey-key", key)
		if signErr != nil {
			t.Fatal(signErr)
		}
		return token
	}
	store, err := entitlementclient.NewFileLeaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	put := func(token string) {
		t.Helper()
		if saveErr := store.Save("buyer@example.test", entitlementclient.LeaseRecord{Lease: token}); saveErr != nil {
			t.Fatal(saveErr)
		}
	}
	gateForStoredLease := func(at time.Time) *sharedmonetization.Gate {
		client := entitlementclient.NewClient("", nil, nil)
		client.SetStore(store)
		client.SetIssuerKeyStore(journeyIssuerKeyStore{jwks: jwks})
		client.Now = func() time.Time { return at }
		return sharedmonetization.NewGate(client, nil, "business_suite")
	}

	t.Run("forged signature is refused", func(t *testing.T) {
		token := newLease("active", now.Add(time.Hour))
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			t.Fatalf("signed lease has %d parts", len(parts))
		}
		parts[1] += "x"
		put(strings.Join(parts, "."))
		decision := gateForStoredLease(now).Feature(t.Context(), "buyer@example.test", "premium", 0)
		if decision.Allowed {
			t.Fatalf("forged lease granted access: %+v", decision)
		}
	})

	t.Run("expired signed lease is refused", func(t *testing.T) {
		// Sign() correctly refuses to issue already-expired leases. Persist a
		// valid lease then advance the verifier clock past its signed expiry.
		put(newLease("active", now.Add(time.Minute)))
		decision := gateForStoredLease(now.Add(2*time.Minute)).Feature(t.Context(), "buyer@example.test", "premium", 0)
		if decision.Allowed || decision.Reason != sharedmonetization.ReasonLeaseExpired {
			t.Fatalf("expired lease decision = %+v", decision)
		}
	})

	t.Run("unpaid subscription is refused", func(t *testing.T) {
		put(newLease("canceled", now.Add(time.Hour)))
		decision := gateForStoredLease(now).Feature(t.Context(), "buyer@example.test", "premium", 0)
		if decision.Allowed || decision.Reason != sharedmonetization.ReasonSubscriptionInactive {
			t.Fatalf("unpaid lease decision = %+v", decision)
		}
	})

	t.Run("valid signed lease survives offline process restart", func(t *testing.T) {
		put(newLease("active", now.Add(time.Hour)))
		// A fresh client has no in-memory lease and no authority URL. It must
		// load and verify the durable signed record before allowing local work.
		decision := gateForStoredLease(now).CachedFeatureAt("buyer@example.test", "premium", 0, now)
		if !decision.Allowed {
			t.Fatalf("valid cached lease denied after restart: %+v", decision)
		}
	})
}
