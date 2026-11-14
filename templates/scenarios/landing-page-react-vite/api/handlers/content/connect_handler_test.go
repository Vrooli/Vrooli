package content_test

import (
	"context"
	"database/sql"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	contentH "landing-page-react-vite-api/handlers/content"
	internalcontent "landing-page-react-vite-api/internal/content"

	internalvariant "landing-page-react-vite-api/internal/variant"
)

func setup(t *testing.T) (*contentH.Deps, int64) {
	t.Helper()
	db := pgtest.NewDB(t)
	// content_sections FKs variants(id): apply the variant schema first.
	pgtest.Apply(t, db, internalvariant.Schema, internalcontent.Schema)
	variantID := createTestVariant(t, db)
	return &contentH.Deps{Service: internalcontent.NewService(db)}, variantID
}

func createTestVariant(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	// The whole package shares one database; reset variants (cascading to
	// content_sections) so each test starts from a clean slate.
	_, err := db.Exec(`DELETE FROM variants`)
	require.NoError(t, err)
	var id int64
	err = db.QueryRow(`
		INSERT INTO variants (slug, name, description, weight, status, created_at, updated_at)
		VALUES ('test-variant', 'Test Variant', 'Test description', 50, 'active', NOW(), NOW())
		RETURNING id`).Scan(&id)
	require.NoError(t, err)
	return id
}

func mustStruct(t *testing.T, m map[string]interface{}) *structpb.Struct {
	t.Helper()
	s, err := structpb.NewStruct(m)
	require.NoError(t, err)
	return s
}

func create(t *testing.T, h *contentH.Deps, variantID int64, sectionType string, order int32, content map[string]interface{}) *landingv1.ContentSection {
	t.Helper()
	handler := contentH.NewConnectHandler(*h)
	resp, err := handler.CreateSection(context.Background(), connect.NewRequest(&landingv1.CreateSectionRequest{
		VariantId:   variantID,
		SectionType: sectionType,
		Content:     mustStruct(t, content),
		Order:       order,
	}))
	require.NoError(t, err)
	return resp.Msg.Section
}

func TestGetSections(t *testing.T) {
	deps, variantID := setup(t)
	handler := contentH.NewConnectHandler(*deps)
	content := map[string]interface{}{"title": "Test Hero", "cta_text": "Get Started"}
	create(t, deps, variantID, "hero", 0, content)
	create(t, deps, variantID, "features", 1, content)

	resp, err := handler.GetSections(context.Background(), connect.NewRequest(&landingv1.GetSectionsRequest{VariantId: variantID}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Sections, 2)
	require.EqualValues(t, 0, resp.Msg.Sections[0].Order)
	require.EqualValues(t, 1, resp.Msg.Sections[1].Order)
}

func TestGetPublicSectionsFiltersDisabled(t *testing.T) {
	deps, variantID := setup(t)
	handler := contentH.NewConnectHandler(*deps)
	create(t, deps, variantID, "hero", 0, map[string]interface{}{"t": "on"})
	// Insert a disabled section directly; it must be excluded from the public view.
	_, err := deps.Service.CreateSection(internalcontent.Section{VariantID: variantID, SectionType: "faq", Content: map[string]interface{}{"q": "?"}, Order: 1, Enabled: false})
	require.NoError(t, err)

	resp, err := handler.GetPublicSections(context.Background(), connect.NewRequest(&landingv1.GetPublicSectionsRequest{VariantId: variantID}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Sections, 1)
	require.Equal(t, "hero", resp.Msg.Sections[0].SectionType)
}

func TestGetSection(t *testing.T) {
	deps, variantID := setup(t)
	handler := contentH.NewConnectHandler(*deps)
	created := create(t, deps, variantID, "hero", 0, map[string]interface{}{"title": "Test Section"})

	resp, err := handler.GetSection(context.Background(), connect.NewRequest(&landingv1.GetSectionRequest{Id: created.Id}))
	require.NoError(t, err)
	require.Equal(t, created.Id, resp.Msg.Section.Id)
	require.Equal(t, "hero", resp.Msg.Section.SectionType)
	require.Equal(t, "Test Section", resp.Msg.Section.Content.AsMap()["title"])
}

func TestGetSectionNotFound(t *testing.T) {
	deps, _ := setup(t)
	handler := contentH.NewConnectHandler(*deps)
	_, err := handler.GetSection(context.Background(), connect.NewRequest(&landingv1.GetSectionRequest{Id: 99999}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestCreateSection(t *testing.T) {
	deps, variantID := setup(t)
	created := create(t, deps, variantID, "hero", 0, map[string]interface{}{"title": "Hero Title", "cta_text": "Get Started"})
	require.NotZero(t, created.Id)
	require.Equal(t, variantID, created.VariantId)
	require.Equal(t, "hero", created.SectionType)
	require.NotNil(t, created.CreatedAt)
	require.True(t, created.Enabled)
}

func TestUpdateSection(t *testing.T) {
	deps, variantID := setup(t)
	handler := contentH.NewConnectHandler(*deps)
	created := create(t, deps, variantID, "hero", 0, map[string]interface{}{"title": "Original Title"})

	resp, err := handler.UpdateSection(context.Background(), connect.NewRequest(&landingv1.UpdateSectionRequest{
		Id:      created.Id,
		Content: mustStruct(t, map[string]interface{}{"title": "Updated Title", "subtitle": "New Subtitle"}),
	}))
	require.NoError(t, err)
	m := resp.Msg.Section.Content.AsMap()
	require.Equal(t, "Updated Title", m["title"])
	require.Equal(t, "New Subtitle", m["subtitle"])
}

func TestDeleteSection(t *testing.T) {
	deps, variantID := setup(t)
	handler := contentH.NewConnectHandler(*deps)
	created := create(t, deps, variantID, "hero", 0, map[string]interface{}{"title": "Test"})

	resp, err := handler.DeleteSection(context.Background(), connect.NewRequest(&landingv1.DeleteSectionRequest{Id: created.Id}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Deleted)

	_, err = handler.GetSection(context.Background(), connect.NewRequest(&landingv1.GetSectionRequest{Id: created.Id}))
	require.Error(t, err)
}

func TestDeleteSectionNotFound(t *testing.T) {
	deps, _ := setup(t)
	handler := contentH.NewConnectHandler(*deps)
	_, err := handler.DeleteSection(context.Background(), connect.NewRequest(&landingv1.DeleteSectionRequest{Id: 99999}))
	require.Error(t, err)
}

func TestContentJSONMarshaling(t *testing.T) {
	deps, variantID := setup(t)
	handler := contentH.NewConnectHandler(*deps)
	created := create(t, deps, variantID, "features", 0, map[string]interface{}{
		"title": "Complex Content",
		"features": []interface{}{
			map[string]interface{}{"name": "Feature 1", "icon": "check"},
			map[string]interface{}{"name": "Feature 2", "icon": "star"},
		},
	})

	resp, err := handler.GetSection(context.Background(), connect.NewRequest(&landingv1.GetSectionRequest{Id: created.Id}))
	require.NoError(t, err)
	features, ok := resp.Msg.Section.Content.AsMap()["features"].([]interface{})
	require.True(t, ok)
	require.Len(t, features, 2)
}
