package templates_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"

	templatesH "react-component-library/handlers/templates"
	localdb "react-component-library/internal/database"
	templates "react-component-library/internal/templates"
)

func newRouter(t *testing.T) *mux.Router {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), sqlDB,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(templates.Schema),
	))
	require.NoError(t, templates.Migrate(context.Background(), sqlDB))
	m := templatesH.Module(sqlDB)
	router := mux.NewRouter()
	m.Mount(router)
	return router
}

func do(t *testing.T, router *mux.Router, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeTemplate(t *testing.T, rec *httptest.ResponseRecorder) templates.Template {
	t.Helper()
	var payload struct {
		Template templates.Template `json:"template"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload), rec.Body.String())
	return payload.Template
}

const apertureBody = `{
	"id": "aperture-study-abc",
	"name": "Aperture study",
	"tagline": "Analytics workspace",
	"category": "analytics",
	"palette": {"bg": "#0c0e12", "accent": "#e0a45e"},
	"display": "serif",
	"radius": 16,
	"spec": {"id": "aperture.page", "type": "page", "props": {"pad": 40}, "children": []},
	"slotHint": "Primary action in the workspace header",
	"idea": "One number leads."
}`

func TestModuleShape(t *testing.T) {
	require.Len(t, templatesH.Endpoints, 4)
	for _, endpoint := range templatesH.Endpoints {
		require.NotNil(t, endpoint.RESTException, "%s must declare why it is REST", endpoint.ID)
	}
}

func TestCreateListUpdateDeleteEnvelopes(t *testing.T) {
	router := newRouter(t)

	created := do(t, router, http.MethodPost, "/api/templates", apertureBody)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	template := decodeTemplate(t, created)
	require.Equal(t, "aperture-study-abc", template.ID)
	require.Equal(t, "Aperture study", template.Name)
	require.Equal(t, "serif", template.Display)
	require.Equal(t, 16, template.Radius)
	require.NotEmpty(t, template.CreatedAt)
	require.NotEmpty(t, template.UpdatedAt)

	listed := do(t, router, http.MethodGet, "/api/templates", "")
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())
	var payload struct {
		Templates []templates.Template `json:"templates"`
	}
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &payload))
	require.Len(t, payload.Templates, 1)
	require.Equal(t, template.ID, payload.Templates[0].ID)

	updated := do(t, router, http.MethodPut, "/api/templates/aperture-study-abc", `{
		"id": "ignored-by-server",
		"name": "Aperture study v2",
		"display": "sans",
		"radius": 4,
		"spec": {"id": "aperture.page", "type": "page", "props": {}, "children": []}
	}`)
	require.Equal(t, http.StatusOK, updated.Code, updated.Body.String())
	updatedTemplate := decodeTemplate(t, updated)
	require.Equal(t, "aperture-study-abc", updatedTemplate.ID, "the path id is authoritative")
	require.Equal(t, "Aperture study v2", updatedTemplate.Name)
	require.Empty(t, updatedTemplate.Tagline, "PUT replaces fields; omitted ones reset to defaults")
	require.JSONEq(t, `{}`, string(updatedTemplate.Palette))

	deleted := do(t, router, http.MethodDelete, "/api/templates/aperture-study-abc", "")
	require.Equal(t, http.StatusNoContent, deleted.Code, deleted.Body.String())
	require.Empty(t, deleted.Body.Bytes())

	again := do(t, router, http.MethodDelete, "/api/templates/aperture-study-abc", "")
	require.Equal(t, http.StatusNoContent, again.Code, "delete is idempotent")

	var empty struct {
		Templates []templates.Template `json:"templates"`
	}
	require.NoError(t, json.Unmarshal(do(t, router, http.MethodGet, "/api/templates", "").Body.Bytes(), &empty))
	require.Empty(t, empty.Templates)
}

func TestCreateDuplicateIDReturnsConflict(t *testing.T) {
	router := newRouter(t)
	require.Equal(t, http.StatusCreated, do(t, router, http.MethodPost, "/api/templates", apertureBody).Code)

	duplicate := do(t, router, http.MethodPost, "/api/templates", apertureBody)
	require.Equal(t, http.StatusConflict, duplicate.Code, duplicate.Body.String())
	require.Contains(t, duplicate.Body.String(), `"conflict"`)
}

func TestListOrdersNewestUpdatedFirst(t *testing.T) {
	router := newRouter(t)
	first := decodeTemplate(t, do(t, router, http.MethodPost, "/api/templates", `{"id":"first","name":"First"}`))
	second := decodeTemplate(t, do(t, router, http.MethodPost, "/api/templates", `{"id":"second","name":"Second"}`))

	var listed struct {
		Templates []templates.Template `json:"templates"`
	}
	require.NoError(t, json.Unmarshal(do(t, router, http.MethodGet, "/api/templates", "").Body.Bytes(), &listed))
	require.Len(t, listed.Templates, 2)
	require.Equal(t, second.ID, listed.Templates[0].ID)
	require.Equal(t, first.ID, listed.Templates[1].ID)
	require.GreaterOrEqual(t, listed.Templates[0].UpdatedAt, listed.Templates[1].UpdatedAt)
}

func TestUpdateMissingReturnsNotFound(t *testing.T) {
	router := newRouter(t)
	rec := do(t, router, http.MethodPut, "/api/templates/design-nope", `{"name":"Nope"}`)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"not_found"`)
}

func TestRequiredAndInvalidFieldsNameTheField(t *testing.T) {
	router := newRouter(t)

	cases := map[string]struct {
		body  string
		field string
	}{
		"create missing id":      {`{"name":"x"}`, "id is required"},
		"create bad id":          {`{"id":"Not A Slug","name":"x"}`, "id must match"},
		"create missing name":    {`{"id":"ok-id"}`, "name is required"},
		"create blank name":      {`{"id":"ok-id","name":"  "}`, "name is required"},
		"create long name":       {`{"id":"ok-id","name":"` + strings.Repeat("n", 121) + `"}`, "name must be at most"},
		"create long tagline":    {`{"id":"ok-id","name":"x","tagline":"` + strings.Repeat("t", 241) + `"}`, "tagline must be at most"},
		"create long category":   {`{"id":"ok-id","name":"x","category":"` + strings.Repeat("c", 81) + `"}`, "category must be at most"},
		"create bad display":     {`{"id":"ok-id","name":"x","display":"mono"}`, "display must be"},
		"create bad radius high": {`{"id":"ok-id","name":"x","radius":49}`, "radius must be between"},
		"create bad radius low":  {`{"id":"ok-id","name":"x","radius":-1}`, "radius must be between"},
		"create palette array":   {`{"id":"ok-id","name":"x","palette":[]}`, "palette must be a JSON object"},
		"create spec string":     {`{"id":"ok-id","name":"x","spec":"page"}`, "spec must be a JSON object"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(t, router, http.MethodPost, "/api/templates", tc.body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"invalid_argument"`)
			require.Contains(t, rec.Body.String(), tc.field)
		})
	}

	created := do(t, router, http.MethodPost, "/api/templates", `{"id":"ok-id","name":"Base"}`)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())

	for name, tc := range map[string]struct{ body, field string }{
		"update missing name": {`{"display":"sans"}`, "name is required"},
		"update bad display":  {`{"name":"x","display":"mono"}`, "display must be"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, router, http.MethodPut, "/api/templates/ok-id", tc.body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), tc.field)
		})
	}
}

func TestJSONLimitsAreRejectedByField(t *testing.T) {
	router := newRouter(t)
	oversizedSpec := `{"id":"big","name":"Big","spec":{"data":"` + strings.Repeat("a", templates.MaxSpecBytes) + `"}}`
	rec := do(t, router, http.MethodPost, "/api/templates", oversizedSpec)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "spec must be at most")

	oversizedPalette := `{"id":"big","name":"Big","palette":{"data":"` + strings.Repeat("a", templates.MaxPaletteBytes) + `"}}`
	rec = do(t, router, http.MethodPost, "/api/templates", oversizedPalette)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "palette must be at most")
}

func TestUnknownAndMalformedBodiesAreRejected(t *testing.T) {
	router := newRouter(t)

	unknown := do(t, router, http.MethodPost, "/api/templates", `{"id":"ok-id","name":"x","shelf":"nope"}`)
	require.Equal(t, http.StatusBadRequest, unknown.Code, unknown.Body.String())
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(unknown.Body.Bytes(), &envelope))
	require.Equal(t, "invalid_argument", envelope.Error.Code)
	require.Equal(t, `unknown field "shelf"`, envelope.Error.Message)

	for name, body := range map[string]string{
		"malformed json": `{"id":`,
		"trailing value": `{"id":"ok-id","name":"x"}{"id":"two"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, router, http.MethodPost, "/api/templates", body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"invalid_argument"`)
		})
	}
}
