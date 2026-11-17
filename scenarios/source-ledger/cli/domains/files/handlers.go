package files

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	filesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/files"
	filesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/files/filesv1connect"
)

type handlers struct {
	client filesconnect.FilesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: filesconnect.NewFilesServiceClient(http, base)}
}

func intFlag(ctx cliapp.OperationContext, name string) int32 {
	value, err := strconv.ParseInt(ctx.Flag(name), 10, 32)
	if err != nil {
		return 0
	}
	return int32(value)
}

func (h *handlers) searchCall(ctx cliapp.OperationContext) (*filesv1.SearchResponse, error) {
	response, err := h.client.Search(context.Background(), connect.NewRequest(&filesv1.SearchRequest{
		Query: ctx.Positional("query"), Corpus: ctx.Flag("corpus"), Topic: ctx.Flag("topic"), Feed: ctx.Flag("feed"), Limit: intFlag(ctx, "limit"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("search research files", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) feedsCall(ctx cliapp.OperationContext) (*filesv1.ListFeedsResponse, error) {
	response, err := h.client.ListFeeds(context.Background(), connect.NewRequest(&filesv1.ListFeedsRequest{
		Corpus: ctx.Flag("corpus"), Topic: ctx.Flag("topic"), Recent: intFlag(ctx, "recent"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list research feeds", err, nil)
	}
	return response.Msg, nil
}

func syncLine(s *filesv1.SyncReport) string {
	line := fmt.Sprintf("Synced on read: %d files seen, %d changed, %d removed in %dms.", s.GetFilesSeen(), s.GetFilesChanged(), s.GetFilesRemoved(), s.GetDurationMs())
	if s.GetPartial() {
		line += fmt.Sprintf(" %d changed files were not reached and are excluded.", s.GetPending())
	}
	return line
}

// coverageLine says what the indexing policy left out or cut short.
func coverageLine(s *filesv1.SyncReport) string {
	return fmt.Sprintf("Not indexed: %d private, %d excluded, %d inside summarised folders, %d oversize JSON; %d indexed files are cut short.",
		s.GetSkippedPrivate(), s.GetSkippedExcluded(), s.GetSummarised(), s.GetSkippedOversizeJson(), s.GetTruncated())
}

func (h *handlers) searchReport(_ cliapp.OperationContext, msg *filesv1.SearchResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetHits()))
	for _, hit := range msg.GetHits() {
		heading := ""
		if hit.GetHeading() != "" {
			heading = " — " + hit.GetHeading()
		}
		cut := ""
		if hit.GetTruncated() {
			cut = " [file cut short in the index]"
		}
		results = append(results, fmt.Sprintf("%s%s (score %.3f)%s\n    %s", hit.GetCitation(), heading, hit.GetScore(), cut, strings.ReplaceAll(hit.GetSnippet(), "\n", " ")))
	}
	summary := []string{fmt.Sprintf("Found %d cited hit(s); ranking %s.", len(results), msg.GetMode()), syncLine(msg.GetSync()), coverageLine(msg.GetSync())}
	for _, d := range msg.GetDegradations() {
		summary = append(summary, "Degraded "+d.GetLeg()+": "+d.GetDetail())
	}
	return cliapp.ListReport{
		Summary: summary, ResultsHeading: "Hits (path#Lstart-Lend under the research home)", Results: results,
		RetrievalHints: []string{"Read a hit at its lines: sed -n '<start>,<end>p' <abs path>", "If Source Ledger is down, the research skills' find.py gives the same shape from files"},
		ListShaped:     true, ResultCount: len(results),
	}
}

func (h *handlers) feedsReport(_ cliapp.OperationContext, msg *filesv1.ListFeedsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetFeeds()))
	for _, f := range msg.GetFeeds() {
		topic := f.GetTopic()
		if topic == "" {
			topic = "(home)"
		}
		line := fmt.Sprintf("%s/%s: %d files, %d KB, %d/%d chunks embedded, last change %s", topic, f.GetFeed(), f.GetFiles(), f.GetBytes()/1024, f.GetEmbedded(), f.GetChunks(), f.GetLastModified())
		for _, r := range f.GetRecent() {
			line += fmt.Sprintf("\n    %s  %s (%s)", r.GetModifiedAt(), r.GetPath(), r.GetHeading())
		}
		results = append(results, line)
	}
	summary := []string{fmt.Sprintf("%d feed(s).", len(results)), syncLine(msg.GetSync()), coverageLine(msg.GetSync())}
	for _, d := range msg.GetDegradations() {
		summary = append(summary, "Degraded "+d.GetLeg()+": "+d.GetDetail())
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Feeds", Results: results, ListShaped: true, ResultCount: len(results)}
}
