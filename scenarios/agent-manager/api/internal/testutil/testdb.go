package testutil

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/database"
	"agent-manager/internal/modules"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite" // pure-Go SQLite driver registration for test DB
)

var testDBSequence atomic.Uint64

// SetupTestDB creates an isolated shared-cache in-memory SQLite database with
// the full schema applied. MaxOpenConns(1) keeps SQLite's lock semantics stable
// while cache=shared ensures every repository and event store sees one database.
func SetupTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	name := fmt.Sprintf("agent-manager-test-%d", testDBSequence.Add(1))
	dsn := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)",
		name,
	)

	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)

	if err := coredb.EnsureSchemas(context.Background(), db, modules.AllSchemas()...); err != nil {
		db.Close()
		t.Fatalf("apply schema: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	dbWrapper := database.NewDB(db, logger)

	cleanup := func() {
		db.Close()
	}
	return dbWrapper, cleanup
}

// SetupTestRepos creates a temporary SQLite database and returns all repositories
// plus the event store and a cleanup function.
func SetupTestRepos(t *testing.T) (*database.Repositories, event.Store, func()) {
	t.Helper()
	db, cleanup := SetupTestDB(t)

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	repos := database.NewRepositories(db, logger)
	eventStore := event.NewSQLiteStore(db, logger)

	return repos, eventStore, cleanup
}

// SetupTestReposWithDB is like SetupTestRepos but uses an externally provided DB.
// This is useful when the test needs direct access to the DB for raw SQL operations.
func SetupTestReposWithDB(t *testing.T, db *database.DB) (*database.Repositories, event.Store, func()) {
	t.Helper()

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	repos := database.NewRepositories(db, logger)
	eventStore := event.NewSQLiteStore(db, logger)

	return repos, eventStore, func() {}
}
