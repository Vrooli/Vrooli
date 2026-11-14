package hub_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"

	"notification-hub/internal/hub"
)

func TestDeriveSenderNamesWhoIsAsking(t *testing.T) {
	for name, tc := range map[string]struct {
		declared, source, eventType, createdBy string
		want                                   string
	}{
		"a declared sender wins, with its spacing tidied": {"  Treasury   steward ", "operator", "", "run:1", "Treasury steward"},
		"a declared sender beats a verified service":      {"Night shift", "research-desk", "x.v1", "", "Night shift"},
		"an event source is named from its scenario":      {"", "research-desk", "research_desk.decision.v1", "", "Research Desk"},
		"a verified service is named from its scenario":   {"", "data-backup-manager", "", "service:data-backup-manager", "Data Backup Manager"},
		"a run that declared nothing is an agent run":     {"", "operator", "", "run:e35e92ee", hub.SenderAgentRun},
		"an owner's direct ask has no name":               {"", "operator", "", "owner:me", hub.SenderUnnamed},
		"a caller with no identity has no name":           {"", "operator", "", "", hub.SenderUnnamed},
		"a blank declaration is no declaration":           {"   ", "operator", "", "run:1", hub.SenderAgentRun},
	} {
		require.Equal(t, tc.want, hub.DeriveSender(tc.declared, tc.source, tc.eventType, tc.createdBy, "operator", nil), name)
	}
}

func TestAskStoresADeclaredSenderOrNamesOne(t *testing.T) {
	l := newListAsks(t)
	declared := l.ask(raise{age: time.Minute, sender: "Treasury steward", by: "run:1"})
	require.Equal(t, "Treasury steward", declared.Sender)
	event := l.ask(raise{age: time.Minute, source: "research-desk"})
	require.Equal(t, "Research Desk", event.Sender, "an event-sourced ask is named from its source")

	spec := decisionSpec(listNow, "sender-too-long")
	spec.Sender = strings.Repeat("x", 61)
	_, _, err := l.service.AskWithSpec(context.Background(), spec)
	require.ErrorIs(t, err, hub.ErrInvalidArgument, "a sender is at most 60 characters")
	spec.Sender = "bad\x00name"
	_, _, err = l.service.AskWithSpec(context.Background(), spec)
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
}

func TestSubjectAskKeepsItsSenderUnlessTheRequestNamesAnother(t *testing.T) {
	service, _ := testService(t)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := subjectSpec(start, "evt-1")
	first.Sender = "Backup steward"
	opened := askSubject(t, service, first)
	require.Equal(t, "Backup steward", opened.Ask.Sender)

	// A repeat that names no sender keeps the one the operator knows.
	later := start.Add(time.Hour)
	at(service, later)
	repeat := askSubject(t, service, subjectSpec(later, "evt-2"))
	require.Equal(t, hub.AskReasserted, repeat.Outcome)
	require.Equal(t, "Backup steward", repeat.Ask.Sender)

	// A text change that names none keeps it too.
	changed := subjectSpec(later, "evt-3")
	changed.Question = "Backups have now been blocked for an hour. Retry or skip?"
	updated := askSubject(t, service, changed)
	require.Equal(t, hub.AskUpdated, updated.Outcome)
	require.Equal(t, "Backup steward", updated.Ask.Sender)

	// A replacement (new answer menu) is the same requester's decision.
	replaced := subjectSpec(later, "evt-4")
	replaced.Options = []hub.AskOption{{Key: "retry", Label: "Retry"}, {Key: "abandon", Label: "Abandon"}}
	replacement := askSubject(t, service, replaced)
	require.Equal(t, hub.AskReplaced, replacement.Outcome)
	require.NotEqual(t, opened.Ask.ID, replacement.Ask.ID)
	require.Equal(t, "Backup steward", replacement.Ask.Sender)

	// A request that names a sender renames the open ask, even when nothing else changed.
	renamed := subjectSpec(later, "evt-5")
	renamed.Options = replaced.Options
	renamed.Question = changed.Question
	renamed.Sender = "Backup manager"
	result := askSubject(t, service, renamed)
	require.Equal(t, replacement.Ask.ID, result.Ask.ID)
	require.Equal(t, "Backup manager", result.Ask.Sender)
	stored, err := service.GetAsk(context.Background(), replacement.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, "Backup manager", stored.Sender)
}

func TestDefaultSenderNamesANewAskAndNothingElse(t *testing.T) {
	ctx := context.Background()
	l := newListAsks(t)

	// A new direct ask with no declared sender takes the suggestion.
	spec := decisionSpec(listNow, "default-direct")
	spec.CreatedBy, spec.Source, spec.SourceEventType = "run:1", "alice", ""
	spec.DefaultSender = "research · steward-channel"
	suggested, _, err := l.service.AskWithSpec(ctx, spec)
	require.NoError(t, err)
	require.Equal(t, "research · steward-channel", suggested.Sender)

	// A declared sender outranks it.
	spec = decisionSpec(listNow, "default-declared")
	spec.CreatedBy, spec.Source, spec.SourceEventType = "run:1", "alice", ""
	spec.Sender = "Treasury steward"
	spec.DefaultSender = "research · steward-channel"
	declared, _, err := l.service.AskWithSpec(ctx, spec)
	require.NoError(t, err)
	require.Equal(t, "Treasury steward", declared.Sender)

	// A suggestion that breaks the sender rules is repaired, never refused.
	spec = decisionSpec(listNow, "default-repaired")
	spec.CreatedBy, spec.Source, spec.SourceEventType = "run:1", "alice", ""
	spec.DefaultSender = "  Backup\x00  steward\n" + strings.Repeat("x", 80)
	repaired, _, err := l.service.AskWithSpec(ctx, spec)
	require.NoError(t, err)
	require.Equal(t, "Backup steward "+strings.Repeat("x", 45), repaired.Sender, "blanks collapse, control characters drop, 60 characters stay")

	// A blank suggestion is no suggestion: the Hub names the caller itself.
	spec = decisionSpec(listNow, "default-blank")
	spec.CreatedBy, spec.Source, spec.SourceEventType = "run:1", "alice", ""
	spec.DefaultSender = " \t "
	blank, _, err := l.service.AskWithSpec(ctx, spec)
	require.NoError(t, err)
	require.Equal(t, hub.SenderAgentRun, blank.Sender)
}

func TestDefaultSenderNeverRenamesARevision(t *testing.T) {
	service, _ := testService(t)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := subjectSpec(start, "evt-1")
	first.DefaultSender = "research · first-session"
	opened := askSubject(t, service, first)
	require.Equal(t, "research · first-session", opened.Ask.Sender, "a new ask with no declared sender takes the suggestion")

	// A later run of the same ask suggests another name; the ask keeps its own.
	later := start.Add(time.Hour)
	at(service, later)
	changed := subjectSpec(later, "evt-2")
	changed.Question = "Backups have now been blocked for an hour. Retry or skip?"
	changed.DefaultSender = "research · second-session"
	updated := askSubject(t, service, changed)
	require.Equal(t, hub.AskUpdated, updated.Outcome)
	require.Equal(t, "research · first-session", updated.Ask.Sender, "a revision without a declared sender keeps the name")

	// A replacement keeps the name the operator knew the ask by.
	replaced := subjectSpec(later, "evt-3")
	replaced.Options = []hub.AskOption{{Key: "retry", Label: "Retry"}, {Key: "abandon", Label: "Abandon"}}
	replaced.DefaultSender = "research · third-session"
	replacement := askSubject(t, service, replaced)
	require.Equal(t, hub.AskReplaced, replacement.Outcome)
	require.Equal(t, "research · first-session", replacement.Ask.Sender)

	// A declared sender renames it, and the suggestion still changes nothing after.
	named := subjectSpec(later, "evt-4")
	named.Options = replaced.Options
	named.Sender = "Backup steward"
	named.DefaultSender = "research · fourth-session"
	renamed := askSubject(t, service, named)
	require.Equal(t, replacement.Ask.ID, renamed.Ask.ID)
	require.Equal(t, "Backup steward", renamed.Ask.Sender)
	again := subjectSpec(later, "evt-5")
	again.Options = replaced.Options
	again.DefaultSender = "research · fifth-session"
	kept := askSubject(t, service, again)
	require.Equal(t, "Backup steward", kept.Ask.Sender)
}

func TestAsksStoredBeforeSendersExistedAreNamedAtBoot(t *testing.T) {
	l := newListAsks(t)
	event := l.ask(raise{age: time.Minute, source: "research-desk"})
	run := l.ask(raise{age: time.Minute, direct: true, by: "run:1"})
	declared := l.ask(raise{age: time.Minute, sender: "Treasury steward"})
	l.set(event.ID, `sender = ''`)
	l.set(run.ID, `sender = ''`)

	l.service.BackfillAskSenders(context.Background())

	for id, want := range map[string]string{event.ID: "Research Desk", run.ID: hub.SenderAgentRun, declared.ID: "Treasury steward"} {
		var stored string
		require.NoError(t, l.db.QueryRowContext(context.Background(), `SELECT sender FROM asks WHERE id = ?`, id).Scan(&stored))
		require.Equal(t, want, stored)
	}
	// And a row that was never backfilled is still read under a name.
	l.set(run.ID, `sender = ''`)
	read, err := l.service.GetAsk(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, hub.SenderAgentRun, read.Sender)
	facets, err := l.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{OpenOnly: true})
	require.NoError(t, err)
	var listed []string
	for _, c := range facets.Facets.Senders {
		listed = append(listed, c.Value)
	}
	require.Contains(t, listed, hub.SenderAgentRun, "the sender facet names a row stored with none")
}

// brandedPreferences is a preferences seam that also knows source display names.
type brandedPreferences struct {
	stubPreferences
	names map[string]string
	db    *database.RoutedDB
}

func (b *brandedPreferences) ResolveBrand(ctx context.Context, source string) (string, string, string, bool) {
	if b.db != nil {
		// The real lookup reads the Hub's database. The store has one connection,
		// so this read waits for a connection another caller holds.
		var one int
		if err := b.db.QueryRowContext(ctx, `SELECT 1`).Scan(&one); err != nil {
			return "", "", "", false
		}
	}
	name, ok := b.names[source]
	return name, "", "", ok
}

func TestAskSenderIsTheNameTheHubRegisteredForTheSource(t *testing.T) {
	l := newListAsks(t)
	l.service.SetPreferences(&brandedPreferences{
		stubPreferences: stubPreferences{decision: hub.PreferenceDecision{Enabled: true}},
		names:           map[string]string{"research-desk": "Research Desk (Vrooli)", "agent-manager": ""},
	})
	registered := l.ask(raise{age: time.Minute, source: "research-desk"})
	require.Equal(t, "Research Desk (Vrooli)", registered.Sender, "a source with a registered name is shown by it")
	unregistered := l.ask(raise{age: time.Minute, source: "storage-manager"})
	require.Equal(t, "Storage Manager", unregistered.Sender, "a source with none is named from its scenario name")
	blank := l.ask(raise{age: time.Minute, source: "agent-manager"})
	require.Equal(t, "Agent Manager", blank.Sender, "an empty registered name falls back the same way")
}

// A dry run names the sender by the rule a real send uses, and says why, so a
// requester sees who the operator will read before anything is sent.
func TestCheckAskNamesTheSenderTheRealAskWouldGet(t *testing.T) {
	ctx := context.Background()
	service, _ := testService(t)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	check := func(spec hub.AskSpec) (string, hub.SenderSource) {
		t.Helper()
		got, err := service.CheckAsk(ctx, spec, hub.CheckInput{})
		require.NoError(t, err)
		require.NotEmpty(t, got.Sender, "the sender is never empty")
		return got.Sender, got.SenderSource
	}
	direct := func(key string) hub.AskSpec {
		spec := decisionSpec(start, key)
		spec.CreatedBy, spec.Source, spec.SourceEventType = "run:1", "alice", ""
		return spec
	}

	name, source := check(direct("c-run"))
	require.Equal(t, hub.SenderAgentRun, name)
	require.Equal(t, hub.SenderSourceAgentRun, source)

	spec := direct("c-default")
	spec.DefaultSender = "research · steward-channel"
	name, source = check(spec)
	require.Equal(t, "research · steward-channel", name)
	require.Equal(t, hub.SenderSourceDefault, source)

	spec.Sender = "Treasury steward"
	name, source = check(spec)
	require.Equal(t, "Treasury steward", name)
	require.Equal(t, hub.SenderSourceDeclared, source, "a declared sender outranks the default")

	name, source = check(decisionSpec(start, "c-event"))
	require.Equal(t, "Agent Manager", name)
	require.Equal(t, hub.SenderSourceService, source)

	spec = direct("c-unnamed")
	spec.CreatedBy = ""
	name, source = check(spec)
	require.Equal(t, hub.SenderUnnamed, name)
	require.Equal(t, hub.SenderSourceUnnamed, source)

	// A revision keeps the open ask's name; a declared sender renames; the
	// preview matches what the real send then stores.
	first := subjectSpec(start, "evt-1")
	first.Sender = "Backup steward"
	opened := askSubject(t, service, first)

	revision := subjectSpec(start.Add(time.Hour), "evt-2")
	revision.DefaultSender = "research · another-session"
	name, source = check(revision)
	require.Equal(t, "Backup steward", name)
	require.Equal(t, hub.SenderSourceKept, source, "the default never renames a revision")
	sent := askSubject(t, service, revision)
	require.Equal(t, name, sent.Ask.Sender)
	require.Equal(t, opened.Ask.ID, sent.Ask.ID)

	revision.Sender = "Backup manager"
	name, source = check(revision)
	require.Equal(t, "Backup manager", name)
	require.Equal(t, hub.SenderSourceDeclared, source)
}

// The Hub has one database connection and an ask is stored in one write
// transaction that holds it. Looking up a source's registered name inside that
// transaction waits on itself until the request times out, and every event that
// raises an ask fails. The name must be read before the transaction opens, on
// every path that stores an ask.
func TestAskedSourceNameIsReadBeforeTheStoringTransactionOpens(t *testing.T) {
	service, db := testService(t)
	service.SetPreferences(&brandedPreferences{
		stubPreferences: stubPreferences{decision: hub.PreferenceDecision{Enabled: true}},
		names:           map[string]string{"agent-manager": "Agent Manager (Vrooli)", "backup-watcher": "Backup watcher"},
		db:              db,
	})
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	plain, _, err := service.AskWithSpec(ctx, decisionSpec(start, "tx-plain"))
	require.NoError(t, err)
	require.Equal(t, "Agent Manager (Vrooli)", plain.Sender, "an event ask without a subject is named from its registered source")

	subject := subjectSpec(start, "tx-subject")
	subject.Source = "backup-watcher"
	keyed, err := service.AskDecision(ctx, subject, "")
	require.NoError(t, err)
	require.Equal(t, "Backup watcher", keyed.Ask.Sender, "a keyed ask is named the same way")
	require.NoError(t, ctx.Err(), "neither ask waited out the request")
}
