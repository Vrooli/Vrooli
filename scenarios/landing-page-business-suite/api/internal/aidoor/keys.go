// Package aidoor owns LPBS's public AI identity and reservation boundary.
package aidoor

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

type Key struct {
	ID         string `json:"id"`
	Token      string `json:"key,omitempty"`
	ParentID   string `json:"-"`
	Kind       string `json:"kind"`
	Account    string `json:"business_account_id"`
	Actor      string `json:"-"`
	App        string `json:"app_bundle_key"`
	Harness    string `json:"harness"`
	Model      string `json:"model"`
	MaxCredits int64  `json:"max_credits"`
	ExpiresAt  int64  `json:"expires_at"`
}

type KeyStore struct{ DB *sql.DB }

func randomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func hashKey(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// IssueDevice is called only after consumer authentication. Membership is
// rechecked in the authoritative account store; no email wallet fallback exists.
func (s KeyStore) IssueDevice(ctx context.Context, account, actor, app string, maxCredits int64) (Key, error) {
	if account == "" || actor == "" || maxCredits <= 0 {
		return Key{}, errors.New("account, actor and positive credit cap required")
	}
	var member int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM business_account_members WHERE business_account_id=$1 AND user_id=$2`, account, actor).Scan(&member); err != nil {
		return Key{}, err
	}
	if member != 1 {
		return Key{}, errors.New("account membership required")
	}
	return s.issue(ctx, Key{Kind: "device", Account: account, Actor: actor, App: app, MaxCredits: maxCredits, ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Unix()})
}
func (s KeyStore) IssueSession(ctx context.Context, parent Key, harness, model string, maxCredits int64, ttl time.Duration) (Key, error) {
	if parent.Kind != "device" || harness == "" || model == "" || maxCredits <= 0 || maxCredits > parent.MaxCredits || ttl <= 0 || ttl > 24*time.Hour {
		return Key{}, errors.New("invalid session scope")
	}
	expires := time.Now().Add(ttl).Unix()
	if expires > parent.ExpiresAt {
		expires = parent.ExpiresAt
	}
	return s.issue(ctx, Key{ParentID: parent.ID, Kind: "session", Account: parent.Account, Actor: parent.Actor, App: parent.App, Harness: harness, Model: model, MaxCredits: maxCredits, ExpiresAt: expires})
}
func (s KeyStore) issue(ctx context.Context, key Key) (Key, error) {
	key.ID = randomID()
	key.Token = "vrooli_" + randomID()
	_, err := s.DB.ExecContext(ctx, `INSERT INTO ai_access_keys(id,token_hash,parent_id,kind,business_account_id,actor_identity,app_bundle_key,harness,model,max_credits,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, key.ID, hashKey(key.Token), key.ParentID, key.Kind, key.Account, key.Actor, key.App, key.Harness, key.Model, key.MaxCredits, key.ExpiresAt)
	return key, err
}
func (s KeyStore) Authenticate(ctx context.Context, bearer string) (Key, error) {
	var key Key
	token := strings.TrimPrefix(bearer, "Bearer ")
	if token == bearer || token == "" {
		return key, errors.New("issued bearer key required")
	}
	err := s.DB.QueryRowContext(ctx, `SELECT k.id,k.parent_id,k.kind,k.business_account_id,k.actor_identity,k.app_bundle_key,k.harness,k.model,k.max_credits,k.expires_at FROM ai_access_keys k JOIN business_account_members m ON m.business_account_id=k.business_account_id AND m.user_id=k.actor_identity WHERE k.token_hash=$1 AND k.revoked=FALSE AND k.expires_at>$2 AND (k.parent_id='' OR EXISTS(SELECT 1 FROM ai_access_keys p WHERE p.id=k.parent_id AND p.revoked=FALSE AND p.expires_at>$2))`, hashKey(token), time.Now().Unix()).Scan(&key.ID, &key.ParentID, &key.Kind, &key.Account, &key.Actor, &key.App, &key.Harness, &key.Model, &key.MaxCredits, &key.ExpiresAt)
	if err != nil {
		return Key{}, errors.New("key expired, revoked or unknown")
	}
	return key, nil
}
func (s KeyStore) Revoke(ctx context.Context, caller Key, id string) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE ai_access_keys SET revoked=TRUE WHERE id=$1 AND (id=$2 OR parent_id=$2)`, id, caller.ID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n != 1 {
		return errors.New("key not in caller scope")
	}
	return err
}
