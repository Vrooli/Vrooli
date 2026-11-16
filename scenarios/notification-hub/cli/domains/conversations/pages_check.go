package conversations

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// pageCover is the decision a page belongs to, read from --cover-file: the
// fields of `ask` the page check compares the page with. Other keys are
// ignored, so a page writer's request file can be passed as it is.
type pageCover struct {
	Question             string `json:"question"`
	BigPicture           string `json:"big_picture"`
	TechnicalDetails     string `json:"technical_details"`
	Recommended          string `json:"recommended"`
	RecommendationReason string `json:"recommendation_reason"`
	Options              []struct {
		Key   string `json:"key"`
		Label string `json:"label"`
	} `json:"options"`
}

var pageTitlePattern = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// PageCheck is `pages check`: the dry run's packet check on one page and the
// decision it belongs to, without writing a whole ask. It reports exactly what
// `ask --dry-run` would, and exits 1 when the Hub would refuse the ask.
func PageCheck(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	var defaultTimeout time.Duration
	if core != nil && core.HTTPClient != nil {
		defaultTimeout = core.HTTPClient.Timeout()
	}
	h, base := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewConversationsServiceClient(hubHTTPClient(core, h, base, defaultTimeout), base)
	return func(ctx cliapp.RunContext) error { return checkPage(ctx, client) }
}

func checkPage(ctx cliapp.RunContext, client connectv1.ConversationsServiceClient) error {
	page := ctx.Positional("page")
	html, err := os.ReadFile(page)
	if err != nil {
		return fmt.Errorf("read the page: %w", unwrapPathError(err))
	}
	cover, hasCover, err := readPageCover(ctx.Flag("cover-file"), html)
	if err != nil {
		return err
	}
	pkt, questions := buildPacket(ctx.FlagValues("attach"), page, "", ctx.Flag("form"))
	for _, warning := range pkt.Warnings {
		fmt.Fprintf(ctx.Stderr(), "Warning: %s\n", warning)
	}
	host, _ := os.Hostname()
	req := &v1.AskRequest{
		Question:         cover.Question,
		SensitivityLabel: "private",
		IdempotencyKey:   "pages-check",
		Deadline:         time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		Questions:        questions,
	}
	for _, option := range cover.Options {
		req.Options = append(req.Options, &v1.AskOption{Key: option.Key, Label: option.Label})
	}
	if cover.Recommended != "" {
		req.Recommended = &cover.Recommended
	}
	if cover.RecommendationReason != "" {
		req.RecommendationReason = &cover.RecommendationReason
	}
	if cover.BigPicture != "" {
		req.BigPicture = &cover.BigPicture
	}
	if cover.TechnicalDetails != "" {
		req.TechnicalDetails = &cover.TechnicalDetails
	}
	for _, att := range pkt.attachments {
		req.Attachments = append(req.Attachments, att.proto(host))
	}
	built := askRequest{req: req, packet: pkt}
	resp, callErr := client.CheckAsk(context.Background(), connect.NewRequest(&v1.CheckAskRequest{Ask: req, FileContents: checkedContents(built)}))
	if callErr != nil {
		return askError(callErr, req)
	}
	warnings := resp.Msg.GetWarnings()
	if !hasCover {
		// Without the decision, only what the check says about the page itself
		// means anything: the cover here is a stand-in.
		warnings = pageOnly(warnings)
	}
	refused := hasBlocking(warnings)
	if ctx.JSON() {
		out := map[string]any{"passes": !refused, "checked_cover": hasCover, "warnings": warnings}
		encoder := json.NewEncoder(ctx.Stdout())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(out); err != nil {
			return err
		}
	} else {
		switch {
		case refused:
			fmt.Fprintln(ctx.Stdout(), "The Hub would refuse this decision. Fix each finding under \"Would refuse the ask\", then check again.")
		case len(warnings) == 0:
			fmt.Fprintln(ctx.Stdout(), "The page passes the Hub's check.")
		default:
			fmt.Fprintln(ctx.Stdout(), "The Hub would accept this decision; the advice below says what still reads poorly.")
		}
		if !hasCover {
			fmt.Fprintln(ctx.Stdout(), "Checked the page alone. Give --cover-file to check that it adds to the question and options, and the recommendation's reason.")
		}
		writePacketWarnings(ctx.Stdout(), warnings, false)
	}
	if refused {
		return &statusError{code: 1, msg: "the page check found what would refuse the ask; fix it and check again"}
	}
	return nil
}

// readPageCover reads --cover-file, or makes a stand-in cover from the page's
// title when none is given.
func readPageCover(path string, html []byte) (pageCover, bool, error) {
	if strings.TrimSpace(path) == "" {
		title := "Check this page"
		if match := pageTitlePattern.FindSubmatch(html); match != nil && strings.TrimSpace(string(match[1])) != "" {
			title = strings.TrimSpace(string(match[1]))
		}
		cover := pageCover{Question: title}
		cover.Options = append(cover.Options, struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		}{"yes", "Yes"}, struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		}{"no", "No"})
		return cover, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return pageCover{}, false, fmt.Errorf("--cover-file: %w", unwrapPathError(err))
	}
	var cover pageCover
	if err := json.Unmarshal(data, &cover); err != nil {
		return pageCover{}, false, fmt.Errorf("--cover-file %s must be JSON with question, options [{key, label}], recommended, recommendation_reason, big_picture: %w", filepath.Base(path), err)
	}
	if strings.TrimSpace(cover.Question) == "" || len(cover.Options) < 2 {
		return pageCover{}, false, fmt.Errorf("--cover-file %s needs a question and at least two options", filepath.Base(path))
	}
	return cover, true, nil
}

// pageOnly keeps the findings about the page and the files it names.
func pageOnly(warnings []*v1.PacketWarning) []*v1.PacketWarning {
	var out []*v1.PacketWarning
	for _, warning := range warnings {
		field := warning.GetField()
		if strings.HasPrefix(field, "attachments") || field == "explainer" || strings.HasPrefix(warning.GetCode(), "explainer_") || warning.GetCode() == "page_unreadable" {
			if warning.GetCode() == "explainer_restates" || warning.GetCode() == "explainer_missing" {
				continue
			}
			out = append(out, warning)
		}
	}
	return out
}
