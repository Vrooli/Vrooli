package conversations_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/proto"

	"notification-hub/internal/hub"
)

// No decision reaches the operator bare: an ask sent without a page is read
// with the Hub's page made from its cover, and a page the sender adds takes
// its place.
func TestAPageLessAskIsReadWithTheHubsPageUntilTheSenderSendsOne(t *testing.T) {
	ctx := context.Background()
	f := newAttachmentFixture(t)
	brief := f.put(t, "# Brief\nOn approve: run the migration.\n")
	asked, err := f.client.Ask(ctx, connect.NewRequest(&v1.AskRequest{
		SubjectKey: "migration", Question: "Run the retention migration tonight?",
		Options:     []*v1.AskOption{{Key: "run", Label: "Yes, run it tonight"}, {Key: "wait", Label: "No, wait a week"}},
		Recommended: proto.String("run"), RecommendationReason: proto.String("The disk fills in four days."),
		BigPicture:       proto.String("Old logs fill the disk; the migration trims them to 30 days."),
		Packet:           proto.String("/srv/work/RETENTION/BRIEF.md"),
		Deadline:         time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		SensitivityLabel: "private", IdempotencyKey: "cover-1",
		Attachments: []*v1.AskAttachment{{Name: "BRIEF.md", Kind: v1.AskAttachment_KIND_FILE, Sha256: brief, SourcePath: "/srv/work/RETENTION/BRIEF.md"}},
	}))
	require.NoError(t, err)
	var codes []string
	for _, warning := range asked.Msg.GetWarnings() {
		codes = append(codes, warning.GetCode())
	}
	require.Contains(t, codes, hub.WarnExplainerMissing, "the Hub's page is a floor; the sender is still told to make one")
	require.NotContains(t, codes, hub.WarnPacketMissing)

	got, err := f.client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: asked.Msg.GetAskId()}))
	require.NoError(t, err)
	require.Equal(t, "/srv/work/RETENTION/BRIEF.md", got.Msg.GetAsk().GetPacket(), "the asker reads its packet back")
	attachments := got.Msg.GetAsk().GetAttachments()
	require.Len(t, attachments, 2)
	require.Equal(t, hub.CoverPageName, attachments[0].GetName())
	require.Equal(t, v1.AskAttachment_ROLE_EXPLAINER, attachments[0].GetRole())

	resp := f.get(t, "/api/v1/asks/"+asked.Msg.GetAskId()+"/attachments/"+hub.CoverPageName)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	page := body(t, resp)
	for _, want := range []string{"Run the retention migration tonight?", "Old logs fill the disk", "Yes, run it tonight", "The disk fills in four days.", "What happens when you approve", `data-hub-open="BRIEF.md"`, "If you do not answer:"} {
		require.Contains(t, page, want)
	}
	require.NotContains(t, page, "/srv/work", "paths are for the executing agent, never the operator")
	require.Equal(t, `"sha256-`+attachments[0].GetSha256()+`"`, resp.Header.Get("ETag"))

	// A revision that changes the cover changes the page.
	_, err = f.client.Ask(ctx, connect.NewRequest(&v1.AskRequest{SubjectKey: "migration", BigPicture: proto.String("Logs now fill the disk in two days."), IdempotencyKey: "cover-2"}))
	require.NoError(t, err)
	require.Contains(t, body(t, f.get(t, "/api/v1/asks/"+asked.Msg.GetAskId()+"/attachments/"+hub.CoverPageName)), "two days")

	// The sender's own page replaces the Hub's.
	own := f.put(t, `<!doctype html><meta name="viewport" content="width=device-width, initial-scale=1"><h1>Retention</h1>`+
		`<table><tr><th>Log folder</th><th>Size</th><th>Oldest file</th></tr><tr><td>api requests</td><td>41 GB</td><td>March</td></tr>`+
		`<tr><td>worker traces</td><td>17 GB</td><td>June</td></tr><tr><td>billing exports</td><td>9 GB</td><td>January</td></tr></table>`+
		`<h2>Disk forecast</h2><p>Free space shrinks about 6 GB each day; without trimming, the volume reaches its limit Thursday morning and uploads start failing.</p><p>Keeping 30 days frees 52 GB tonight; the archive keeps a compressed monthly summary, `+
		`so audits still see totals. The migration pauses writes for under a minute and resumes on its own.</p>`)
	_, err = f.client.Ask(ctx, connect.NewRequest(&v1.AskRequest{
		SubjectKey: "migration", IdempotencyKey: "cover-3",
		Attachments: []*v1.AskAttachment{{Name: "page.html", Kind: v1.AskAttachment_KIND_FILE, Role: v1.AskAttachment_ROLE_EXPLAINER, Sha256: own}},
	}))
	require.NoError(t, err)
	got, err = f.client.GetAsk(ctx, connect.NewRequest(&v1.GetAskRequest{AskId: asked.Msg.GetAskId()}))
	require.NoError(t, err)
	var names []string
	for _, attachment := range got.Msg.GetAsk().GetAttachments() {
		names = append(names, attachment.GetName())
	}
	require.Equal(t, []string{"page.html", "BRIEF.md"}, names)
	require.Equal(t, http.StatusNotFound, f.get(t, "/api/v1/asks/"+asked.Msg.GetAskId()+"/attachments/"+hub.CoverPageName).StatusCode)
	require.Equal(t, "/srv/work/RETENTION/BRIEF.md", got.Msg.GetAsk().GetPacket(), "a revision that leaves the packet unsaid keeps it")
}

func TestAPacketMustBeAnAbsolutePath(t *testing.T) {
	f := newAttachmentFixture(t)
	_, err := f.client.Ask(context.Background(), connect.NewRequest(&v1.AskRequest{
		Question: "Approve?", AllowedAnswers: []string{"yes", "no"}, Packet: proto.String("work/X/BRIEF.md"),
		Deadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339), SensitivityLabel: "private", IdempotencyKey: "relative",
	}))
	require.Equal(t, connect.CodeInvalidArgument, codeOf(t, err))
	require.True(t, strings.Contains(err.Error(), "absolute"), err.Error())
}
