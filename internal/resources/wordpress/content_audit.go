package wordpress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var contentHTMLTagPattern = regexp.MustCompile(`<[^>]*>`)
var contentHeadingPattern = regexp.MustCompile(`(?is)<h[1-6](?:\s|>)`)

var contentAuditActions = map[string]string{
	"short_title":      "Write a descriptive title with at least three words.",
	"missing_excerpt":  "Add a concise editorial excerpt.",
	"short_excerpt":    "Expand the excerpt to at least six words.",
	"sparse_body":      "Add substantive body text with at least twenty words.",
	"missing_heading":  "Add a descriptive HTML heading to structure the body.",
	"missing_category": "Assign at least one appropriate WordPress category.",
	"missing_tag":      "Assign at least one useful WordPress tag.",
}

type ContentAuditFinding struct {
	Code   string `json:"code"`
	Action string `json:"action"`
}

type ContentAuditPost struct {
	Slug         string                `json:"slug"`
	WordCount    int                   `json:"wordCount"`
	HeadingCount int                   `json:"headingCount"`
	Score        int                   `json:"score"`
	Disposition  string                `json:"disposition"`
	Findings     []ContentAuditFinding `json:"findings"`
}

type ContentAuditReceipt struct {
	SiteID      string             `json:"siteId"`
	Contract    string             `json:"contract"`
	PostCount   int                `json:"postCount"`
	ReadyCount  int                `json:"readyCount"`
	ReviewCount int                `json:"reviewCount"`
	Posts       []ContentAuditPost `json:"posts"`
	Status      string             `json:"status"`
}

type ContentAuditGroundReceipt struct {
	SiteID        string   `json:"siteId"`
	Contract      string   `json:"contract"`
	TemporarySlug string   `json:"temporarySlug"`
	FindingCodes  []string `json:"findingCodes"`
	Disposition   string   `json:"disposition"`
	Cleanup       string   `json:"cleanup"`
	Status        string   `json:"status"`
}

// AuditCoreContent grades only content-core-blog posts marked synthetic. The
// rubric is a lightweight editorial preflight, not an SEO or accessibility claim.
func (o *Owner) AuditCoreContent(ctx context.Context, siteID string) (ContentAuditReceipt, error) {
	if err := validateSiteID(siteID); err != nil {
		return ContentAuditReceipt{}, err
	}
	container, secrets, err := o.requireRunningSite(ctx, siteID)
	if err != nil {
		return ContentAuditReceipt{}, err
	}
	archive, err := o.exportCoreContent(ctx, siteID, container, secrets.DatabasePassword)
	if err != nil {
		return ContentAuditReceipt{}, err
	}
	if len(archive.Records) == 0 {
		return ContentAuditReceipt{}, errors.New("content audit found no recipe-owned synthetic posts")
	}
	return AuditCoreContentRecords(siteID, archive.Records), nil
}

// VerifyCoreContentAuditGround inserts one temporary deficient synthetic post,
// checks that the live audit finds it, and removes it before returning.
func (o *Owner) VerifyCoreContentAuditGround(ctx context.Context, siteID string) (receipt ContentAuditGroundReceipt, returnErr error) {
	if err := validateSiteID(siteID); err != nil {
		return ContentAuditGroundReceipt{}, err
	}
	container, secrets, err := o.requireRunningSite(ctx, siteID)
	if err != nil {
		return ContentAuditGroundReceipt{}, err
	}
	code := "$slug='k3-audit-ground-'.strtolower(wp_generate_password(12,false,false));$id=wp_insert_post(['post_type'=>'post','post_status'=>'publish','post_title'=>'Tiny','post_content'=>'<p>Short synthetic content.</p>','post_excerpt'=>'','post_name'=>$slug],true);if(is_wp_error($id)){throw new RuntimeException('audit_ground_post_create_failed');}foreach(['_vrooli_recipe_id'=>'content-core-blog','client_type'=>'synthetic','outcome_metric'=>0,'verified_on'=>'2026-11-14'] as $key=>$value){update_post_meta((int)$id,$key,$value);}echo wp_json_encode(['id'=>(int)$id,'slug'=>$slug]);"
	args := o.wpCLIArgs(container, secrets.DatabasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		return ContentAuditGroundReceipt{}, fmt.Errorf("create temporary WordPress content-audit ground post: %w", err)
	}
	var created struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(output, &created); err != nil || created.ID <= 0 || !contentSlugPattern.MatchString(created.Slug) || !strings.HasPrefix(created.Slug, "k3-audit-ground-") {
		return ContentAuditGroundReceipt{}, fmt.Errorf("WordPress content-audit ground creation returned an invalid receipt: %s", strings.TrimSpace(string(output)))
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cleanupCode := fmt.Sprintf("$deleted=wp_delete_post(%s,true);echo wp_json_encode(['deleted'=>$deleted!==false]);", strconv.Itoa(created.ID))
		cleanupArgs := o.wpCLIArgs(container, secrets.DatabasePassword, true)
		cleanupArgs = append(cleanupArgs, "wp", "eval", cleanupCode, "--allow-root")
		cleanupOutput, cleanupErr := o.runner().Run(cleanupCtx, cleanupArgs...)
		if cleanupErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("delete temporary WordPress content-audit ground post: %w", cleanupErr))
			return
		}
		var deleted struct {
			Deleted bool `json:"deleted"`
		}
		if err := json.Unmarshal(cleanupOutput, &deleted); err != nil || !deleted.Deleted {
			returnErr = errors.Join(returnErr, fmt.Errorf("WordPress content-audit ground post %q was not confirmed deleted", created.Slug))
		}
	}()
	audit, err := o.AuditCoreContent(ctx, siteID)
	if err != nil {
		return ContentAuditGroundReceipt{}, err
	}
	var audited *ContentAuditPost
	for index := range audit.Posts {
		if audit.Posts[index].Slug == created.Slug {
			audited = &audit.Posts[index]
			break
		}
	}
	if audited == nil || audited.Disposition != "needs-review" {
		return ContentAuditGroundReceipt{}, errors.New("live content audit did not report the temporary synthetic post as needing review")
	}
	found := make(map[string]bool, len(audited.Findings))
	codes := make([]string, 0, len(audited.Findings))
	for _, finding := range audited.Findings {
		found[finding.Code] = true
		codes = append(codes, finding.Code)
	}
	for _, required := range []string{"short_title", "missing_excerpt", "sparse_body", "missing_heading", "missing_tag"} {
		if !found[required] {
			return ContentAuditGroundReceipt{}, fmt.Errorf("live content audit omitted expected finding %q", required)
		}
	}
	return ContentAuditGroundReceipt{SiteID: siteID, Contract: "vrooli.wordpress.content-audit-ground.v1", TemporarySlug: created.Slug, FindingCodes: codes, Disposition: audited.Disposition, Cleanup: "deleted-in-finally", Status: "verified"}, nil
}

// AuditCoreContentRecords applies the versioned editorial checklist to a
// supplied synthetic corpus so hard-ground fixtures and live runs share rules.
func AuditCoreContentRecords(siteID string, records []ContentMigrationRecord) ContentAuditReceipt {
	report := ContentAuditReceipt{SiteID: siteID, Contract: "vrooli.wordpress.content-audit.v1", PostCount: len(records), Posts: make([]ContentAuditPost, 0, len(records)), Status: "graded"}
	for _, record := range records {
		text := html.UnescapeString(contentHTMLTagPattern.ReplaceAllString(record.Content, " "))
		wordCount := len(strings.Fields(text))
		headingCount := len(contentHeadingPattern.FindAllString(record.Content, -1))
		post := ContentAuditPost{Slug: record.Slug, WordCount: wordCount, HeadingCount: headingCount, Score: 100, Disposition: "ready", Findings: []ContentAuditFinding{}}
		add := func(code string, penalty int) {
			post.Findings = append(post.Findings, ContentAuditFinding{Code: code, Action: contentAuditActions[code]})
			post.Score -= penalty
			post.Disposition = "needs-review"
		}
		if len(strings.Fields(record.Title)) < 3 {
			add("short_title", 10)
		}
		if strings.TrimSpace(record.Excerpt) == "" {
			add("missing_excerpt", 20)
		} else if len(strings.Fields(record.Excerpt)) < 6 {
			add("short_excerpt", 10)
		}
		if wordCount < 20 {
			add("sparse_body", 20)
		}
		if headingCount == 0 {
			add("missing_heading", 10)
		}
		if len(record.Categories) == 0 {
			add("missing_category", 15)
		}
		if len(record.Tags) == 0 {
			add("missing_tag", 10)
		}
		if post.Score < 0 {
			post.Score = 0
		}
		if post.Disposition == "ready" {
			report.ReadyCount++
		} else {
			report.ReviewCount++
		}
		report.Posts = append(report.Posts, post)
	}
	if report.PostCount == 0 {
		report.Status = "empty"
	}
	return report
}

// AuditSummary returns the compact known-answer projection used by hard ground.
func AuditSummary(report ContentAuditReceipt) map[string]any {
	findings := make([]string, 0)
	actions := make([]string, 0)
	score, wordCount, headingCount := 0, 0, 0
	if len(report.Posts) > 0 {
		score = report.Posts[0].Score
		wordCount = report.Posts[0].WordCount
		headingCount = report.Posts[0].HeadingCount
		for _, finding := range report.Posts[0].Findings {
			findings = append(findings, finding.Code)
			actions = append(actions, finding.Action)
		}
	}
	return map[string]any{"postCount": report.PostCount, "readyCount": report.ReadyCount, "reviewCount": report.ReviewCount, "firstDisposition": firstAuditDisposition(report), "firstScore": score, "firstWordCount": wordCount, "firstHeadingCount": headingCount, "firstFindings": findings, "firstActions": actions}
}

func firstAuditDisposition(report ContentAuditReceipt) string {
	if len(report.Posts) == 0 {
		return "empty"
	}
	return report.Posts[0].Disposition
}
