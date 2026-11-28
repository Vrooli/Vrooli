package journeytest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/aidoor"
)

// SM5 is an observation of authoritative SQL facts, never another key store.
type keyFacts struct {
	Kind                                                                      string
	Absent, Revoked, Expired, MembershipRemoved, ParentRevoked, ParentExpired bool
}

func (f keyFacts) valid() bool {
	return !f.Absent && !f.Revoked && !f.Expired && !f.MembershipRemoved && !f.ParentRevoked && !f.ParentExpired
}

type keyEvent struct {
	Name, Actor, Effect string
	Success, Denied     int
	DeviceOnly          bool `json:"device_only"`
}

type keySpec struct {
	Machine          string
	TargetLevel      int `json:"target_level"`
	Kinds, Facts     []string
	DeviceFactCount  int  `json:"device_fact_count"`
	SessionFactCount int  `json:"session_fact_count"`
	IncludeAbsent    bool `json:"include_absent"`
	Events           []keyEvent
	Traces           []struct {
		Name, Kind string
		Events     []string
	}
}

func readKeySpec(t *testing.T) keySpec {
	t.Helper()
	b, err := os.ReadFile("../internal/aidoor/keys.spec.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec keySpec
	if err := json.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Machine != "SM5" || spec.TargetLevel != 3 || !spec.IncludeAbsent || spec.DeviceFactCount != 3 || spec.SessionFactCount != 5 || !reflect.DeepEqual(spec.Kinds, []string{"device", "session"}) || !reflect.DeepEqual(spec.Facts, []string{"revoked", "expired", "membership_removed", "parent_revoked", "parent_expired"}) {
		t.Fatal("SM5 vocabulary drift")
	}
	want := []string{"issue", "authenticate", "mint_child", "mint_excess_cap", "mint_excess_ttl", "revoke_self", "revoke_child", "revoke_parent", "revoke_unrelated", "model_out_of_scope", "expire", "parent_revoke", "parent_expire", "membership_remove"}
	var got []string
	for _, e := range spec.Events {
		got = append(got, e.Name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event vocabulary: %v", got)
	}
	return spec
}

type keyFixture struct{ target, parent, child, unrelated aidoor.Key }

func (s *spine) keySQL(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := s.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func (s *spine) keyFixture(t *testing.T, f keyFacts) keyFixture {
	t.Helper()
	s.keySQL(t, `DELETE FROM ai_access_keys`)
	s.keySQL(t, `INSERT INTO business_account_members(business_account_id,user_id,role) VALUES('account','actor','owner') ON CONFLICT DO NOTHING`)
	store := aidoor.KeyStore{DB: s.db}
	issue := func() aidoor.Key {
		k, err := store.IssueDevice(context.Background(), "account", "actor", "journey", 10000)
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	fxt := keyFixture{target: issue(), unrelated: issue()}
	fxt.parent = fxt.unrelated // Device has no parent; this is an out-of-scope target.
	if f.Kind == "session" {
		fxt.parent = fxt.target
		var err error
		fxt.target, err = store.IssueSession(context.Background(), fxt.parent, "opencode", "fixture-model", 5000, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
	}
	// A session's "child" target is a sibling: sessions cannot own children.
	parent := fxt.target
	if f.Kind == "session" {
		parent = fxt.parent
	}
	var err error
	fxt.child, err = store.IssueSession(context.Background(), parent, "opencode", "fixture-model", 5000, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if f.Revoked {
		s.keySQL(t, `UPDATE ai_access_keys SET revoked=TRUE WHERE id=$1`, fxt.target.ID)
	}
	if f.Expired {
		s.keySQL(t, `UPDATE ai_access_keys SET expires_at=$1 WHERE id=$2`, time.Now().Unix()-1, fxt.target.ID)
	}
	if f.ParentRevoked {
		s.keySQL(t, `UPDATE ai_access_keys SET revoked=TRUE WHERE id=$1`, fxt.parent.ID)
	}
	if f.ParentExpired {
		s.keySQL(t, `UPDATE ai_access_keys SET expires_at=$1 WHERE id=$2`, time.Now().Unix()-1, fxt.parent.ID)
	}
	if f.MembershipRemoved {
		s.keySQL(t, `DELETE FROM business_account_members`)
	}
	if f.Absent {
		s.keySQL(t, `DELETE FROM ai_access_keys WHERE id=$1`, fxt.target.ID)
	}
	return fxt
}

// The oracle describes caller authority independently of the SQL predicates.
// Issuance always targets a NEW key; it cannot clear the subject's old facts.
func keyExpected(f keyFacts, e keyEvent) (keyFacts, int) {
	next := f
	if e.Actor == "target" && !f.valid() {
		return next, 401
	}
	if e.Actor == "parent" {
		if f.Kind == "device" {
			return next, 204
		} // no parent: environmental no-op
		if f.ParentRevoked || f.ParentExpired || f.MembershipRemoved {
			return next, 401
		}
	}
	if e.DeviceOnly && f.Kind != "device" {
		return next, e.Denied
	}
	if e.Actor == "issuer" {
		if f.Kind == "device" && f.MembershipRemoved {
			return next, e.Denied
		}
		if f.Kind == "session" && (f.ParentRevoked || f.ParentExpired || f.MembershipRemoved) {
			return next, 401
		}
	}
	switch e.Effect {
	case "revoke_self":
		next.Revoked = true
	case "expire":
		if !f.Absent {
			next.Expired = true
		}
	case "parent_revoke":
		if f.Kind == "session" {
			next.ParentRevoked = true
		}
	case "parent_expire":
		if f.Kind == "session" {
			next.ParentExpired = true
		}
	case "membership_remove":
		next.MembershipRemoved = true
	case "issue", "mint", "none", "revoke_child":
	default:
		panic("undeclared effect " + e.Effect)
	}
	return next, e.Success
}

func (s *spine) keyHTTP(t *testing.T, token, method, path, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, s.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, b
}

func (s *spine) keySnapshot(t *testing.T) map[string]string {
	t.Helper()
	rows, err := s.db.Query(`SELECT id, row_to_json(k)::text FROM ai_access_keys k ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, row string
		if err := rows.Scan(&id, &row); err != nil {
			t.Fatal(err)
		}
		out[id] = row
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func (s *spine) replayKeyEvent(t *testing.T, fxt keyFixture, f keyFacts, e keyEvent) keyFacts {
	t.Helper()
	next, want := keyExpected(f, e)
	before := s.keySnapshot(t)
	status := 204
	var body []byte
	mint := `{"harness":"opencode","model":"fixture-model","max_credits":1000,"ttl_seconds":300}`
	switch e.Name {
	case "issue":
		if f.Kind == "device" {
			_, err := (aidoor.KeyStore{DB: s.db}).IssueDevice(context.Background(), "account", "actor", "journey", 10000)
			status = 200
			if err != nil {
				status = 403
			}
		} else {
			status, body = s.keyHTTP(t, fxt.parent.Token, "POST", "/api/v1/ai/keys/session", mint)
		}
	case "authenticate":
		_, err := (aidoor.KeyStore{DB: s.db}).Authenticate(context.Background(), "Bearer "+fxt.target.Token)
		if (err == nil) != f.valid() {
			t.Fatalf("direct Authenticate err=%v facts=%+v", err, f)
		}
		status, body = s.keyHTTP(t, fxt.target.Token, "GET", "/sm5-authenticated-probe", "")
	case "mint_child", "mint_excess_cap", "mint_excess_ttl":
		if e.Name == "mint_excess_cap" {
			mint = strings.Replace(mint, "1000", "10001", 1)
		}
		if e.Name == "mint_excess_ttl" {
			mint = strings.Replace(mint, "300", "86401", 1)
		}
		status, body = s.keyHTTP(t, fxt.target.Token, "POST", "/api/v1/ai/keys/session", mint)
	case "revoke_self", "revoke_child", "revoke_parent", "revoke_unrelated":
		id := fxt.target.ID
		switch e.Name {
		case "revoke_child":
			id = fxt.child.ID
		case "revoke_parent":
			id = fxt.parent.ID
		case "revoke_unrelated":
			id = fxt.unrelated.ID
		}
		status, body = s.keyHTTP(t, fxt.target.Token, "DELETE", "/api/v1/ai/keys/"+id, "")
	case "model_out_of_scope":
		// Empty model is invalid for device; a session also forbids a different model.
		model := ""
		if f.Kind == "session" {
			model = "other-model"
		}
		status, body = s.keyHTTP(t, fxt.target.Token, "POST", "/api/v1/ai/chat/completions", fmt.Sprintf(`{"model":%q}`, model))
	case "expire":
		s.keySQL(t, `UPDATE ai_access_keys SET expires_at=$1 WHERE id=$2`, time.Now().Unix()-1, fxt.target.ID)
	case "parent_revoke":
		if f.Kind == "session" {
			status, body = s.keyHTTP(t, fxt.parent.Token, "DELETE", "/api/v1/ai/keys/"+fxt.parent.ID, "")
		}
	case "parent_expire":
		if f.Kind == "session" {
			s.keySQL(t, `UPDATE ai_access_keys SET expires_at=$1 WHERE id=$2`, time.Now().Unix()-1, fxt.parent.ID)
		}
	case "membership_remove":
		s.keySQL(t, `DELETE FROM business_account_members`)
	default:
		t.Fatalf("unreplayed event %s", e.Name)
	}
	if status != want {
		t.Fatalf("%+v --%s--> status=%d want=%d body=%s", f, e.Name, status, want, body)
	}
	after := s.keySnapshot(t)
	changed := ""
	if status < 300 {
		switch e.Effect {
		case "revoke_self", "expire":
			changed = fxt.target.ID
		case "revoke_child":
			changed = fxt.child.ID
		case "parent_revoke", "parent_expire":
			if f.Kind == "session" {
				changed = fxt.parent.ID
			}
		}
	}
	for id, row := range before {
		if id != changed && after[id] != row {
			t.Fatalf("%s changed unrelated/denied row %s", e.Name, id)
		}
	}
	wantCount := len(before)
	if status == 200 && (e.Effect == "issue" || e.Effect == "mint") {
		wantCount++
	}
	if len(after) != wantCount {
		t.Fatalf("%s key rows=%d want=%d", e.Name, len(after), wantCount)
	}
	if !f.Absent {
		var revoked, expired bool
		if err := s.db.QueryRow(`SELECT revoked,expires_at<=$1 FROM ai_access_keys WHERE id=$2`, time.Now().Unix(), fxt.target.ID).Scan(&revoked, &expired); err != nil {
			t.Fatal(err)
		}
		if revoked != next.Revoked || expired != next.Expired {
			t.Fatalf("stored target differs: revoked=%v expired=%v want=%+v", revoked, expired, next)
		}
	}
	if f.Kind == "session" {
		var revoked, expired bool
		if err := s.db.QueryRow(`SELECT revoked,expires_at<=$1 FROM ai_access_keys WHERE id=$2`, time.Now().Unix(), fxt.parent.ID).Scan(&revoked, &expired); err != nil {
			t.Fatal(err)
		}
		if revoked != next.ParentRevoked || expired != next.ParentExpired {
			t.Fatalf("stored parent differs: revoked=%v expired=%v want=%+v", revoked, expired, next)
		}
	}
	var members int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_account_members WHERE business_account_id='account' AND user_id='actor'`).Scan(&members); err != nil {
		t.Fatal(err)
	}
	if (members == 0) != next.MembershipRemoved {
		t.Fatalf("membership differs: count=%d want=%+v", members, next)
	}
	if e.Effect == "revoke_child" && status == 204 {
		var revoked bool
		if err := s.db.QueryRow(`SELECT revoked FROM ai_access_keys WHERE id=$1`, fxt.child.ID).Scan(&revoked); err != nil || !revoked {
			t.Fatalf("child revocation: %v %v", revoked, err)
		}
	}
	_, err := (aidoor.KeyStore{DB: s.db}).Authenticate(context.Background(), "Bearer "+fxt.target.Token)
	if (err == nil) != next.valid() {
		t.Fatalf("post-event validity=%v want=%v facts=%+v", err == nil, next.valid(), next)
	}
	t.Logf("SM5 %s %+v --%s/%d--> %+v", f.Kind, f, e.Name, status, next)
	return next
}

func TestJ17_KeyLifecycleMatrixAndTraces(t *testing.T) {
	spec := readKeySpec(t)
	s := newSpine(t)
	pairs := 0
	for _, kind := range spec.Kinds {
		bits := spec.DeviceFactCount
		if kind == "session" {
			bits = spec.SessionFactCount
		}
		for _, absent := range []bool{false, true} {
			for mask := 0; mask < 1<<bits; mask++ {
				// A missing row has no own revoked/expiry facts, but still has issuer prerequisites.
				if absent && mask&3 != 0 {
					continue
				}
				f := keyFacts{Kind: kind, Absent: absent}
				f.Revoked = mask&1 != 0
				f.Expired = mask&2 != 0
				f.MembershipRemoved = mask&4 != 0
				f.ParentRevoked = mask&8 != 0
				f.ParentExpired = mask&16 != 0
				for _, e := range spec.Events {
					t.Run(fmt.Sprintf("matrix/%s/absent=%v/%02d/%s", kind, absent, mask, e.Name), func(t *testing.T) { s.replayKeyEvent(t, s.keyFixture(t, f), f, e) })
					pairs++
				}
			}
		}
	}
	if pairs != 700 {
		t.Fatalf("matrix pairs=%d want=700", pairs)
	}
	for _, trace := range spec.Traces {
		t.Run("trace/"+trace.Name, func(t *testing.T) {
			f := keyFacts{Kind: trace.Kind}
			fxt := s.keyFixture(t, f)
			for _, name := range trace.Events {
				found := false
				for _, e := range spec.Events {
					if e.Name == name {
						f = s.replayKeyEvent(t, fxt, f, e)
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("undeclared trace event %s", name)
				}
			}
		})
	}
	assertCharged(t, s, 0)
	var claims int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM ai_key_charges`).Scan(&claims); err != nil || claims != 0 {
		t.Fatalf("lifecycle consumed cap: %d %v", claims, err)
	}
	t.Logf("SM5 replayed %d pairs and %d traces through real Postgres/KeyStore/Door", pairs, len(spec.Traces))
}

func TestJ17_KeyLifecyclePaidTraces(t *testing.T) {
	spec := readKeySpec(t)
	s := newSpine(t)
	var total int64
	for _, kind := range spec.Kinds {
		for _, name := range []string{"revoke_self", "expire", "parent_revoke", "parent_expire", "membership_remove"} {
			if kind == "device" && strings.HasPrefix(name, "parent_") {
				continue
			}
			t.Run(kind+"/"+name, func(t *testing.T) {
				f := keyFacts{Kind: kind}
				fxt := s.keyFixture(t, f)
				resp := s.request(t, fxt.target.Token, "sm5-paid-"+name, "hello")
				_, err := io.Copy(io.Discard, resp.Body)
				if err != nil || resp.StatusCode != 200 {
					t.Fatalf("paid request status=%d err=%v", resp.StatusCode, err)
				}
				total += 200
				assertCharged(t, s, total)
				scopes := []string{fxt.target.ID}
				if kind == "session" {
					scopes = append(scopes, fxt.parent.ID)
				}
				for _, id := range scopes {
					var n int
					var credits int64
					if err := s.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(actual_credits),0) FROM ai_key_charges WHERE key_id=$1`, id).Scan(&n, &credits); err != nil || n != 1 || credits != 200 {
						t.Fatalf("scope settlement count=%d credits=%d err=%v", n, credits, err)
					}
				}
				for _, e := range spec.Events {
					if e.Name == name {
						s.replayKeyEvent(t, fxt, f, e)
						break
					}
				}
				var claimsBefore, claimsAfter int
				if err := s.db.QueryRow(`SELECT COUNT(*) FROM ai_key_charges`).Scan(&claimsBefore); err != nil {
					t.Fatal(err)
				}
				denied := s.request(t, fxt.target.Token, "sm5-after-"+name, "hello")
				io.Copy(io.Discard, denied.Body)
				if denied.StatusCode != 401 {
					t.Fatalf("invalidated key spent: status=%d", denied.StatusCode)
				}
				assertCharged(t, s, total)
				if err := s.db.QueryRow(`SELECT COUNT(*) FROM ai_key_charges`).Scan(&claimsAfter); err != nil || claimsAfter != claimsBefore {
					t.Fatalf("denial mutated cap claims: before=%d after=%d err=%v", claimsBefore, claimsAfter, err)
				}
			})
		}
	}
}

func TestJ17_KeyIssuanceBounds(t *testing.T) {
	s := newSpine(t)
	parentExpiry := time.Now().Add(30 * time.Minute).Unix()
	s.keySQL(t, `UPDATE ai_access_keys SET expires_at=$1 WHERE id=$2`, parentExpiry, s.key.ID)
	status, body := s.keyHTTP(t, s.key.Token, "POST", "/api/v1/ai/keys/session", `{"harness":"opencode","model":"fixture-model","max_credits":5000,"ttl_seconds":3600}`)
	if status != 200 {
		t.Fatalf("mint status=%d", status)
	}
	var child aidoor.Key
	if err := json.Unmarshal(body, &child); err != nil {
		t.Fatal(err)
	}
	if child.ExpiresAt != parentExpiry || child.Kind != "session" || child.ID == s.key.ID || child.Token == s.key.Token {
		t.Fatal("session did not inherit parent expiry and fresh identity")
	}
	auth, err := (aidoor.KeyStore{DB: s.db}).Authenticate(context.Background(), "Bearer "+child.Token)
	if err != nil || auth.ParentID != s.key.ID || auth.Account != s.key.Account || auth.Actor != s.key.Actor || auth.App != s.key.App || auth.Harness != "opencode" || auth.Model != "fixture-model" || auth.MaxCredits != 5000 {
		t.Fatalf("child scope mismatch err=%v", err)
	}
	for _, body := range []string{
		`{"model":"fixture-model","max_credits":1000,"ttl_seconds":300}`,
		`{"harness":"opencode","max_credits":1000,"ttl_seconds":300}`,
		`{"harness":"opencode","model":"fixture-model","max_credits":0,"ttl_seconds":300}`,
		`{"harness":"opencode","model":"fixture-model","max_credits":1000,"ttl_seconds":0}`,
		`{"harness":"opencode","model":"fixture-model","max_credits":10001,"ttl_seconds":300}`,
		`{"harness":"opencode","model":"fixture-model","max_credits":1000,"ttl_seconds":86401}`,
	} {
		before := s.keySnapshot(t)
		status, _ := s.keyHTTP(t, s.key.Token, "POST", "/api/v1/ai/keys/session", body)
		if status != 400 || !reflect.DeepEqual(before, s.keySnapshot(t)) {
			t.Fatalf("invalid scope created a key: status=%d", status)
		}
	}
	store := aidoor.KeyStore{DB: s.db}
	for _, input := range []struct {
		account, actor string
		cap            int64
	}{{"", "actor", 100}, {"account", "", 100}, {"account", "actor", 0}, {"account", "non-member", 100}, {"other-account", "actor", 100}} {
		before := s.keySnapshot(t)
		if _, err := store.IssueDevice(context.Background(), input.account, input.actor, "journey", input.cap); err == nil || !reflect.DeepEqual(before, s.keySnapshot(t)) {
			t.Fatalf("invalid device issuance accepted: %+v", input)
		}
	}
}
