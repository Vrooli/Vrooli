package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/catalog"
	svc "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/catalog/catalog_v1connect"
	"music-library/internal/catalog"
)

// This opt-in contract exercise runs the real CLI against the real Connect
// handler, using only the six already-judged Aquila records in an isolated
// temporary database. It never touches the serving Music Library instance or
// reads, copies, or generates audio.
func TestMQ5CLIReservationContract(t *testing.T) {
	if os.Getenv("MQ5_CONTRACT") != "1" {
		t.Skip("set MQ5_CONTRACT=1 to exercise the real CLI and API contract")
	}

	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)))
	routed := apidb.NewFromPrimary(db)
	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }).Methods(http.MethodGet)
	Module(routed).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	client := svc.NewCatalogServiceClient(server.Client(), server.URL)
	for _, track := range aquilaBankFixtures() {
		_, err := client.PutTrack(context.Background(), connect.NewRequest(&pb.PutTrackRequest{Track: track}))
		require.NoError(t, err, "seed existing judged record %s", track.Title)
	}
	_, err := client.RecordUse(context.Background(), connect.NewRequest(&pb.RecordUseRequest{
		Sha256: aquilaAHash,
		Use:    &pb.TrackUse{Project: "aquila-launch", Revision: "constellation-score-a-review", Status: "draft", Purpose: "launch-video"},
	}))
	require.NoError(t, err)

	cli := buildMusicLibraryCLI(t)
	runCLI := func(args ...string) string {
		commandArgs := append([]string{"--api-base", server.URL, "--no-color"}, args...)
		command := exec.Command(cli, commandArgs...)
		output, runErr := command.CombinedOutput()
		require.NoError(t, runErr, "%s\n%s", strings.Join(commandArgs, " "), output)
		return string(output)
	}
	reservationFrom := func(output string) string {
		match := regexp.MustCompile(`reservation ([0-9a-f]{32})`).FindStringSubmatch(output)
		require.Len(t, match, 2, "reservation id should be returned to the CLI consumer: %s", output)
		return match[1]
	}

	firstOutput := runCLI("catalog", "reserve", "--purpose", "launch-video", "--project", "mq5-contract", "--revision", "cut-1", "--consumer", "mq5-run-1")
	require.Contains(t, firstOutput, aquilaCHash, "the preferred unused C record should be selected after A's existing draft binding")
	firstReservation := reservationFrom(firstOutput)

	secondOutput := runCLI("catalog", "reserve", "--purpose", "launch-video", "--project", "mq5-contract", "--revision", "cut-2", "--consumer", "mq5-run-2")
	require.Contains(t, secondOutput, aquilaBHash, "an active lease must not be offered to a second consumer")
	secondReservation := reservationFrom(secondOutput)
	runCLI("catalog", "release", "--reservation", secondReservation, "--consumer", "mq5-run-2")

	runCLI("catalog", "commit", "--reservation", firstReservation, "--consumer", "mq5-run-1", "--status", "draft")
	unused := runCLI("catalog", "list", "--purpose", "launch-video", "--unused", "true")
	require.NotContains(t, unused, aquilaCHash, "a committed draft is already bound to its film")
	require.NotContains(t, unused, aquilaAHash, "the pre-existing draft binding remains in use")
	require.Contains(t, unused, aquilaBHash, "releasing a lease returns that track to the available bank")
	runCLI("catalog", "rollback", "--reservation", firstReservation, "--consumer", "mq5-run-1")
	var outboxState string
	require.NoError(t, db.QueryRow(`SELECT state FROM music_catalog_replenish_outbox WHERE event_id=?`, firstReservation).Scan(&outboxState))
	require.Equal(t, "cancelled", outboxState, "rolling back a draft cancels its not-yet-delivered refill signal")
	unusedAfterRollback := runCLI("catalog", "list", "--purpose", "launch-video", "--unused", "true")
	require.Contains(t, unusedAfterRollback, aquilaCHash, "a rolled-back draft returns to the eligible bank")

	reuseOutput := runCLI("catalog", "reserve", "--purpose", "launch-video", "--project", "mq5-contract", "--revision", "cut-3", "--consumer", "mq5-run-3", "--sha256", aquilaCHash)
	require.Contains(t, reuseOutput, aquilaCHash, "reuse of a used liked track must require and honor an explicit SHA-256")
	reuseReservation := reservationFrom(reuseOutput)
	runCLI("catalog", "release", "--reservation", reuseReservation, "--consumer", "mq5-run-3")
}

const (
	aquilaAHash = "db2e8a4ce96d5f3949aedfc36cacb3a63d902694a74aeb7bd0595b4914e18ed1"
	aquilaBHash = "5bb8dbe2c2d07e9466d850cb18262da1e10d8222671430a203aaf9d0ba686cb4"
	aquilaCHash = "a6ab5d3585ec7fd9bf29e3aa973509a3513de8f39d8feeaa774d94986f4f167c"
)

func aquilaBankFixtures() []*pb.Track {
	root := "/home/matthalloran8/.vrooli/research/topics/music-quality/runs/launch-instrumentals-20261029"
	return []*pb.Track{
		{Sha256: aquilaAHash, Title: "a-prismatic", SourcePath: filepath.Join(root, "a-prismatic.wav"), TakeId: "a225142e-6948-41d2-91ad-a17332c84c7c-take-01", ReceiptPath: filepath.Join(root, "a-receipt.json"), Tags: []string{"instrumental", "launch-video", "electronic"}, Purposes: []string{"launch-video"}, BankPriority: 1, Approval: "liked", Feedback: "Operator preferred A and confirmed its pacing works with the film."},
		{Sha256: aquilaBHash, Title: "b-chrome", SourcePath: filepath.Join(root, "b-chrome.wav"), TakeId: "cee1fe41-7188-4046-8c91-ba8c5c9ddae9-take-01", ReceiptPath: filepath.Join(root, "b-receipt.json"), Tags: []string{"instrumental", "launch-video", "electronic"}, Purposes: []string{"launch-video"}, BankPriority: 100, Approval: "liked", Feedback: "Operator liked this track."},
		{Sha256: aquilaCHash, Title: "c-orbit", SourcePath: filepath.Join(root, "c-orbit.wav"), TakeId: "7d7b9ed8-1086-4dfe-af7e-01cf14bc2e38-take-01", ReceiptPath: filepath.Join(root, "c-receipt.json"), Tags: []string{"instrumental", "launch-video", "electronic"}, Purposes: []string{"launch-video"}, BankPriority: 2, Approval: "liked", Feedback: "Operator ranked C second."},
		{Sha256: "b63a9ce898bf8ca9b359d54359524e43d5f08f6edebb384d2eb5c0f556b5f6af", Title: "d-breakline", SourcePath: filepath.Join(root, "d-breakline.wav"), TakeId: "652d7408-6f4e-4955-a170-027152fbea9e-take-01", ReceiptPath: filepath.Join(root, "d-receipt.json"), Tags: []string{"instrumental", "launch-video", "electronic"}, Purposes: []string{"launch-video"}, BankPriority: 100, Approval: "liked", Feedback: "Operator liked this track."},
		{Sha256: "0913a02530c256570c42d0e7fba142d9e7bb953cdf83c680e135a38c68106dfc", Title: "e-fresh", SourcePath: filepath.Join(root, "e-fresh.wav"), TakeId: "8e60e63e-0973-4f47-9fc6-53101b461f36-take-01", ReceiptPath: filepath.Join(root, "e-receipt.json"), Tags: []string{"instrumental", "launch-video", "electronic"}, Purposes: []string{"launch-video"}, BankPriority: 100, Approval: "liked", Feedback: "Operator said this sounds just as great as the first four."},
		{Sha256: "593a8880a1bd3fcacfa924e0d48da0d5ea7805ebe8faca515977f5611500f819", Title: "f-fresh", SourcePath: filepath.Join(root, "f-fresh.wav"), TakeId: "e63381b1-0d4e-4e4e-a7d4-1967f7f1d678-take-01", ReceiptPath: filepath.Join(root, "f-receipt.json"), Tags: []string{"instrumental", "launch-video", "electronic"}, Purposes: []string{"launch-video"}, BankPriority: 100, Approval: "liked", Feedback: "Operator said this sounds just as great as the first four."},
	}
}

func buildMusicLibraryCLI(t *testing.T) string {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	cliDir := filepath.Clean(filepath.Join(filepath.Dir(testFile), "../../../cli"))
	binary := filepath.Join(t.TempDir(), "music-library")
	command := exec.Command("go", "build", "-o", binary, ".")
	command.Dir = cliDir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "build Music Library CLI: %s", output)
	return binary
}
