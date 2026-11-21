package wordpress

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var contentMigrationSiteIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var ErrContentSlugConflict = errors.New("content migration refused a destination slug not owned by this source")

// ContentMigrationRecord is deliberately limited to the synthetic core-content
// recipe. Arbitrary site content and client data are outside this research path.
type ContentMigrationRecord struct {
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Excerpt       string   `json:"excerpt"`
	Categories    []string `json:"categories"`
	Tags          []string `json:"tags"`
	Status        string   `json:"status"`
	ClientType    string   `json:"client_type"`
	OutcomeMetric int      `json:"outcome_metric"`
	VerifiedOn    string   `json:"verified_on"`
	SourceKey     string   `json:"sourceKey"`
	ContentHash   string   `json:"contentHash"`
}

type ContentMetricUpdateReceipt struct {
	SiteID string `json:"siteId"`
	Slug   string `json:"slug"`
	Before int    `json:"before"`
	After  int    `json:"after"`
	Status string `json:"status"`
}

type ContentTermUpdateReceipt struct {
	SiteID   string   `json:"siteId"`
	Slug     string   `json:"slug"`
	Taxonomy string   `json:"taxonomy"`
	Term     string   `json:"term"`
	Present  bool     `json:"present"`
	Before   []string `json:"before"`
	After    []string `json:"after"`
	Status   string   `json:"status"`
}

// UpdateCoreContentMetric edits only an existing synthetic post owned by the
// core blog recipe. This provides a local client-style content change that can
// then be transferred through MigrateCoreContent.
func (o *Owner) UpdateCoreContentMetric(ctx context.Context, siteID, slug string, value int) (ContentMetricUpdateReceipt, error) {
	if err := validateSiteID(siteID); err != nil {
		return ContentMetricUpdateReceipt{}, err
	}
	if !contentSlugPattern.MatchString(slug) || value < 0 || value > 1_000_000_000 {
		return ContentMetricUpdateReceipt{}, errors.New("content metric update requires a safe slug and bounded nonnegative value")
	}
	container, secrets, err := o.requireRunningSite(ctx, siteID)
	if err != nil {
		return ContentMetricUpdateReceipt{}, err
	}
	code := fmt.Sprintf("$post=get_page_by_path('%s',OBJECT,'post');if(!$post){throw new RuntimeException('post_not_found');}$id=(int)$post->ID;if(get_post_meta($id,'_vrooli_recipe_id',true)!=='content-core-blog'||get_post_meta($id,'client_type',true)!=='synthetic'){throw new RuntimeException('post_not_owned_or_synthetic');}$before=(int)get_post_meta($id,'outcome_metric',true);update_post_meta($id,'outcome_metric',%d);$after=(int)get_post_meta($id,'outcome_metric',true);if($after!==%d){throw new RuntimeException('metric_update_failed');}echo wp_json_encode(['slug'=>get_post_field('post_name',$id),'before'=>$before,'after'=>$after]);", slug, value, value)
	args := o.wpCLIArgs(container, secrets.DatabasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		return ContentMetricUpdateReceipt{}, fmt.Errorf("update synthetic WordPress content metric: %w", err)
	}
	var result struct {
		Slug   string `json:"slug"`
		Before int    `json:"before"`
		After  int    `json:"after"`
	}
	if err := json.Unmarshal(output, &result); err != nil || result.Slug != slug || result.After != value {
		return ContentMetricUpdateReceipt{}, fmt.Errorf("WordPress content metric update returned an invalid receipt: %s", strings.TrimSpace(string(output)))
	}
	return ContentMetricUpdateReceipt{SiteID: siteID, Slug: slug, Before: result.Before, After: result.After, Status: "verified"}, nil
}

// SetCoreContentTerm changes one core category/tag on a synthetic post owned
// by the blog recipe. It provides a bounded source edit for proving migration
// planning and reconciliation without editing a site's database directly.
func (o *Owner) SetCoreContentTerm(ctx context.Context, siteID, slug, taxonomy, termName string, present bool) (ContentTermUpdateReceipt, error) {
	if err := validateSiteID(siteID); err != nil {
		return ContentTermUpdateReceipt{}, err
	}
	if !contentSlugPattern.MatchString(slug) || (taxonomy != "category" && taxonomy != "tag") || !contentTermNamePattern.MatchString(termName) {
		return ContentTermUpdateReceipt{}, errors.New("content term update requires a safe slug, category or tag, and bounded term name")
	}
	container, secrets, err := o.requireRunningSite(ctx, siteID)
	if err != nil {
		return ContentTermUpdateReceipt{}, err
	}
	payload, err := json.Marshal(map[string]any{"slug": slug, "taxonomy": taxonomy, "term": termName, "present": present})
	if err != nil {
		return ContentTermUpdateReceipt{}, err
	}
	code := fmt.Sprintf("$input=json_decode(base64_decode('%s'),true,512,JSON_THROW_ON_ERROR);$post=get_page_by_path($input['slug'],OBJECT,'post');if(!$post){throw new RuntimeException('post_not_found');}$id=(int)$post->ID;if(get_post_meta($id,'_vrooli_recipe_id',true)!=='content-core-blog'||get_post_meta($id,'client_type',true)!=='synthetic'){throw new RuntimeException('post_not_owned_or_synthetic');}$taxonomy=$input['taxonomy']==='category'?'category':'post_tag';$before=wp_get_post_terms($id,$taxonomy,['fields'=>'names']);$term=term_exists($input['term'],$taxonomy);if($input['present']&&!$term){$term=wp_insert_term($input['term'],$taxonomy);if(is_wp_error($term)){throw new RuntimeException('term_create_failed');}}$names=$before;if($input['present']&&!in_array($input['term'],$names,true)){$names[]=$input['term'];}if(!$input['present']){$names=array_values(array_filter($names,function($name)use($input){return $name!==$input['term'];}));if($taxonomy==='category'&&count($names)===0){throw new RuntimeException('cannot_remove_last_category');}}$ids=[];foreach($names as $name){$found=term_exists($name,$taxonomy);if(!$found){$found=wp_insert_term($name,$taxonomy);}if(is_wp_error($found)){throw new RuntimeException('term_resolution_failed');}$ids[]=(int)(is_array($found)?$found['term_id']:$found);} $set=wp_set_post_terms($id,$ids,$taxonomy,false);if(is_wp_error($set)){throw new RuntimeException('term_assignment_failed');}$after=wp_get_post_terms($id,$taxonomy,['fields'=>'names']);sort($before);sort($after);$want=$names;sort($want);if($after!==$want){throw new RuntimeException('term_update_verify_failed');}echo wp_json_encode(['slug'=>$input['slug'],'before'=>$before,'after'=>$after]);", base64.StdEncoding.EncodeToString(payload))
	args := o.wpCLIArgs(container, secrets.DatabasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		return ContentTermUpdateReceipt{}, fmt.Errorf("update synthetic WordPress content taxonomy: %w", err)
	}
	var result struct {
		Slug   string   `json:"slug"`
		Before []string `json:"before"`
		After  []string `json:"after"`
	}
	if err := json.Unmarshal(output, &result); err != nil || result.Slug != slug {
		return ContentTermUpdateReceipt{}, fmt.Errorf("WordPress content taxonomy update returned an invalid receipt: %s", strings.TrimSpace(string(output)))
	}
	return ContentTermUpdateReceipt{SiteID: siteID, Slug: slug, Taxonomy: taxonomy, Term: termName, Present: present, Before: result.Before, After: result.After, Status: "verified"}, nil
}

type ContentMigrationReceipt struct {
	SourceSiteID  string                         `json:"sourceSiteId"`
	TargetSiteID  string                         `json:"targetSiteId"`
	Contract      string                         `json:"contract"`
	RecordCount   int                            `json:"recordCount"`
	Created       int                            `json:"created"`
	Updated       int                            `json:"updated"`
	Duplicate     int                            `json:"duplicate"`
	ArchiveSHA256 string                         `json:"archiveSha256"`
	VerifiedPosts []ContentMigrationVerifiedPost `json:"verifiedPosts"`
	Status        string                         `json:"status"`
}

type ContentMigrationVerifiedPost struct {
	Slug       string   `json:"slug"`
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
}

type ContentMigrationPlanEntry struct {
	Slug        string `json:"slug"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason,omitempty"`
}

type ContentMigrationPlanReceipt struct {
	SourceSiteID  string                      `json:"sourceSiteId"`
	TargetSiteID  string                      `json:"targetSiteId"`
	Contract      string                      `json:"contract"`
	RecordCount   int                         `json:"recordCount"`
	Create        int                         `json:"create"`
	Update        int                         `json:"update"`
	Duplicate     int                         `json:"duplicate"`
	Conflict      int                         `json:"conflict"`
	ArchiveSHA256 string                      `json:"archiveSha256"`
	Entries       []ContentMigrationPlanEntry `json:"entries"`
	Status        string                      `json:"status"`
}

// ContentMigrationArchive is the versioned, synthetic-only portable export
// produced by ExportCoreContentArchive and consumed by the migration workflow.
type ContentMigrationArchive struct {
	SchemaVersion int                      `json:"schemaVersion"`
	SourceSiteID  string                   `json:"sourceSiteId"`
	Records       []ContentMigrationRecord `json:"records"`
}

// ExportCoreContentArchive returns the exact migration contract for the
// synthetic posts owned by the core blog recipe. It never includes arbitrary
// site content and performs no writes.
func (o *Owner) ExportCoreContentArchive(ctx context.Context, siteID string) (ContentMigrationArchive, error) {
	if err := validateSiteID(siteID); err != nil {
		return ContentMigrationArchive{}, err
	}
	container, secrets, err := o.requireRunningSite(ctx, siteID)
	if err != nil {
		return ContentMigrationArchive{}, err
	}
	archive, err := o.exportCoreContent(ctx, siteID, container, secrets.DatabasePassword)
	if err != nil {
		return ContentMigrationArchive{}, err
	}
	if len(archive.Records) == 0 {
		return ContentMigrationArchive{}, errors.New("content export has no recipe-owned synthetic posts")
	}
	return archive, nil
}

// PlanCoreContentMigration compares the validated synthetic source archive
// with the destination without writing either WordPress site.
func (o *Owner) PlanCoreContentMigration(ctx context.Context, sourceSiteID, targetSiteID string) (ContentMigrationPlanReceipt, error) {
	if err := validateSiteID(sourceSiteID); err != nil {
		return ContentMigrationPlanReceipt{}, fmt.Errorf("source site: %w", err)
	}
	if err := validateSiteID(targetSiteID); err != nil {
		return ContentMigrationPlanReceipt{}, fmt.Errorf("target site: %w", err)
	}
	if sourceSiteID == targetSiteID {
		return ContentMigrationPlanReceipt{}, errors.New("content migration source and target must be different sites")
	}
	sourceContainer, sourceSecrets, err := o.requireRunningSite(ctx, sourceSiteID)
	if err != nil {
		return ContentMigrationPlanReceipt{}, fmt.Errorf("source site: %w", err)
	}
	targetContainer, targetSecrets, err := o.requireRunningSite(ctx, targetSiteID)
	if err != nil {
		return ContentMigrationPlanReceipt{}, fmt.Errorf("target site: %w", err)
	}
	archive, err := o.exportCoreContent(ctx, sourceSiteID, sourceContainer, sourceSecrets.DatabasePassword)
	if err != nil {
		return ContentMigrationPlanReceipt{}, err
	}
	if len(archive.Records) == 0 {
		return ContentMigrationPlanReceipt{}, errors.New("content migration source has no recipe-owned synthetic posts")
	}
	payload, err := json.Marshal(archive)
	if err != nil {
		return ContentMigrationPlanReceipt{}, err
	}
	archiveDigest := sha256.Sum256(payload)
	code := fmt.Sprintf("$archive=json_decode(base64_decode('%s'),true,512,JSON_THROW_ON_ERROR);$entries=[];foreach($archive['records'] as $row){$post=get_page_by_path($row['slug'],OBJECT,'post');if(!$post){$entries[]=['slug'=>$row['slug'],'disposition'=>'create','reason'=>'slug-available'];continue;}$id=(int)$post->ID;$source=get_post_meta($id,'_vrooli_migration_source',true);if($source!==$row['sourceKey']){$entries[]=['slug'=>$row['slug'],'disposition'=>'conflict','reason'=>'destination-slug-owned-by-another-recipe-or-source'];continue;}$hash=get_post_meta($id,'_vrooli_migration_hash',true);$cats=wp_get_post_terms($id,'category',['fields'=>'names']);$tags=wp_get_post_terms($id,'post_tag',['fields'=>'names']);sort($cats);sort($tags);$wantCats=$row['categories'];$wantTags=$row['tags'];sort($wantCats);sort($wantTags);if($hash===$row['contentHash']&&$cats===$wantCats&&$tags===$wantTags){$entries[]=['slug'=>$row['slug'],'disposition'=>'duplicate','reason'=>'content-hash-and-taxonomy-unchanged'];continue;}$entries[]=['slug'=>$row['slug'],'disposition'=>'update','reason'=>'content-or-taxonomy-changed'];}echo wp_json_encode(['entries'=>$entries]);", base64.StdEncoding.EncodeToString(payload))
	args := o.wpCLIArgs(targetContainer, targetSecrets.DatabasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		return ContentMigrationPlanReceipt{}, fmt.Errorf("plan synthetic WordPress content migration: %w", err)
	}
	var result struct {
		Entries []ContentMigrationPlanEntry `json:"entries"`
	}
	if err := json.Unmarshal(output, &result); err != nil || len(result.Entries) != len(archive.Records) {
		return ContentMigrationPlanReceipt{}, fmt.Errorf("WordPress content migration plan returned an invalid receipt: %s", strings.TrimSpace(string(output)))
	}
	plan := ContentMigrationPlanReceipt{SourceSiteID: sourceSiteID, TargetSiteID: targetSiteID, Contract: "vrooli.wordpress.content-migration.v1", RecordCount: len(archive.Records), ArchiveSHA256: hex.EncodeToString(archiveDigest[:]), Entries: result.Entries, Status: "planned-read-only"}
	for index, entry := range result.Entries {
		if entry.Slug != archive.Records[index].Slug {
			return ContentMigrationPlanReceipt{}, errors.New("WordPress content migration plan order does not match the source archive")
		}
		switch entry.Disposition {
		case "create":
			plan.Create++
		case "update":
			plan.Update++
		case "duplicate":
			plan.Duplicate++
		case "conflict":
			plan.Conflict++
		default:
			return ContentMigrationPlanReceipt{}, fmt.Errorf("WordPress content migration plan returned unknown disposition %q", entry.Disposition)
		}
	}
	return plan, nil
}

// MigrateCoreContent transfers only recipe-owned synthetic WordPress posts
// between two running owner-controlled local sites. The destination is
// preflighted for slug ownership before any write; retries are keyed by source
// site and slug, and content hashes distinguish duplicate replay from update.
func (o *Owner) MigrateCoreContent(ctx context.Context, sourceSiteID, targetSiteID string) (ContentMigrationReceipt, error) {
	if err := validateSiteID(sourceSiteID); err != nil {
		return ContentMigrationReceipt{}, fmt.Errorf("source site: %w", err)
	}
	if err := validateSiteID(targetSiteID); err != nil {
		return ContentMigrationReceipt{}, fmt.Errorf("target site: %w", err)
	}
	if sourceSiteID == targetSiteID {
		return ContentMigrationReceipt{}, errors.New("content migration source and target must be different sites")
	}
	sourceContainer, sourceSecrets, err := o.requireRunningSite(ctx, sourceSiteID)
	if err != nil {
		return ContentMigrationReceipt{}, fmt.Errorf("source site: %w", err)
	}
	targetContainer, targetSecrets, err := o.requireRunningSite(ctx, targetSiteID)
	if err != nil {
		return ContentMigrationReceipt{}, fmt.Errorf("target site: %w", err)
	}

	archive, err := o.exportCoreContent(ctx, sourceSiteID, sourceContainer, sourceSecrets.DatabasePassword)
	if err != nil {
		return ContentMigrationReceipt{}, err
	}
	if len(archive.Records) == 0 {
		return ContentMigrationReceipt{}, errors.New("content migration source has no recipe-owned synthetic posts")
	}
	payload, err := json.Marshal(archive)
	if err != nil {
		return ContentMigrationReceipt{}, err
	}
	archiveHash := sha256.Sum256(payload)
	encoded := base64.StdEncoding.EncodeToString(payload)
	code := fmt.Sprintf("$archive=json_decode(base64_decode('%s'),true,512,JSON_THROW_ON_ERROR);$rows=$archive['records'];foreach($rows as $row){$post=get_page_by_path($row['slug'],OBJECT,'post');if($post&&get_post_meta((int)$post->ID,'_vrooli_migration_source',true)!==$row['sourceKey']){throw new RuntimeException('destination_slug_conflict');}}$created=0;$updated=0;$duplicate=0;$verified=[];foreach($rows as $row){$content=wp_kses_post($row['content']);$excerpt=sanitize_text_field($row['excerpt']);$post=get_page_by_path($row['slug'],OBJECT,'post');if($post){$id=(int)$post->ID;$oldHash=get_post_meta($id,'_vrooli_migration_hash',true);if($oldHash===$row['contentHash']){$cats=wp_get_post_terms($id,'category',['fields'=>'names']);$tags=wp_get_post_terms($id,'post_tag',['fields'=>'names']);sort($cats);sort($tags);$wantCats=$row['categories'];$wantTags=$row['tags'];sort($wantCats);sort($wantTags);if($cats===$wantCats&&$tags===$wantTags){$duplicate++;$verified[]=['slug'=>get_post_field('post_name',$id),'contentHash'=>$oldHash,'categories'=>$cats,'tags'=>$tags];continue;}}$write=wp_update_post(['ID'=>$id,'post_title'=>$row['title'],'post_content'=>$content,'post_excerpt'=>$excerpt,'post_status'=>$row['status']],true);if(is_wp_error($write)||(int)$write!==$id){throw new RuntimeException('destination_post_update_failed');}$updated++;}else{$id=wp_insert_post(['post_type'=>'post','post_status'=>$row['status'],'post_title'=>$row['title'],'post_content'=>$content,'post_excerpt'=>$excerpt,'post_name'=>$row['slug']],true);if(is_wp_error($id)){throw new RuntimeException('destination_post_create_failed');}$created++;}foreach(['client_type','outcome_metric','verified_on'] as $field){update_post_meta($id,$field,$row[$field]);}foreach(['category'=>'categories','post_tag'=>'tags'] as $taxonomy=>$field){$termIds=[];foreach(($row[$field]??[]) as $name){$term=term_exists($name,$taxonomy);if(!$term){$term=wp_insert_term($name,$taxonomy);}if(is_wp_error($term)){throw new RuntimeException('destination_term_failed');}$termIds[]=(int)(is_array($term)?$term['term_id']:$term);} $set=wp_set_post_terms($id,$termIds,$taxonomy,false);if(is_wp_error($set)){throw new RuntimeException('destination_term_assign_failed');}}update_post_meta($id,'_vrooli_migration_source',$row['sourceKey']);update_post_meta($id,'_vrooli_migration_hash',$row['contentHash']);$actualCats=wp_get_post_terms($id,'category',['fields'=>'names']);$actualTags=wp_get_post_terms($id,'post_tag',['fields'=>'names']);sort($actualCats);sort($actualTags);$wantCats=$row['categories'];$wantTags=$row['tags'];sort($wantCats);sort($wantTags);$actual=['slug'=>get_post_field('post_name',$id),'title'=>get_post_field('post_title',$id),'content'=>get_post_field('post_content',$id),'excerpt'=>get_post_field('post_excerpt',$id),'status'=>get_post_field('post_status',$id),'client_type'=>get_post_meta($id,'client_type',true),'outcome_metric'=>(int)get_post_meta($id,'outcome_metric',true),'verified_on'=>get_post_meta($id,'verified_on',true),'contentHash'=>get_post_meta($id,'_vrooli_migration_hash',true)];if($actual['slug']!==$row['slug']||$actual['title']!==$row['title']||$actual['content']!==$content||$actual['excerpt']!==$excerpt||$actual['status']!==$row['status']||$actual['client_type']!==$row['client_type']||$actual['outcome_metric']!==(int)$row['outcome_metric']||$actual['verified_on']!==$row['verified_on']||$actual['contentHash']!==$row['contentHash']||$actualCats!==$wantCats||$actualTags!==$wantTags){throw new RuntimeException('destination_content_verify_failed');}$verified[]=['slug'=>$actual['slug'],'contentHash'=>$actual['contentHash'],'categories'=>$actualCats,'tags'=>$actualTags];}echo wp_json_encode(['created'=>$created,'updated'=>$updated,'duplicate'=>$duplicate,'verified'=>$verified]);", encoded)
	args := o.wpCLIArgs(targetContainer, targetSecrets.DatabasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		if strings.Contains(err.Error(), "destination_slug_conflict") {
			return ContentMigrationReceipt{}, ErrContentSlugConflict
		}
		return ContentMigrationReceipt{}, fmt.Errorf("apply synthetic WordPress content migration: %w", err)
	}
	var result struct {
		Created   int `json:"created"`
		Updated   int `json:"updated"`
		Duplicate int `json:"duplicate"`
		Verified  []struct {
			Slug        string   `json:"slug"`
			ContentHash string   `json:"contentHash"`
			Categories  []string `json:"categories"`
			Tags        []string `json:"tags"`
		} `json:"verified"`
	}
	if err := json.Unmarshal(output, &result); err != nil || len(result.Verified) != len(archive.Records) || result.Created+result.Updated+result.Duplicate != len(archive.Records) {
		return ContentMigrationReceipt{}, fmt.Errorf("WordPress content migration returned an invalid receipt: %s", strings.TrimSpace(string(output)))
	}
	verifiedPosts := make([]ContentMigrationVerifiedPost, 0, len(archive.Records))
	for index, record := range archive.Records {
		if result.Verified[index].Slug != record.Slug || result.Verified[index].ContentHash != record.ContentHash || !sameStrings(result.Verified[index].Categories, record.Categories) || !sameStrings(result.Verified[index].Tags, record.Tags) {
			return ContentMigrationReceipt{}, errors.New("WordPress content migration verification did not match the source archive")
		}
		verifiedPosts = append(verifiedPosts, ContentMigrationVerifiedPost{Slug: record.Slug, Categories: result.Verified[index].Categories, Tags: result.Verified[index].Tags})
	}
	return ContentMigrationReceipt{SourceSiteID: sourceSiteID, TargetSiteID: targetSiteID, Contract: "vrooli.wordpress.content-migration.v1", RecordCount: len(archive.Records), Created: result.Created, Updated: result.Updated, Duplicate: result.Duplicate, ArchiveSHA256: hex.EncodeToString(archiveHash[:]), VerifiedPosts: verifiedPosts, Status: "verified"}, nil
}

func (o *Owner) requireRunningSite(ctx context.Context, siteID string) (string, instanceSecrets, error) {
	container := siteContainerPrefix + siteID
	info, exists, err := o.inspectContainer(ctx, container)
	if err != nil {
		return "", instanceSecrets{}, err
	}
	if !exists || !ownsContainer(info, siteID, "site") || info.State.Status != "running" {
		return "", instanceSecrets{}, errors.New("matching owned WordPress site must be running")
	}
	paths, err := o.instancePaths(siteID, false)
	if err != nil {
		return "", instanceSecrets{}, err
	}
	secrets, err := o.ensureSecrets(paths)
	if err != nil {
		return "", instanceSecrets{}, err
	}
	return container, secrets, nil
}

func (o *Owner) exportCoreContent(ctx context.Context, siteID, container, databasePassword string) (ContentMigrationArchive, error) {
	code := fmt.Sprintf("$posts=get_posts(['post_type'=>'post','post_status'=>['publish','draft'],'posts_per_page'=>1000,'meta_key'=>'_vrooli_recipe_id','meta_value'=>'content-core-blog','orderby'=>'ID','order'=>'ASC']);$records=[];foreach($posts as $post){$id=(int)$post->ID;$client=get_post_meta($id,'client_type',true);if($client!=='synthetic'){continue;}$categories=wp_get_post_terms($id,'category',['fields'=>'names']);$tags=wp_get_post_terms($id,'post_tag',['fields'=>'names']);sort($categories);sort($tags);$records[]=['slug'=>get_post_field('post_name',$id),'title'=>get_post_field('post_title',$id),'content'=>get_post_field('post_content',$id),'excerpt'=>get_post_field('post_excerpt',$id),'categories'=>$categories,'tags'=>$tags,'status'=>get_post_field('post_status',$id),'client_type'=>$client,'outcome_metric'=>(int)get_post_meta($id,'outcome_metric',true),'verified_on'=>get_post_meta($id,'verified_on',true)];}echo wp_json_encode(['schemaVersion'=>1,'sourceSiteId'=>'%s','records'=>$records]);", siteID)
	args := o.wpCLIArgs(container, databasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		return ContentMigrationArchive{}, fmt.Errorf("export synthetic WordPress core content: %w", err)
	}
	var archive ContentMigrationArchive
	if err := json.Unmarshal(output, &archive); err != nil || archive.SchemaVersion != 1 || archive.SourceSiteID != siteID || len(archive.Records) > 1000 {
		return ContentMigrationArchive{}, fmt.Errorf("decode WordPress content export: %s", strings.TrimSpace(string(output)))
	}
	seen := make(map[string]bool, len(archive.Records))
	for index := range archive.Records {
		record := &archive.Records[index]
		post := map[string]any{"slug": record.Slug, "title": record.Title, "content": record.Content, "excerpt": record.Excerpt, "categories": record.Categories, "tags": record.Tags, "status": record.Status, "client_type": record.ClientType, "outcome_metric": float64(record.OutcomeMetric), "verified_on": record.VerifiedOn}
		if err := validateCoreContentPost(post); err != nil || seen[record.Slug] {
			return ContentMigrationArchive{}, fmt.Errorf("unsafe or duplicate WordPress content export record %q", record.Slug)
		}
		seen[record.Slug] = true
		record.SourceKey = siteID + "/" + record.Slug
		record.ContentHash, err = contentMigrationHash(*record)
		if err != nil {
			return ContentMigrationArchive{}, err
		}
	}
	return archive, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}
