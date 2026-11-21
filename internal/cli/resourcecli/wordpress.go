package resourcecli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type WordPressCommandID string

var migrationSiteIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var wordpressTermNamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9 &-]{0,62}[A-Za-z0-9])?$`)

const (
	WordPressAcquire              WordPressCommandID = "acquire"
	WordPressStart                WordPressCommandID = "start"
	WordPressStop                 WordPressCommandID = "stop"
	WordPressReset                WordPressCommandID = "reset"
	WordPressStatus               WordPressCommandID = "status"
	WordPressSnapshot             WordPressCommandID = "snapshot"
	WordPressApplyRecipe          WordPressCommandID = "apply-recipe"
	WordPressInstallDestination   WordPressCommandID = "install-delivery-destination"
	WordPressVerifyDestination    WordPressCommandID = "verify-delivery-destination"
	WordPressVerifyContent        WordPressCommandID = "verify-content-ground"
	WordPressMigrateContent       WordPressCommandID = "migrate-content"
	WordPressPlanContentMigration WordPressCommandID = "plan-content-migration"
	WordPressExportContent        WordPressCommandID = "export-core-content"
	WordPressUpdateContentMetric  WordPressCommandID = "update-core-content-metric"
	WordPressSetContentTerm       WordPressCommandID = "set-core-content-term"
	WordPressAuditCoreContent     WordPressCommandID = "audit-core-content"
	WordPressVerifyContentAudit   WordPressCommandID = "verify-content-audit-ground"
	WordPressGradeGround          WordPressCommandID = "grade-ground"
	WordPressFormWebhook          WordPressCommandID = "form-webhook"
)

type WordPressRequest struct {
	SiteID         string
	SourceSiteID   string
	TargetSiteID   string
	ContentSlug    string
	ContentMetric  int
	RecipeID       string
	JSON           bool
	WebhookEnabled bool
	Taxonomy       string
	TermName       string
	TermPresent    bool
}

func WordPressCommandSpecs() []commandtree.Spec[WordPressCommandID] {
	site := commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--site-id", ValueName: "slug", Description: "Local isolated site identifier"}, commandtree.JSONOption()}}
	recipe := commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--site-id", ValueName: "slug", Description: "Local isolated site identifier"}, {Name: "--recipe-id", ValueName: "id", Description: "Versioned site recipe"}, commandtree.JSONOption()}}
	webhook := commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--site-id", ValueName: "slug", Description: "Local isolated site identifier"}, {Name: "--enabled", ValueName: "true|false", Description: "Enable the local n8n form webhook"}, commandtree.JSONOption()}}
	migration := commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--source-site-id", ValueName: "slug", Description: "Source owner-managed local WordPress site"}, {Name: "--target-site-id", ValueName: "slug", Description: "Destination owner-managed local WordPress site"}, commandtree.JSONOption()}}
	contentUpdate := commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--site-id", ValueName: "slug", Description: "Local isolated WordPress site identifier"}, {Name: "--slug", ValueName: "post-slug", Description: "Recipe-owned synthetic post slug"}, {Name: "--value", ValueName: "integer", Description: "New synthetic outcome metric"}, commandtree.JSONOption()}}
	contentTerm := commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--site-id", ValueName: "slug", Description: "Local isolated WordPress site identifier"}, {Name: "--slug", ValueName: "post-slug", Description: "Recipe-owned synthetic post slug"}, {Name: "--taxonomy", ValueName: "category|tag", Description: "WordPress core taxonomy"}, {Name: "--term", ValueName: "name", Description: "Safe bounded taxonomy term"}, {Name: "--present", ValueName: "true|false", Description: "Whether the post should have this term"}, commandtree.JSONOption()}}
	return []commandtree.Spec[WordPressCommandID]{
		{Name: string(WordPressAcquire), Summary: "Acquire only the digest-pinned WordPress, MariaDB and WP-CLI images", Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{commandtree.JSONOption()}}, Handler: WordPressAcquire},
		{Name: string(WordPressStart), Summary: "Start or create one isolated local WordPress site", Args: site, Handler: WordPressStart},
		{Name: string(WordPressStop), Summary: "Stop one owned WordPress site and database", Args: site, Handler: WordPressStop},
		{Name: string(WordPressReset), Summary: "Reset one owned WordPress site while retaining snapshots", Args: site, Handler: WordPressReset},
		{Name: string(WordPressStatus), Summary: "Observe one local WordPress site", Args: site, Handler: WordPressStatus},
		{Name: string(WordPressSnapshot), Summary: "Create one bounded database and content snapshot", Args: site, Handler: WordPressSnapshot},
		{Name: string(WordPressApplyRecipe), Summary: "Apply an approved staged recipe to one site", Args: recipe, Handler: WordPressApplyRecipe},
		{Name: string(WordPressInstallDestination), Summary: "Install the first-party local form and order idempotency destination", Args: site, Handler: WordPressInstallDestination},
		{Name: string(WordPressVerifyDestination), Summary: "Verify live form and order destination idempotency with synthetic data", Args: site, Handler: WordPressVerifyDestination},
		{Name: string(WordPressVerifyContent), Summary: "Verify real WordPress core custom-field behavior with a temporary synthetic post", Args: site, Handler: WordPressVerifyContent},
		{Name: string(WordPressMigrateContent), Summary: "Migrate synthetic recipe-owned content between two local WordPress sites", Args: migration, Handler: WordPressMigrateContent},
		{Name: string(WordPressPlanContentMigration), Summary: "Plan synthetic content changes between two local WordPress sites without writes", Args: migration, Handler: WordPressPlanContentMigration},
		{Name: string(WordPressExportContent), Summary: "Export the versioned synthetic core-content archive from one local site", Args: site, Handler: WordPressExportContent},
		{Name: string(WordPressUpdateContentMetric), Summary: "Update an outcome metric on one recipe-owned synthetic post", Args: contentUpdate, Handler: WordPressUpdateContentMetric},
		{Name: string(WordPressSetContentTerm), Summary: "Set one core taxonomy term on a recipe-owned synthetic post", Args: contentTerm, Handler: WordPressSetContentTerm},
		{Name: string(WordPressAuditCoreContent), Summary: "Audit recipe-owned synthetic content against the local editorial checklist", Args: site, Handler: WordPressAuditCoreContent},
		{Name: string(WordPressVerifyContentAudit), Summary: "Verify live content-audit findings with a temporary synthetic post", Args: site, Handler: WordPressVerifyContentAudit},
		{Name: string(WordPressGradeGround), Summary: "Execute all synthetic WordPress delivery ground cases against known answers", Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{commandtree.JSONOption()}}, Handler: WordPressGradeGround},
		{Name: string(WordPressFormWebhook), Summary: "Enable or disable the fixed local n8n form webhook target", Args: webhook, Handler: WordPressFormWebhook},
	}
}

func ParseWordPressRequest(action WordPressCommandID, args []string) (WordPressRequest, error) {
	var spec *commandtree.Spec[WordPressCommandID]
	for _, item := range WordPressCommandSpecs() {
		if item.Handler == action {
			copy := item
			spec = &copy
			break
		}
	}
	if spec == nil {
		return WordPressRequest{}, fmt.Errorf("unknown WordPress operation %q", action)
	}
	parsed, err := commandtree.ParseArgs("resource wordpress "+string(action), "", spec.Args, args)
	if err != nil {
		return WordPressRequest{}, err
	}
	req := WordPressRequest{SiteID: strings.TrimSpace(parsed.FlagValue("--site-id")), SourceSiteID: strings.TrimSpace(parsed.FlagValue("--source-site-id")), TargetSiteID: strings.TrimSpace(parsed.FlagValue("--target-site-id")), RecipeID: strings.TrimSpace(parsed.FlagValue("--recipe-id")), JSON: parsed.HasFlag("--json")}
	req.ContentSlug = strings.TrimSpace(parsed.FlagValue("--slug"))
	if action == WordPressMigrateContent || action == WordPressPlanContentMigration {
		if !migrationSiteIDPattern.MatchString(req.SourceSiteID) || !migrationSiteIDPattern.MatchString(req.TargetSiteID) {
			return WordPressRequest{}, fmt.Errorf("--source-site-id and --target-site-id must be safe site slugs")
		}
		if req.SourceSiteID == req.TargetSiteID {
			return WordPressRequest{}, fmt.Errorf("source and target site IDs must differ")
		}
	}
	if action == WordPressUpdateContentMetric {
		if req.ContentSlug == "" || !migrationSiteIDPattern.MatchString(req.ContentSlug) {
			return WordPressRequest{}, fmt.Errorf("--slug must be a safe post slug")
		}
		value, err := strconv.Atoi(strings.TrimSpace(parsed.FlagValue("--value")))
		if err != nil || value < 0 || value > 1_000_000_000 {
			return WordPressRequest{}, fmt.Errorf("--value must be an integer from 0 to 1000000000")
		}
		req.ContentMetric = value
	}
	if action == WordPressSetContentTerm {
		if req.ContentSlug == "" || !migrationSiteIDPattern.MatchString(req.ContentSlug) {
			return WordPressRequest{}, fmt.Errorf("--slug must be a safe post slug")
		}
		req.Taxonomy = strings.TrimSpace(parsed.FlagValue("--taxonomy"))
		if req.Taxonomy != "category" && req.Taxonomy != "tag" {
			return WordPressRequest{}, fmt.Errorf("--taxonomy must be category or tag")
		}
		req.TermName = strings.TrimSpace(parsed.FlagValue("--term"))
		if !wordpressTermNamePattern.MatchString(req.TermName) {
			return WordPressRequest{}, fmt.Errorf("--term must be a safe name of at most 64 characters")
		}
		switch strings.TrimSpace(parsed.FlagValue("--present")) {
		case "true":
			req.TermPresent = true
		case "false":
			req.TermPresent = false
		default:
			return WordPressRequest{}, fmt.Errorf("--present must be true or false")
		}
	}
	if action != WordPressAcquire && action != WordPressGradeGround && action != WordPressMigrateContent && action != WordPressPlanContentMigration && req.SiteID == "" {
		return WordPressRequest{}, fmt.Errorf("--site-id is required")
	}
	if action == WordPressApplyRecipe && req.RecipeID == "" {
		return WordPressRequest{}, fmt.Errorf("--recipe-id is required")
	}
	if action == WordPressFormWebhook {
		switch strings.TrimSpace(parsed.FlagValue("--enabled")) {
		case "true":
			req.WebhookEnabled = true
		case "false":
			req.WebhookEnabled = false
		default:
			return WordPressRequest{}, fmt.Errorf("--enabled must be true or false")
		}
	}
	return req, nil
}
