package main

import (
	"context"
	"encoding/json"
	"fmt"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// This process prepares isolated storage and a production API artifact. It does
// not serve LPBS: cloud-target must stage the artifact and the scenario lifecycle
// must launch it. The HTTP endpoint below only observes the fixture database.
func TestJourneyHostedSignInServer(t *testing.T) {
	if os.Getenv("VROOLI_JOURNEY_HOSTED_SIGNIN") != "1" {
		t.Skip("hosted integration fixture")
	}
	root := os.Getenv("VROOLI_JOURNEY_HOSTED_ROOT")
	if root == "" {
		t.Fatal("isolated target root is required")
	}
	t.Setenv("TEST_DATABASE_URL", "")
	t.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "0")
	initialize := exec.Command(filepath.Join(root, ".vrooli/bin/vrooli"), "credentials", "store", "init", "--format", "json")
	initialize.Stdin = strings.NewReader("J13-isolated-store-passphrase")
	if output, err := initialize.CombinedOutput(); err != nil {
		t.Fatalf("initialize isolated encrypted authority: %v: %s", err, output)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic bootstrap credentials go into the real encrypted authority in
	// the isolated target HOME. Nothing is read from the operator's authority.
	for key, value := range map[string]string{
		"session-secret":         "j13-isolated-session-secret-32-characters",
		"admin-default-password": "J13-isolated-admin-password!",
	} {
		if err := authority.Put(credentialauthority.Identity("vrooli/landing-page-business-suite"), key, value); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := postgres.RunContainer(ctx, tc.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("hosted_signin"), postgres.WithUsername("fixture"), postgres.WithPassword("fixture-password"),
		tc.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Error(err)
		}
	})
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := openAndPrepareTestDatabase(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	binary := filepath.Join(root, "landing-page-business-suite-api")
	build := exec.CommandContext(context.Background(), "go", "build", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build production LPBS API: %v: %s", err, output)
	}
	done := make(chan struct{})
	var once sync.Once
	observer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/__fixture/stop" {
			once.Do(func() { close(done) })
			return
		}
		var tokens, queued int
		if err := db.QueryRowContext(r.Context(), `SELECT count(*) FROM auth_tokens WHERE email='hosted-journey@example.invalid' AND delivery_status='queued'`).Scan(&tokens); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err := db.QueryRowContext(r.Context(), `SELECT count(*) FROM email_outbox WHERE recipient='hosted-journey@example.invalid'`).Scan(&queued); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]int{"queued_tokens": tokens, "outbox_messages": queued})
	}))
	defer observer.Close()
	info, _ := json.Marshal(map[string]string{"binary": binary, "database_url": dsn, "observer_url": observer.URL})
	fmt.Println("HOSTED_FIXTURE=" + string(info))
	// The parent keeps storage alive until its lifecycle stop has completed.
	<-done
}
