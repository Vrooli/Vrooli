package credentialgrant

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scheduletest"
	"github.com/vrooli/vrooli/packages/proto/sealing"

	grantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	_ "modernc.org/sqlite"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/channelsign"
	internalgrant "vrooli-bridge/internal/credentialgrant"
	"vrooli-bridge/internal/presence"
)

type handlerTestSigner struct{ key ed25519.PrivateKey }

func (s handlerTestSigner) Sign(msg []byte) []byte { return ed25519.Sign(s.key, msg) }

// [REQ:BRG-P1-002] The production delivery path sends consent metadata before
// the current value, but the value remains sealed for the node and absent from
// every serialized control-plane payload.
func TestDeliverGrantSendsMetadataAndSealedValueWithoutPlaintext(t *testing.T) {
	_, signingKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	nodeKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)

	hub := presence.NewHub(scheduletest.New(time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)))
	conn := hub.Connect("node-1")
	defer conn.Close()

	const secret = "handler-fixture-secret"
	h := NewHandler(ModuleDeps{
		Presence: hub,
		Signer:   handlerTestSigner{key: signingKey},
		SealingPublicKey: func(context.Context, string) ([]byte, error) {
			return nodeKey.PublicKey().Bytes(), nil
		},
		ResolveValue: func(context.Context, string, string) (string, error) {
			return secret, nil
		},
	})
	grant := internalgrant.Grant{
		ID: "grant-1", NodeID: "node-1", LogicalID: "vrooli/test", Field: "api-key",
		Class: internalgrant.ClassUserPrompt, Retention: internalgrant.RetentionEphemeral, Generation: 3,
	}

	require.NoError(t, h.deliverGrant(context.Background(), grant))

	metadataPayload := <-conn.Out()
	metadataFrame, err := channelsign.Open(signingKey.Public().(ed25519.PublicKey), metadataPayload)
	require.NoError(t, err)
	require.NotNil(t, metadataFrame.GetCredentialGrant())
	require.NotContains(t, string(metadataPayload), secret)

	valuePayload := <-conn.Out()
	valueFrame, err := channelsign.Open(signingKey.Public().(ed25519.PublicKey), valuePayload)
	require.NoError(t, err)
	require.NotContains(t, string(valuePayload), secret)
	push := valueFrame.GetCredentialPush()
	require.NotNil(t, push)
	plain, err := sealing.Open(nodeKey, push.GetSealedValue(), push.GetAad())
	require.NoError(t, err)
	require.Equal(t, secret, string(plain))
	require.Equal(t, sealing.CredentialContext("node-1", "vrooli/test", "api-key", 3), push.GetAad())
}

// A node's store passphrase must reach it before any durable value that is
// written into that store.
func TestDeliveryOrderSendsTheNodeStorePassphraseFirst(t *testing.T) {
	grants := []internalgrant.Grant{
		{ID: "a", LogicalID: "vrooli/openrouter"},
		{ID: "b", LogicalID: "vrooli/postgres"},
		{ID: "c", LogicalID: nodeStorePassphrasePrefix + "machine-1"},
	}
	ordered := deliveryOrder(grants)
	require.Equal(t, []string{"c", "a", "b"}, []string{ordered[0].ID, ordered[1].ID, ordered[2].ID})
	require.Equal(t, "a", grants[0].ID, "the caller's slice is left in place")
}

type metadataNodeResolver struct {
	generation int64
	revoked    bool
}

func (metadataNodeResolver) NodeKind(context.Context, string) (string, error) { return "agent", nil }
func (r metadataNodeResolver) NodeGeneration(context.Context, string) (int64, bool, error) {
	return r.generation, r.revoked, nil
}

// ListGrantMetadata must remain a pure metadata read even while the node is
// connected and delivery dependencies are fully configured.
func TestListGrantMetadataHasNoDeliverySideEffectOrSecretFields(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(internalgrant.Schema())
	require.NoError(t, err)
	now := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	repo := internalgrant.NewSQLiteRepository(db, func() time.Time { return now })
	grant, err := repo.Create(context.Background(), internalgrant.Grant{NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: internalgrant.ClassUserPrompt, Retention: internalgrant.RetentionEphemeral, Generation: 2, GrantedAt: now})
	require.NoError(t, err)
	require.NoError(t, repo.RecordReceipt(context.Background(), grant.ID, 2, true, "raw-error-secret-marker", now.Add(time.Second)))
	service := internalgrant.NewService(repo, metadataNodeResolver{generation: 8}, func() time.Time { return now })
	_, signingKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	nodeKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	hub := presence.NewHub(scheduletest.New(now))
	conn := hub.Connect("node-1")
	defer conn.Close()
	resolveCalls := 0
	h := NewHandler(ModuleDeps{Service: service, Presence: hub, Signer: handlerTestSigner{key: signingKey}, SealingPublicKey: func(context.Context, string) ([]byte, error) { return nodeKey.PublicKey().Bytes(), nil }, ResolveValue: func(context.Context, string, string) (string, error) {
		resolveCalls++
		return "credential-secret-marker", nil
	}})
	ctx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1", Kind: identity.ActorHuman})
	resp, err := h.ListGrantMetadata(ctx, connect.NewRequest(&grantv1.ListGrantMetadataRequest{NodeId: "node-1", EnrollmentGeneration: 8}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetGrants(), 1)
	metadata := resp.Msg.GetGrants()[0]
	require.Equal(t, grant.ID, metadata.GetId())
	require.Equal(t, "service/api", metadata.GetLogicalId())
	require.True(t, metadata.GetReceiptAccepted())
	require.NotEmpty(t, metadata.GetVersion())
	wire, _ := json.Marshal(resp.Msg)
	require.NotContains(t, string(wire), "raw-error-secret-marker")
	require.NotContains(t, string(wire), "credential-secret-marker")
	require.NotContains(t, string(wire), "sealed_value")
	require.NotContains(t, string(wire), "receipt_reason")
	require.Zero(t, resolveCalls)
	select {
	case payload := <-conn.Out():
		t.Fatalf("metadata read pushed a frame: %x", payload)
	default:
	}
	_, err = h.ListGrantMetadata(context.Background(), connect.NewRequest(&grantv1.ListGrantMetadataRequest{NodeId: "node-1", EnrollmentGeneration: 8}))
	require.Error(t, err)
	_, err = h.ListGrantMetadata(ctx, connect.NewRequest(&grantv1.ListGrantMetadataRequest{NodeId: "node-1"}))
	require.Error(t, err)
	staleSvc := internalgrant.NewService(repo, metadataNodeResolver{generation: 9}, func() time.Time { return now })
	h.service = staleSvc
	_, err = h.ListGrantMetadata(ctx, connect.NewRequest(&grantv1.ListGrantMetadataRequest{NodeId: "node-1", EnrollmentGeneration: 8}))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "target enrollment generation changed"))
}

func TestOwnerGrantMutationReceiptsReplayWithoutRedispatchAfterRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "grants.db")
	openDB := func() *sql.DB {
		db, err := sql.Open("sqlite", dbPath)
		require.NoError(t, err)
		db.SetMaxOpenConns(1)
		return db
	}
	db := openDB()
	_, err := db.Exec(internalgrant.Schema())
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE nodes (id TEXT PRIMARY KEY, kind TEXT NOT NULL, enrollment_generation INTEGER NOT NULL, revoked_at TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO nodes (id,kind,enrollment_generation,revoked_at) VALUES ('node-1','agent',7,'')`)
	require.NoError(t, err)
	now := time.Date(2026, 11, 20, 11, 0, 0, 0, time.UTC)
	_, signingKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	nodeKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	hub := presence.NewHub(scheduletest.New(now))
	conn := hub.Connect("node-1")
	defer conn.Close()
	resolveCalls := 0
	newHandler := func(db *sql.DB) *handler {
		service := internalgrant.NewService(internalgrant.NewSQLiteRepository(db, func() time.Time { return now }), metadataNodeResolver{generation: 7}, func() time.Time { return now })
		return NewHandler(ModuleDeps{Service: service, Presence: hub, Signer: handlerTestSigner{key: signingKey}, SealingPublicKey: func(context.Context, string) ([]byte, error) { return nodeKey.PublicKey().Bytes(), nil }, ResolveValue: func(context.Context, string, string) (string, error) {
			resolveCalls++
			return "fixture-only-secret", nil
		}})
	}
	h := newHandler(db)
	ownerCtx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-a", Kind: identity.ActorHuman})
	createReq := &grantv1.CreateGrantWithReceiptRequest{NodeId: "node-1", EnrollmentGeneration: 7, LogicalId: "service/api", Field: "token", Class: string(internalgrant.ClassUserPrompt), Retention: string(internalgrant.RetentionEphemeral), Generation: 1, ExpectedRevision: 0, RequestId: "create-request-1"}
	created, err := h.CreateGrantWithReceipt(ownerCtx, connect.NewRequest(createReq))
	require.NoError(t, err)
	require.Equal(t, "owner-a", created.Msg.GetActorId())
	require.Equal(t, int64(1), created.Msg.GetResultRevision())
	require.NotEmpty(t, created.Msg.GetPayloadDigest())
	createdWire, _ := json.Marshal(created.Msg)
	require.NotContains(t, string(createdWire), "fixture-only-secret")
	require.NotContains(t, string(createdWire), "sealed_value")
	require.NotEmpty(t, <-conn.Out())
	require.NotEmpty(t, <-conn.Out())
	require.Equal(t, 1, resolveCalls)
	// Model the independently authenticated target acknowledgement before
	// restart. An exact replay after confirmation must not reapply the effect.
	repo := internalgrant.NewSQLiteRepository(db, func() time.Time { return now })
	require.NoError(t, repo.RecordReceipt(context.Background(), created.Msg.GetGrant().GetId(), 1, true, "", now.Add(time.Second)))
	require.NoError(t, db.Close())

	db = openDB()
	h = newHandler(db)
	repo = internalgrant.NewSQLiteRepository(db, func() time.Time { return now })
	replay, err := h.CreateGrantWithReceipt(ownerCtx, connect.NewRequest(createReq))
	require.NoError(t, err)
	require.Equal(t, created.Msg.GetId(), replay.Msg.GetId())
	require.Equal(t, created.Msg.GetCreatedAt().AsTime(), replay.Msg.GetCreatedAt().AsTime())
	require.Equal(t, 1, resolveCalls)
	select {
	case frame := <-conn.Out():
		t.Fatalf("create replay dispatched another frame: %x", frame)
	default:
	}
	changed := *createReq
	changed.LogicalId = "other/service"
	if _, err = h.CreateGrantWithReceipt(ownerCtx, connect.NewRequest(&changed)); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed request binding=%v", err)
	}
	select {
	case frame := <-conn.Out():
		t.Fatalf("changed request dispatched frame: %x", frame)
	default:
	}

	revokeReq := &grantv1.RevokeGrantWithReceiptRequest{Id: created.Msg.GetGrant().GetId(), NodeId: "node-1", EnrollmentGeneration: 7, ExpectedRevision: 2, RequestId: "revoke-request-1"}
	revoked, err := h.RevokeGrantWithReceipt(ownerCtx, connect.NewRequest(revokeReq))
	require.NoError(t, err)
	require.Equal(t, int64(3), revoked.Msg.GetResultRevision())
	require.NotEmpty(t, <-conn.Out())
	require.NoError(t, repo.RecordPurgeReceipt(context.Background(), created.Msg.GetGrant().GetId(), "node-1", 1, true, "", now.Add(2*time.Second)))
	require.NoError(t, db.Close())
	db = openDB()
	h = newHandler(db)
	repo = internalgrant.NewSQLiteRepository(db, func() time.Time { return now })
	revokeReplay, err := h.RevokeGrantWithReceipt(ownerCtx, connect.NewRequest(revokeReq))
	require.NoError(t, err)
	require.Equal(t, revoked.Msg.GetId(), revokeReplay.Msg.GetId())
	select {
	case frame := <-conn.Out():
		t.Fatalf("revoke replay dispatched another frame: %x", frame)
	default:
	}
	require.NoError(t, db.Close())
}

func TestOwnerGrantMutationReplayResumesUnconfirmedDelivery(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	_, err = db.Exec(internalgrant.Schema())
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE nodes (id TEXT PRIMARY KEY, kind TEXT NOT NULL, enrollment_generation INTEGER NOT NULL, revoked_at TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO nodes (id,kind,enrollment_generation,revoked_at) VALUES ('node-1','agent',7,'')`)
	require.NoError(t, err)
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	_, signingKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	nodeKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	hub := presence.NewHub(scheduletest.New(now))
	service := internalgrant.NewService(internalgrant.NewSQLiteRepository(db, func() time.Time { return now }), metadataNodeResolver{generation: 7}, func() time.Time { return now })
	h := NewHandler(ModuleDeps{Service: service, Presence: hub, Signer: handlerTestSigner{key: signingKey}, SealingPublicKey: func(context.Context, string) ([]byte, error) { return nodeKey.PublicKey().Bytes(), nil }, ResolveValue: func(context.Context, string, string) (string, error) { return "synthetic-secret", nil }})
	ownerCtx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-a", Kind: identity.ActorHuman})
	createReq := &grantv1.CreateGrantWithReceiptRequest{NodeId: "node-1", EnrollmentGeneration: 7, LogicalId: "service/api", Field: "token", Class: string(internalgrant.ClassUserPrompt), Retention: string(internalgrant.RetentionEphemeral), Generation: 1, ExpectedRevision: 0, RequestId: "offline-create"}
	created, err := h.CreateGrantWithReceipt(ownerCtx, connect.NewRequest(createReq))
	require.NoError(t, err) // accepted durably, but no target was connected
	require.Equal(t, "retryable", created.Msg.GetDeliveryState())
	conn := hub.Connect("node-1")
	defer conn.Close()
	resent, err := h.CreateGrantWithReceipt(ownerCtx, connect.NewRequest(createReq))
	require.NoError(t, err)
	require.Equal(t, "dispatched", resent.Msg.GetDeliveryState())
	require.NotEmpty(t, <-conn.Out())
	require.NotEmpty(t, <-conn.Out())
	repo := internalgrant.NewSQLiteRepository(db, func() time.Time { return now })
	require.NoError(t, repo.RecordReceipt(context.Background(), created.Msg.GetGrant().GetId(), 1, true, "", now.Add(time.Second)))
	confirmed, err := h.CreateGrantWithReceipt(ownerCtx, connect.NewRequest(createReq))
	require.NoError(t, err)
	require.Equal(t, "confirmed", confirmed.Msg.GetDeliveryState())
	select {
	case frame := <-conn.Out():
		t.Fatalf("confirmed duplicate replay dispatched another frame: %x", frame)
	default:
	}

	revokeReq := &grantv1.RevokeGrantWithReceiptRequest{Id: created.Msg.GetGrant().GetId(), NodeId: "node-1", EnrollmentGeneration: 7, ExpectedRevision: 2, RequestId: "offline-revoke"}
	_, err = h.RevokeGrantWithReceipt(ownerCtx, connect.NewRequest(revokeReq))
	require.NoError(t, err)
	// Remove the first purge delivery's frame to model loss before target apply.
	<-conn.Out()
	retriedPurge, err := h.RevokeGrantWithReceipt(ownerCtx, connect.NewRequest(revokeReq))
	require.NoError(t, err)
	require.Equal(t, "dispatched", retriedPurge.Msg.GetDeliveryState())
	require.NotEmpty(t, <-conn.Out())
	require.NoError(t, repo.RecordPurgeReceipt(context.Background(), created.Msg.GetGrant().GetId(), "node-1", 1, true, "", now.Add(2*time.Second)))
	confirmedPurge, err := h.RevokeGrantWithReceipt(ownerCtx, connect.NewRequest(revokeReq))
	require.NoError(t, err)
	require.Equal(t, "confirmed", confirmedPurge.Msg.GetDeliveryState())
	select {
	case frame := <-conn.Out():
		t.Fatalf("confirmed purge replay dispatched another frame: %x", frame)
	default:
	}

	refusedReq := *createReq
	refusedReq.LogicalId = "service/refused"
	refusedReq.RequestId = "refused-create"
	refused, err := h.CreateGrantWithReceipt(ownerCtx, connect.NewRequest(&refusedReq))
	require.NoError(t, err)
	grantID := refused.Msg.GetGrant().GetId()
	<-conn.Out()
	<-conn.Out()
	require.NoError(t, repo.RecordReceipt(context.Background(), grantID, 1, false, "synthetic refusal reason", now.Add(3*time.Second)))
	refusedReplay, err := h.CreateGrantWithReceipt(ownerCtx, connect.NewRequest(&refusedReq))
	require.NoError(t, err)
	require.Equal(t, "refused", refusedReplay.Msg.GetDeliveryState())
	refusedWire, _ := json.Marshal(refusedReplay.Msg)
	require.NotContains(t, string(refusedWire), "synthetic refusal reason")
	select {
	case frame := <-conn.Out():
		t.Fatalf("terminal refusal replay dispatched another frame: %x", frame)
	default:
	}
}
