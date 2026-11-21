package files

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	"source-ledger/internal/files"

	filesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/files"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/shared"
)

type connectHandler struct{ service *files.Service }

func NewConnectHandler(service *files.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) Search(ctx context.Context, req *connect.Request[filesv1.SearchRequest]) (*connect.Response[filesv1.SearchResponse], error) {
	if strings.TrimSpace(req.Msg.GetQuery()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errQuery)
	}
	res, err := h.service.Search(ctx, files.Query{Text: req.Msg.GetQuery(), Corpus: req.Msg.GetCorpus(), Topic: req.Msg.GetTopic(), Feed: req.Msg.GetFeed(), Limit: int(req.Msg.GetLimit()), IncludeRaw: req.Msg.GetIncludeRaw()})
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	out := &filesv1.SearchResponse{Sync: syncProto(res.Sync), Mode: res.Mode, Source: "source-ledger"}
	for _, hit := range res.Hits {
		out.Hits = append(out.Hits, &filesv1.Hit{
			Citation: hit.Citation, Path: hit.Path, AbsPath: hit.AbsPath, Topic: hit.Topic, Feed: hit.Feed,
			LineStart: int32(hit.LineStart), LineEnd: int32(hit.LineEnd), Heading: hit.Heading, Snippet: hit.Snippet,
			Score: hit.Score, ModifiedAt: hit.ModifiedAt.Format(time.RFC3339), Truncated: hit.Truncated,
		})
	}
	out.Degradations = degradationsProto(res.Degradations)
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListFeeds(ctx context.Context, req *connect.Request[filesv1.ListFeedsRequest]) (*connect.Response[filesv1.ListFeedsResponse], error) {
	feeds, report, err := h.service.ListFeeds(ctx, req.Msg.GetCorpus(), req.Msg.GetTopic(), int(req.Msg.GetRecent()))
	if err != nil {
		return nil, connect.NewError(codeFor(err), err)
	}
	out := &filesv1.ListFeedsResponse{Sync: syncProto(report), Degradations: degradationsProto(files.ProblemsDegradations(report.Problems))}
	for _, f := range feeds {
		feed := &filesv1.Feed{Topic: f.Topic, Feed: f.Feed, Files: int32(f.Files), Bytes: f.Bytes, LastModified: f.LastModified.Format(time.RFC3339), Chunks: int32(f.Chunks), Embedded: int32(f.Embedded)}
		for _, r := range f.Recent {
			feed.Recent = append(feed.Recent, &filesv1.FeedItem{Path: r.Path, Heading: r.Title, ModifiedAt: r.ModifiedAt.Format(time.RFC3339)})
		}
		out.Feeds = append(out.Feeds, feed)
	}
	return connect.NewResponse(out), nil
}

func syncProto(r files.SyncReport) *filesv1.SyncReport {
	return &filesv1.SyncReport{
		FilesSeen: int32(r.Seen), FilesChanged: int32(r.Changed), FilesRemoved: int32(r.Removed), DurationMs: int32(r.Duration.Milliseconds()),
		SyncedAt: r.At.Format(time.RFC3339), Partial: r.Partial, Pending: int32(r.Pending),
		SkippedPrivate: int32(r.Stats.Private), SkippedExcluded: int32(r.Stats.Excluded), Summarised: int32(r.Stats.Summarised),
		SkippedOversizeJson: int32(r.Stats.OversizeJSON), Truncated: int32(r.Stats.Truncated), Problems: r.Problems,
	}
}

func degradationsProto(in []files.Degradation) []*sharedv1.Degradation {
	out := make([]*sharedv1.Degradation, 0, len(in))
	for _, d := range in {
		out = append(out, &sharedv1.Degradation{Reason: d.Reason, Leg: d.Leg, Detail: d.Detail, Retryable: true})
	}
	return out
}

var errQuery = errors.New("query is required")

// codeFor maps caller mistakes (unknown corpus, bad topic, empty query) to
// InvalidArgument; everything else is an internal failure.
func codeFor(err error) connect.Code {
	msg := err.Error()
	for _, caller := range []string{"unknown corpus", "invalid topic", "no searchable terms"} {
		if strings.Contains(msg, caller) {
			return connect.CodeInvalidArgument
		}
	}
	return connect.CodeInternal
}
