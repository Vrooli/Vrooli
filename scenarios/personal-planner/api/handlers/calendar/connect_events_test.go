package calendar

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar"
	vc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar/calendar_v1connect"
	wv "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/workspace"
	wvc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/workspace/workspace_v1connect"
	_ "modernc.org/sqlite"

	healthHandler "personal-planner/handlers/health"
	workspaceHandler "personal-planner/handlers/workspace"
	d "personal-planner/internal/calendar"
)

func TestCalendarEventConnectCRUDKeepsIdentityAndRevision(t *testing.T) {
	repository := &httpEventRepository{events: map[string]d.Event{}}
	router := mux.NewRouter()
	path, handler := vc.NewCalendarServiceHandler(NewConnectHandler(Deps{Events: d.NewEventService(repository)}))
	router.PathPrefix(path).Handler(handler)
	server := httptest.NewServer(router)
	defer server.Close()
	client := vc.NewCalendarServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()

	createdResponse, err := client.CreateEvent(ctx, connect.NewRequest(&v.CreateEventRequest{
		IdempotencyKey: "synthetic-connect-e1",
		Event:          &v.CalendarEvent{Title: "Synthetic Connect event", Subject: "owner fixture", Availability: "busy", Timezone: "America/New_York", AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-07"},
	}))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	created := createdResponse.Msg.Event
	if created.Id != "event-http-1" || created.Revision != 1 {
		t.Fatalf("created identity/revision = %s/%d, want event-http-1/1", created.Id, created.Revision)
	}
	if got := created.GetCreatedAt().AsTime(); !got.Equal(time.Unix(1, 0)) {
		t.Fatalf("created_at=%s, want deterministic repository timestamp", got)
	}

	updatedResponse, err := client.UpdateEvent(ctx, connect.NewRequest(&v.UpdateEventRequest{
		ExpectedRevision: 1,
		Event:            &v.CalendarEvent{Id: created.Id, Title: "Synthetic Connect edited", Subject: "owner fixture", Availability: "busy", Timezone: "America/New_York", AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-07"},
	}))
	if err != nil {
		t.Fatalf("update event: %v", err)
	}
	if got := updatedResponse.Msg.Event; got.Id != created.Id || got.Revision != 2 || got.Title != "Synthetic Connect edited" {
		t.Fatalf("updated event identity/revision/title = %s/%d/%q", got.Id, got.Revision, got.Title)
	}

	getResponse, err := client.GetEvent(ctx, connect.NewRequest(&v.GetEventRequest{EventId: created.Id}))
	if err != nil {
		t.Fatalf("get event: %v", err)
	}
	if got := getResponse.Msg.Event; got.Id != created.Id || got.Revision != 2 || got.Title != "Synthetic Connect edited" {
		t.Fatalf("reopened event identity/revision/title = %s/%d/%q", got.Id, got.Revision, got.Title)
	}

	listResponse, err := client.ListEvents(ctx, connect.NewRequest(&v.ListEventsRequest{StartLocalDate: "2026-10-05", EndLocalDate: "2026-10-05"}))
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(listResponse.Msg.Events) != 1 || listResponse.Msg.Events[0].Id != created.Id || listResponse.Msg.Events[0].Revision != 2 {
		t.Fatalf("list did not return same durable event: %#v", listResponse.Msg.Events)
	}

	_, err = client.UpdateEvent(ctx, connect.NewRequest(&v.UpdateEventRequest{ExpectedRevision: 1, Event: &v.CalendarEvent{Id: created.Id, Title: "stale", AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-07", Timezone: "America/New_York", Availability: "busy"}}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale update code=%s err=%v, want ABORTED", connect.CodeOf(err), err)
	}
}

func TestCalendarRESTConnectAndWorkspaceShareRoutedTestPoolIdentityAndRevisions(t *testing.T) {
	t.Setenv(apihttp.TestModeForceEnableEnv, "1")
	ctx := context.Background()
	testDir := t.TempDir()
	routed, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, Scenario: "personal-planner", DSN: filepath.Join(testDir, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer routed.Close()

	schemas := []database.SchemaProvider{
		database.SchemaProviderFunc(Schema),
		database.SchemaProviderFunc(workspaceHandler.Schema),
	}
	if err := database.EnsureSchemas(ctx, routed.Primary(), schemas...); err != nil {
		t.Fatalf("initialize primary schemas: %v", err)
	}
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(ctx, pool, schemas...)
	})
	const leaseID = "planner-connect-parity-test"
	if err := routed.InstallTestPool(ctx, filepath.Join(testDir, "test-pool.db"), leaseID, time.Hour); err != nil {
		t.Fatalf("install synthetic routed test pool: %v", err)
	}
	defer func() {
		if err := routed.ClearTestPool(leaseID); err != nil {
			t.Errorf("clear synthetic test pool: %v", err)
		}
	}()

	logger := log.New(io.Discard, "", 0)
	router := mux.NewRouter()
	calendarModule := Module(routed, schedule.System(), logger)
	workspaceModule := workspaceHandler.Module(routed, schedule.System(), logger)
	healthModule := healthHandler.Module(routed, "personal-planner-api", "test")
	calendarModule.Mount(router)
	workspaceModule.Mount(router)
	healthModule.Mount(router)
	actors := &observedActors{}
	authenticated := authn.Middleware(authn.Config{Providers: []authn.Provider{syntheticActorProvider{}}})(actors.wrap(router))
	server := httptest.NewServer(apihttp.TestModeMiddleware(authenticated))
	defer server.Close()

	// The UI's event client uses the REST surface. Its test-mode header must
	// route this write to the installed temporary pool.
	createReq, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/calendar/events", strings.NewReader(`{"title":"Synthetic routed event","subject":"fixture owner","availability":"busy","timezone":"America/New_York","all_day":true,"start_date":"2099-11-01","end_date_exclusive":"2099-11-03","idempotency_key":"routed-connect-parity"}`))
	if err != nil {
		t.Fatal(err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set(apihttp.TestModeHeader, apihttp.TestModeValue)
	createReq.Header.Set("Authorization", "Bearer synthetic-planner-actor")
	createResponse, err := server.Client().Do(createReq)
	if err != nil {
		t.Fatalf("UI-style REST create: %v", err)
	}
	var created d.Event
	if createResponse.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResponse.Body)
		createResponse.Body.Close()
		t.Fatalf("create status=%d body=%s", createResponse.StatusCode, body)
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		createResponse.Body.Close()
		t.Fatalf("decode created event: %v", err)
	}
	createResponse.Body.Close()
	if created.ID == "" || created.Revision != 1 {
		t.Fatalf("REST create identity/revision=%s/%d, want nonempty ID/1", created.ID, created.Revision)
	}

	connectHTTP := testModeConnectHTTPClient{HTTPClient: server.Client()}
	eventsClient := vc.NewCalendarServiceClient(connectHTTP, server.URL)
	listed, err := eventsClient.ListEvents(ctx, connect.NewRequest(&v.ListEventsRequest{StartLocalDate: "2099-11-01", EndLocalDate: "2099-11-02"}))
	if err != nil {
		t.Fatalf("CLI/Connect list: %v", err)
	}
	if len(listed.Msg.Events) != 1 || listed.Msg.Events[0].Id != created.ID || listed.Msg.Events[0].Revision != 1 {
		t.Fatalf("Connect list disagrees with REST create: %#v created=%#v", listed.Msg.Events, created)
	}

	// A second verified principal cannot observe or mutate the first owner's
	// event through either handler surface. Also seed a legacy row with no
	// owner relation and verify it remains inaccessible to owner A.
	legacyID := "legacy-unowned-connect-fixture"
	if err := routed.QueryRowContext(database.WithTestMode(ctx), `SELECT id FROM calendar_events WHERE id=?`, legacyID).Scan(new(string)); err == nil {
		t.Fatal("legacy fixture ID already exists in isolated test pool")
	}
	if _, err := routed.ExecContext(database.WithTestMode(ctx), `INSERT INTO calendar_events (id,title,subject,availability,timezone,all_day,start_date,end_date_exclusive,range_start,range_end,revision,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, legacyID, "Legacy private fixture", "synthetic-planner-actor", "busy", "America/New_York", 1, "2099-11-01", "2099-11-02", "2099-11-01", "2099-11-02", 9, "2099-01-01T00:00:00Z", "2099-01-01T00:00:00Z"); err != nil {
		t.Fatalf("insert isolated unowned legacy fixture: %v", err)
	}
	requestAs := func(token, method, path, body string) *http.Response {
		t.Helper()
		request, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set(apihttp.TestModeHeader, apihttp.TestModeValue)
		request.Header.Set("Authorization", "Bearer "+token)
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatalf("owner B %s %s: %v", method, path, err)
		}
		return response
	}
	ownerBHTTP := func(method, path, body string) *http.Response {
		return requestAs("synthetic-planner-owner-b", method, path, body)
	}
	otherList := ownerBHTTP(http.MethodGet, "/api/v1/calendar/events?start_local_date=2099-11-01&end_local_date=2099-11-02", "")
	var otherEvents eventListResponse
	if otherList.StatusCode != http.StatusOK || json.NewDecoder(otherList.Body).Decode(&otherEvents) != nil || len(otherEvents.Events) != 0 {
		body, _ := io.ReadAll(otherList.Body)
		otherList.Body.Close()
		t.Fatalf("owner B REST list status=%d events=%#v body=%s", otherList.StatusCode, otherEvents.Events, body)
	}
	otherList.Body.Close()
	for _, id := range []string{created.ID, legacyID} {
		response := ownerBHTTP(http.MethodGet, "/api/v1/calendar/events/"+id, "")
		if response.StatusCode != http.StatusNotFound {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			t.Fatalf("owner B REST get %s status=%d body=%s", id, response.StatusCode, body)
		}
		response.Body.Close()
	}
	foreignUpdate := `{"title":"unauthorized owner B edit","availability":"busy","timezone":"America/New_York","all_day":true,"start_date":"2099-11-01","end_date_exclusive":"2099-11-03","expected_revision":3}`
	for _, id := range []string{created.ID, legacyID} {
		response := ownerBHTTP(http.MethodPut, "/api/v1/calendar/events/"+id, foreignUpdate)
		if response.StatusCode != http.StatusNotFound {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			t.Fatalf("owner B REST update %s status=%d body=%s", id, response.StatusCode, body)
		}
		response.Body.Close()
	}
	connectB := vc.NewCalendarServiceClient(testModeConnectHTTPClient{HTTPClient: server.Client(), authorization: "Bearer synthetic-planner-owner-b"}, server.URL)
	connectBList, err := connectB.ListEvents(ctx, connect.NewRequest(&v.ListEventsRequest{StartLocalDate: "2099-11-01", EndLocalDate: "2099-11-02"}))
	if err != nil || len(connectBList.Msg.Events) != 0 {
		t.Fatalf("owner B Connect list=%#v err=%v, want empty", connectBList, err)
	}
	for _, id := range []string{created.ID, legacyID} {
		_, err := connectB.GetEvent(ctx, connect.NewRequest(&v.GetEventRequest{EventId: id}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("owner B Connect get %s code=%s err=%v", id, connect.CodeOf(err), err)
		}
		_, err = connectB.UpdateEvent(ctx, connect.NewRequest(&v.UpdateEventRequest{ExpectedRevision: 3, Event: &v.CalendarEvent{Id: id, Title: "unauthorized", Availability: "busy", Timezone: "America/New_York", AllDay: true, StartDate: "2099-11-01", EndDateExclusive: "2099-11-03"}}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("owner B Connect update %s code=%s err=%v", id, connect.CodeOf(err), err)
		}
	}
	legacyOwnerA := requestAs("synthetic-planner-actor", http.MethodGet, "/api/v1/calendar/events/"+legacyID, "")
	legacyOwnerA.Body.Close()
	if legacyOwnerA.StatusCode != http.StatusNotFound {
		t.Fatalf("owner A REST get of unowned legacy row status=%d", legacyOwnerA.StatusCode)
	}
	legacyOwnerAEdit := requestAs("synthetic-planner-actor", http.MethodPut, "/api/v1/calendar/events/"+legacyID, foreignUpdate)
	legacyOwnerAEdit.Body.Close()
	if legacyOwnerAEdit.StatusCode != http.StatusNotFound {
		t.Fatalf("owner A REST update of unowned legacy row status=%d", legacyOwnerAEdit.StatusCode)
	}
	for _, op := range []string{"get", "update"} {
		if op == "get" {
			_, err = eventsClient.GetEvent(ctx, connect.NewRequest(&v.GetEventRequest{EventId: legacyID}))
		} else {
			_, err = eventsClient.UpdateEvent(ctx, connect.NewRequest(&v.UpdateEventRequest{ExpectedRevision: 9, Event: &v.CalendarEvent{Id: legacyID, Title: "changed", Availability: "busy", Timezone: "America/New_York", AllDay: true, StartDate: "2099-11-01", EndDateExclusive: "2099-11-02"}}))
		}
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("owner A Connect %s of unowned legacy row code=%s err=%v", op, connect.CodeOf(err), err)
		}
	}
	cliBinary := buildPlannerCLI(t)
	cliBaseArgs := []string{"--api-base", server.URL}
	cliEnv := []string{"PERSONAL_PLANNER_API_BASE=" + server.URL, "PERSONAL_PLANNER_TEST_MODE=1", "VROOLI_API_TOKEN=synthetic-planner-actor"}
	cliList := runPlannerCLI(t, cliBinary, cliEnv, append(append([]string{}, cliBaseArgs...), "calendar", "events", "--start-date", "2099-11-01", "--end-date", "2099-11-02")...)
	if !strings.Contains(cliList, created.ID) || !strings.Contains(cliList, "rev 1") {
		t.Fatalf("CLI event list did not return the REST-created event at revision 1: %s", cliList)
	}
	cliGet := runPlannerCLI(t, cliBinary, cliEnv, append(append([]string{}, cliBaseArgs...), "calendar", "get-event", "--event-id", created.ID)...)
	if !strings.Contains(cliGet, created.ID) || !strings.Contains(cliGet, "revision 1") {
		t.Fatalf("CLI get did not return the REST-created event at revision 1: %s", cliGet)
	}
	cliUpdate := runPlannerCLI(t, cliBinary, cliEnv, append(append([]string{}, cliBaseArgs...), "calendar", "update-event", "--event-id", created.ID, "--revision", "1", "--title", "CLI updated routed event", "--subject", "fixture owner", "--availability", "free", "--timezone", "America/New_York", "--all-day", "true", "--start-date", "2099-11-01", "--end-date-exclusive", "2099-11-03")...)
	if !strings.Contains(cliUpdate, created.ID) || !strings.Contains(cliUpdate, "revision 2") {
		t.Fatalf("CLI update did not preserve event identity and advance to revision 2: %s", cliUpdate)
	}
	cliUpdated := &v.CalendarEvent{Id: created.ID, Title: "CLI updated routed event", Subject: "fixture owner", Availability: "free", Timezone: "America/New_York", AllDay: true, StartDate: "2099-11-01", EndDateExclusive: "2099-11-03"}
	updated, err := eventsClient.UpdateEvent(ctx, connect.NewRequest(&v.UpdateEventRequest{Event: cliUpdated, ExpectedRevision: 2}))
	if err != nil {
		t.Fatalf("Connect update after CLI revision: %v", err)
	}
	if updated.Msg.Event.Id != created.ID || updated.Msg.Event.Revision != 3 {
		t.Fatalf("Connect update identity/revision=%s/%d, want %s/3", updated.Msg.Event.Id, updated.Msg.Event.Revision, created.ID)
	}
	reopened, err := eventsClient.GetEvent(ctx, connect.NewRequest(&v.GetEventRequest{EventId: created.ID}))
	if err != nil {
		t.Fatalf("Connect get: %v", err)
	}
	if reopened.Msg.Event.Id != created.ID || reopened.Msg.Event.Revision != 3 || reopened.Msg.Event.Title != "CLI updated routed event" {
		t.Fatalf("Connect reopen identity/revision/title=%s/%d/%q", reopened.Msg.Event.Id, reopened.Msg.Event.Revision, reopened.Msg.Event.Title)
	}
	cliReadback := runPlannerCLI(t, cliBinary, cliEnv, append(append([]string{}, cliBaseArgs...), "calendar", "get-event", "--event-id", created.ID)...)
	if !strings.Contains(cliReadback, created.ID) || !strings.Contains(cliReadback, "revision 3") {
		t.Fatalf("CLI readback did not observe Connect revision 3: %s", cliReadback)
	}

	// UI and CLI workspace clients both use this same WorkspaceService Connect
	// contract. The actual CLI process reads and updates the same profile; the
	// Connect readback proves its durable owner and revision.
	workspaceClient := wvc.NewWorkspaceServiceClient(connectHTTP, server.URL)
	profile, err := workspaceClient.GetProfile(ctx, connect.NewRequest(&wv.GetProfileRequest{}))
	if err != nil {
		t.Fatalf("workspace profile read: %v", err)
	}
	if profile.Msg.Profile == nil || profile.Msg.Profile.Id != "default" || profile.Msg.Profile.Revision != 1 {
		t.Fatalf("initial profile identity/revision=%#v, want default/1", profile.Msg.Profile)
	}
	cliProfile := runPlannerCLI(t, cliBinary, cliEnv, append(append([]string{}, cliBaseArgs...), "workspace", "profile")...)
	if !strings.Contains(cliProfile, "timezone=UTC") || !strings.Contains(cliProfile, "revision=1") {
		t.Fatalf("CLI profile did not read the same initial workspace profile: %s", cliProfile)
	}
	cliProfileUpdate := runPlannerCLI(t, cliBinary, cliEnv, append(append([]string{}, cliBaseArgs...), "workspace", "update-profile", "--timezone", "America/New_York", "--week-start", "monday", "--daily-capacity-minutes", "420", "--reserve-minutes", "45", "--focus-session-minutes", "40", "--revision", "1")...)
	if !strings.Contains(cliProfileUpdate, "revision 2") {
		t.Fatalf("CLI profile update did not advance shared profile to revision 2: %s", cliProfileUpdate)
	}
	profileReadback, err := workspaceClient.GetProfile(ctx, connect.NewRequest(&wv.GetProfileRequest{}))
	if err != nil {
		t.Fatalf("workspace profile readback: %v", err)
	}
	if profileReadback.Msg.Profile.Id != profile.Msg.Profile.Id || profileReadback.Msg.Profile.Revision != 2 || profileReadback.Msg.Profile.Timezone != "America/New_York" {
		t.Fatalf("workspace profile owner/revision readback mismatch: initial=%#v readback=%#v", profile.Msg.Profile, profileReadback.Msg.Profile)
	}

	// Independently read through the UI REST shape and assert the same durable
	// event record; also prove that no request escaped to primary storage.
	readReq, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/calendar/events/"+created.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	readReq.Header.Set(apihttp.TestModeHeader, apihttp.TestModeValue)
	readReq.Header.Set("Authorization", "Bearer synthetic-planner-actor")
	readResponse, err := server.Client().Do(readReq)
	if err != nil {
		t.Fatalf("UI-style REST readback: %v", err)
	}
	var uiReadback d.Event
	if readResponse.StatusCode != http.StatusOK || json.NewDecoder(readResponse.Body).Decode(&uiReadback) != nil {
		body, _ := io.ReadAll(readResponse.Body)
		readResponse.Body.Close()
		t.Fatalf("REST readback status=%d body=%s", readResponse.StatusCode, body)
	}
	readResponse.Body.Close()
	if uiReadback.ID != created.ID || uiReadback.Revision != 3 || uiReadback.Title != reopened.Msg.Event.Title {
		t.Fatalf("REST/Connect event readback mismatch: REST=%#v Connect=%#v", uiReadback, reopened.Msg.Event)
	}
	runPlannerUITypescriptParity(t, server.URL, created.ID, cliBinary)

	var testPoolEvents, primaryEvents, testPoolEventRevision int64
	var testPoolEventTitle string
	if err := routed.QueryRowContext(database.WithTestMode(ctx), `SELECT COUNT(*), MAX(revision), MAX(title) FROM calendar_events e JOIN calendar_event_owners o ON o.event_id=e.id WHERE o.owner_subject='synthetic-planner-actor'`).Scan(&testPoolEvents, &testPoolEventRevision, &testPoolEventTitle); err != nil {
		t.Fatal(err)
	}
	if err := routed.Primary().QueryRowContext(ctx, `SELECT COUNT(*) FROM calendar_events`).Scan(&primaryEvents); err != nil {
		t.Fatal(err)
	}
	var testPoolProfileRevision, primaryProfiles, primaryActorProfiles, legacyTestPoolProfiles int64
	var testPoolProfileTimezone, testPoolProfileWeekStart string
	var testPoolProfileCapacity, testPoolProfileReserve, testPoolProfileFocus int
	if err := routed.QueryRowContext(database.WithTestMode(ctx), `SELECT revision,timezone,week_start,daily_capacity_minutes,reserve_minutes,focus_session_minutes FROM actor_planning_profiles WHERE owner_subject=? AND id='default'`, "synthetic-planner-actor").Scan(&testPoolProfileRevision, &testPoolProfileTimezone, &testPoolProfileWeekStart, &testPoolProfileCapacity, &testPoolProfileReserve, &testPoolProfileFocus); err != nil {
		t.Fatalf("read test-pool actor profile: %v", err)
	}
	if err := routed.QueryRowContext(database.WithTestMode(ctx), `SELECT COUNT(*) FROM planning_profiles WHERE id='default'`).Scan(&legacyTestPoolProfiles); err != nil {
		t.Fatalf("count test-pool legacy profiles: %v", err)
	}
	if err := routed.Primary().QueryRowContext(ctx, `SELECT COUNT(*) FROM planning_profiles WHERE id='default'`).Scan(&primaryProfiles); err != nil {
		t.Fatalf("count primary profiles: %v", err)
	}
	if err := routed.Primary().QueryRowContext(ctx, `SELECT COUNT(*) FROM actor_planning_profiles`).Scan(&primaryActorProfiles); err != nil {
		t.Fatalf("count primary actor profiles: %v", err)
	}
	stats := routed.LeaseStats()
	if testPoolEvents != 1 || primaryEvents != 0 || testPoolEventRevision != 4 || testPoolEventTitle != "UI TypeScript client event" || testPoolProfileRevision != 3 || testPoolProfileTimezone != "America/New_York" || testPoolProfileWeekStart != "sunday" || testPoolProfileCapacity != 400 || testPoolProfileReserve != 40 || testPoolProfileFocus != 35 || legacyTestPoolProfiles != 0 || primaryProfiles != 0 || primaryActorProfiles != 0 || stats.TestPoolRequests == 0 || stats.PrimaryDuringTestModeRequests != 0 {
		t.Fatalf("routing proof: test_events=%d/%d/%q primary_events=%d actor_profile=%d/%s/%s/%d/%d/%d legacy_test_profiles=%d primary_profiles=%d primary_actor_profiles=%d lease_stats=%+v", testPoolEvents, testPoolEventRevision, testPoolEventTitle, primaryEvents, testPoolProfileRevision, testPoolProfileTimezone, testPoolProfileWeekStart, testPoolProfileCapacity, testPoolProfileReserve, testPoolProfileFocus, legacyTestPoolProfiles, primaryProfiles, primaryActorProfiles, stats)
	}
	if got := actors.snapshot(); len(got) < 19 {
		t.Fatalf("authenticated actor observed on %d requests, want the REST/CLI/Connect parity journeys", len(got))
	} else {
		ownerBObserved := false
		for _, actor := range got {
			switch actor {
			case "synthetic-planner-actor":
			case "synthetic-planner-owner-b":
				ownerBObserved = true
			default:
				t.Fatalf("REST/CLI/Connect request actor=%q, want one of the verified fixture actors; all=%v", actor, got)
			}
		}
		if !ownerBObserved {
			t.Fatal("cross-owner requests did not reach the API with the verified owner B actor")
		}
	}

}

func runPlannerUITypescriptParity(t *testing.T, apiBase, eventID, cliBinary string) {
	t.Helper()
	uiRoot := filepath.Join("..", "..", "..", "ui")
	command := exec.Command("pnpm", "exec", "vitest", "run", "src/api/parity.integration.test.ts", "--reporter=verbose", "--maxWorkers=1", "--minWorkers=1")
	command.Dir = uiRoot
	command.Env = withoutEnv(os.Environ(), "PLANNER_PARITY_API_BASE", "PLANNER_PARITY_EVENT_ID", "PLANNER_PARITY_CLI", "PLANNER_PARITY_ACTOR_TOKEN", "PERSONAL_PLANNER_API_BASE", "PERSONAL_PLANNER_API_URL", "PERSONAL_PLANNER_API_PORT", "PERSONAL_PLANNER_API_TOKEN", "PERSONAL_PLANNER_CONFIG_DIR", "VROOLI_CLI_CONFIG_DIR", "API_BASE_URL", "VITE_API_BASE_URL", "PERSONAL_PLANNER_TEST_MODE", "VROOLI_API_TOKEN", "VROOLI_AGENT_IDENTITY_TOKEN")
	command.Env = append(command.Env,
		"PLANNER_PARITY_API_BASE="+apiBase,
		"PLANNER_PARITY_EVENT_ID="+eventID,
		"PLANNER_PARITY_CLI="+cliBinary,
		"PLANNER_PARITY_ACTOR_TOKEN=synthetic-planner-actor",
		"PERSONAL_PLANNER_API_BASE="+apiBase,
		"PERSONAL_PLANNER_TEST_MODE=1",
		"VROOLI_API_TOKEN=synthetic-planner-actor",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run Planner TypeScript UI parity test: %v\n%s", err, output)
	}
}

func buildPlannerCLI(t *testing.T) string {
	t.Helper()
	cliRoot := filepath.Join("..", "..", "..", "cli")
	binary := filepath.Join(t.TempDir(), "personal-planner")
	command := exec.Command("go", "build", "-o", binary, ".")
	command.Dir = cliRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build supported Planner CLI: %v\n%s", err, output)
	}
	return binary
}

func runPlannerCLI(t *testing.T, binary string, extraEnv []string, args ...string) string {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Env = withoutEnv(os.Environ(), "PERSONAL_PLANNER_API_BASE", "PERSONAL_PLANNER_API_URL", "PERSONAL_PLANNER_API_PORT", "PERSONAL_PLANNER_API_TOKEN", "PERSONAL_PLANNER_CONFIG_DIR", "VROOLI_CLI_CONFIG_DIR", "API_BASE_URL", "VITE_API_BASE_URL", "PERSONAL_PLANNER_TEST_MODE", "VROOLI_API_TOKEN", "VROOLI_AGENT_IDENTITY_TOKEN")
	command.Env = append(command.Env, extraEnv...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run Planner CLI %q: %v\n%s", args, err, output)
	}
	return string(output)
}

func withoutEnv(env []string, keys ...string) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		remove := false
		for _, blocked := range keys {
			if key == blocked {
				remove = true
				break
			}
		}
		if !remove {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

type testModeConnectHTTPClient struct {
	connect.HTTPClient
	authorization string
}

func (c testModeConnectHTTPClient) Do(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	if clone.Header == nil {
		clone.Header = make(http.Header)
	}
	clone.Header.Set(apihttp.TestModeHeader, apihttp.TestModeValue)
	authorization := c.authorization
	if authorization == "" {
		authorization = "Bearer synthetic-planner-actor"
	}
	clone.Header.Set("Authorization", authorization)
	return c.HTTPClient.Do(clone)
}

type syntheticActorProvider struct{}

func (syntheticActorProvider) Source() identity.AuthSource {
	return identity.SourceScenarioAuthenticator
}

func (syntheticActorProvider) VerifyRequest(_ context.Context, req *http.Request) (identity.Principal, error) {
	const bearerPrefix = "Bearer synthetic-planner-"
	token := strings.TrimPrefix(req.Header.Get("Authorization"), bearerPrefix)
	if token == req.Header.Get("Authorization") || token == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceScenarioAuthenticator)
	}
	return identity.Principal{Kind: identity.ActorHuman, Subject: "synthetic-planner-" + token, Verified: true, Source: identity.SourceScenarioAuthenticator}, nil
}

type observedActors struct {
	mu     sync.Mutex
	actors []string
}

func (a *observedActors) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		principal, ok := identity.PrincipalFromContext(req.Context())
		if !ok {
			http.Error(w, "verified actor missing", http.StatusUnauthorized)
			return
		}
		a.mu.Lock()
		a.actors = append(a.actors, principal.Subject)
		a.mu.Unlock()
		next.ServeHTTP(w, req)
	})
}

func (a *observedActors) snapshot() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.actors...)
}
