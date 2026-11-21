package designs_test

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"

	designsH "react-component-library/handlers/designs"
	localdb "react-component-library/internal/database"
	designs "react-component-library/internal/designs"
)

func newRouter(t *testing.T) *mux.Router {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), sqlDB,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(designs.Schema),
	))
	require.NoError(t, designs.Migrate(context.Background(), sqlDB))
	m := designsH.Module(sqlDB)
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

func decodeDesign(t *testing.T, rec *httptest.ResponseRecorder) designs.Document {
	t.Helper()
	var payload struct {
		Design designs.Document `json:"design"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload), rec.Body.String())
	return payload.Design
}

func decodeAsset(t *testing.T, rec *httptest.ResponseRecorder) designs.Asset {
	t.Helper()
	var payload struct {
		Asset designs.Asset `json:"asset"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload), rec.Body.String())
	return payload.Asset
}

func pngBytes(width, height uint32) []byte {
	data := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 13}
	data = append(data, "IHDR"...)
	var dimensions [8]byte
	binary.BigEndian.PutUint32(dimensions[0:4], width)
	binary.BigEndian.PutUint32(dimensions[4:8], height)
	return append(data, dimensions[:]...)
}

func assetBody(name, mime string, data []byte) string {
	payload, err := json.Marshal(map[string]string{
		"name":       name,
		"mime":       mime,
		"dataBase64": base64.StdEncoding.EncodeToString(data),
	})
	if err != nil {
		panic(err)
	}
	return string(payload)
}

func TestModuleShape(t *testing.T) {
	require.Len(t, designsH.Endpoints, 11)
	ids := make([]string, 0, len(designsH.Endpoints))
	for _, endpoint := range designsH.Endpoints {
		ids = append(ids, endpoint.ID)
		require.NotNil(t, endpoint.RESTException, "%s must declare why it is REST", endpoint.ID)
	}
	require.Contains(t, ids, "designs_assist")
}

func TestCreateGetListSaveDelete(t *testing.T) {
	router := newRouter(t)

	created := do(t, router, http.MethodPost, "/api/designs", `{
		"name": "Checkout redesign",
		"frames": [{"id":"frame-1","name":"Desktop","device":"desktop","spec":{"root":{"kind":"column"}}}],
		"theme": {"ink":"#111111"}
	}`)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	doc := decodeDesign(t, created)
	require.NotEmpty(t, doc.ID)
	require.Equal(t, 1, doc.Revision)
	require.JSONEq(t, `[{"id":"frame-1","name":"Desktop","device":"desktop","spec":{"root":{"kind":"column"}}}]`, string(doc.Frames))
	require.JSONEq(t, `{"ink":"#111111"}`, string(doc.Theme))

	got := do(t, router, http.MethodGet, "/api/designs/"+doc.ID, "")
	require.Equal(t, http.StatusOK, got.Code, got.Body.String())
	require.Equal(t, doc.ID, decodeDesign(t, got).ID)

	listed := do(t, router, http.MethodGet, "/api/designs", "")
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())
	var listPayload struct {
		Designs []designs.Summary `json:"designs"`
	}
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &listPayload))
	require.Len(t, listPayload.Designs, 1)
	require.Equal(t, doc.ID, listPayload.Designs[0].ID)
	require.Equal(t, 1, listPayload.Designs[0].FrameCount)

	saved := do(t, router, http.MethodPut, "/api/designs/"+doc.ID, `{
		"id": "ignored-by-server",
		"updatedAt": "ignored-by-server",
		"name": "Checkout redesign v2",
		"frames": [],
		"theme": {}
	}`)
	require.Equal(t, http.StatusOK, saved.Code, saved.Body.String())
	savedDoc := decodeDesign(t, saved)
	require.Equal(t, doc.ID, savedDoc.ID, "the path id is authoritative")
	require.Equal(t, 2, savedDoc.Revision)
	require.Equal(t, "Checkout redesign v2", savedDoc.Name)
	require.JSONEq(t, `[]`, string(savedDoc.Frames))

	deleted := do(t, router, http.MethodDelete, "/api/designs/"+doc.ID, "")
	require.Equal(t, http.StatusNoContent, deleted.Code, deleted.Body.String())
	missing := do(t, router, http.MethodGet, "/api/designs/"+doc.ID, "")
	require.Equal(t, http.StatusNotFound, missing.Code, missing.Body.String())
}

func TestDocumentPartsAreRequired(t *testing.T) {
	router := newRouter(t)
	created := do(t, router, http.MethodPost, "/api/designs", `{"name":"Base","frames":[],"theme":{}}`)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	id := decodeDesign(t, created).ID

	cases := map[string]struct {
		method string
		path   string
		body   string
		field  string
	}{
		"create missing name":   {http.MethodPost, "/api/designs", `{"frames":[],"theme":{}}`, "name"},
		"create missing frames": {http.MethodPost, "/api/designs", `{"name":"x","theme":{}}`, "frames"},
		"create missing theme":  {http.MethodPost, "/api/designs", `{"name":"x","frames":[]}`, "theme"},
		"save missing name":     {http.MethodPut, "/api/designs/" + id, `{"frames":[],"theme":{}}`, "name"},
		"save missing frames":   {http.MethodPut, "/api/designs/" + id, `{"name":"x","theme":{}}`, "frames"},
		"save missing theme":    {http.MethodPut, "/api/designs/" + id, `{"name":"x","frames":[]}`, "theme"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(t, router, tc.method, tc.path, tc.body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"invalid_argument"`)
			require.Contains(t, rec.Body.String(), tc.field+" is required")
		})
	}
}

func TestCreateWithoutOptionalDocumentParts(t *testing.T) {
	router := newRouter(t)
	created := do(t, router, http.MethodPost, "/api/designs", `{"name":"Blank","frames":null,"theme":null}`)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	doc := decodeDesign(t, created)
	require.JSONEq(t, `[]`, string(doc.Frames))
	require.JSONEq(t, `{}`, string(doc.Theme))
	require.JSONEq(t, `{}`, string(doc.Options))
	require.Nil(t, doc.Scenario)
	require.Nil(t, doc.Page)
}

func TestCreateRejectsBadRequests(t *testing.T) {
	router := newRouter(t)

	for name, body := range map[string]string{
		"malformed json":   `{"name":`,
		"frames not json":  `{"name":"x","frames":"nope","theme":{}}`,
		"theme not object": `{"name":"x","frames":[],"theme":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, router, http.MethodPost, "/api/designs", body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"invalid_argument"`)
		})
	}
}

func TestBindingRoundTripAndFilter(t *testing.T) {
	router := newRouter(t)
	created := do(t, router, http.MethodPost, "/api/designs", `{
		"name":"Inbox",
		"frames":[],
		"theme":{},
		"scenario":"agent-inbox",
		"page":"inbox"
	}`)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	doc := decodeDesign(t, created)
	require.NotNil(t, doc.Scenario)
	require.Equal(t, "agent-inbox", *doc.Scenario)
	require.NotNil(t, doc.Page)
	require.Equal(t, "inbox", *doc.Page)

	other := do(t, router, http.MethodPost, "/api/designs", `{"name":"Loose","frames":[],"theme":{}}`)
	require.Equal(t, http.StatusCreated, other.Code, other.Body.String())

	var filtered struct {
		Designs []designs.Summary `json:"designs"`
	}
	rec := do(t, router, http.MethodGet, "/api/designs?scenario=agent-inbox&page=inbox", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &filtered))
	require.Len(t, filtered.Designs, 1)
	require.Equal(t, doc.ID, filtered.Designs[0].ID)
	require.Equal(t, "agent-inbox", *filtered.Designs[0].Scenario)
	require.Equal(t, "inbox", *filtered.Designs[0].Page)

	none := do(t, router, http.MethodGet, "/api/designs?scenario=agent-inbox&page=drafts", "")
	require.Equal(t, http.StatusOK, none.Code)
	require.NoError(t, json.Unmarshal(none.Body.Bytes(), &filtered))
	require.Empty(t, filtered.Designs)
}

func TestOptionsRoundTripAndDefaults(t *testing.T) {
	router := newRouter(t)

	created := do(t, router, http.MethodPost, "/api/designs", `{
		"name":"Chrome",
		"frames":[],
		"theme":{},
		"options":{"radius":20,"display":"sans"}
	}`)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	doc := decodeDesign(t, created)
	require.JSONEq(t, `{"radius":20,"display":"sans"}`, string(doc.Options))

	got := decodeDesign(t, do(t, router, http.MethodGet, "/api/designs/"+doc.ID, ""))
	require.JSONEq(t, `{"radius":20,"display":"sans"}`, string(got.Options))

	saved := do(t, router, http.MethodPut, "/api/designs/"+doc.ID, `{
		"name":"Chrome",
		"frames":[],
		"theme":{},
		"options":{"display":"serif"}
	}`)
	require.Equal(t, http.StatusOK, saved.Code, saved.Body.String())
	require.JSONEq(t, `{"display":"serif"}`, string(decodeDesign(t, saved).Options))

	// A pre-options client sends no options at all: the save still lands and
	// the document reports {} rather than null.
	legacy := do(t, router, http.MethodPut, "/api/designs/"+doc.ID, `{"name":"Legacy body","frames":[],"theme":{}}`)
	require.Equal(t, http.StatusOK, legacy.Code, legacy.Body.String())
	legacyDoc := decodeDesign(t, legacy)
	require.JSONEq(t, `{}`, string(legacyDoc.Options))
	require.Equal(t, "Legacy body", legacyDoc.Name)

	// Explicit null is the same as absent.
	defaulted := do(t, router, http.MethodPost, "/api/designs", `{"name":"Null","frames":[],"theme":{},"options":null}`)
	require.Equal(t, http.StatusCreated, defaulted.Code, defaulted.Body.String())
	require.JSONEq(t, `{}`, string(decodeDesign(t, defaulted).Options))
}

func TestOptionsValidationNamesTheField(t *testing.T) {
	router := newRouter(t)
	created := decodeDesign(t, do(t, router, http.MethodPost, "/api/designs", `{"name":"Base","frames":[],"theme":{}}`))
	oversized := `{"note":"` + strings.Repeat("x", 4200) + `"}`

	cases := map[string]struct {
		method  string
		path    string
		options string
		field   string
	}{
		"create unknown key": {http.MethodPost, "/api/designs", `{"shadow":true}`, "options.shadow"},
		"create bad radius":  {http.MethodPost, "/api/designs", `{"radius":49}`, "options.radius"},
		"create low radius":  {http.MethodPost, "/api/designs", `{"radius":-1}`, "options.radius"},
		"create bad display": {http.MethodPost, "/api/designs", `{"display":"mono"}`, "options.display"},
		"create non-object":  {http.MethodPost, "/api/designs", `[]`, "options must be a JSON object"},
		"create oversized":   {http.MethodPost, "/api/designs", oversized, "options must be at most 4 KiB"},
		"save unknown key":   {http.MethodPut, "/api/designs/" + created.ID, `{"shadow":true}`, "options.shadow"},
		"save bad radius":    {http.MethodPut, "/api/designs/" + created.ID, `{"radius":49}`, "options.radius"},
		"save bad display":   {http.MethodPut, "/api/designs/" + created.ID, `{"display":"mono"}`, "options.display"},
		"save non-object":    {http.MethodPut, "/api/designs/" + created.ID, `"sans"`, "options must be a JSON object"},
		"save oversized":     {http.MethodPut, "/api/designs/" + created.ID, oversized, "options must be at most 4 KiB"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			body := `{"name":"Base","frames":[],"theme":{},"options":` + tc.options + `}`
			rec := do(t, router, tc.method, tc.path, body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"invalid_argument"`)
			require.Contains(t, rec.Body.String(), tc.field)
		})
	}
}

func TestSaveRevisionConflictEnvelope(t *testing.T) {
	router := newRouter(t)
	created := do(t, router, http.MethodPost, "/api/designs", `{"name":"Base","frames":[],"theme":{}}`)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	doc := decodeDesign(t, created)

	first := do(t, router, http.MethodPut, "/api/designs/"+doc.ID, `{"name":"First","frames":[],"theme":{},"revision":1}`)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	require.Equal(t, 2, decodeDesign(t, first).Revision)

	stale := do(t, router, http.MethodPut, "/api/designs/"+doc.ID, `{"name":"Stale","frames":[],"theme":{},"revision":1}`)
	require.Equal(t, http.StatusConflict, stale.Code, stale.Body.String())
	var payload struct {
		Error  map[string]string `json:"error"`
		Design designs.Document  `json:"design"`
	}
	require.NoError(t, json.Unmarshal(stale.Body.Bytes(), &payload))
	require.Equal(t, "conflict", payload.Error["code"])
	require.Equal(t, "design was modified elsewhere", payload.Error["message"])
	require.Equal(t, doc.ID, payload.Design.ID)
	require.Equal(t, "First", payload.Design.Name)
	require.Equal(t, 2, payload.Design.Revision)

	unchanged := do(t, router, http.MethodGet, "/api/designs/"+doc.ID, "")
	require.Equal(t, http.StatusOK, unchanged.Code)
	require.Equal(t, "First", decodeDesign(t, unchanged).Name, "a conflict must not write")
}

func TestAssetLifecycle(t *testing.T) {
	router := newRouter(t)
	created := do(t, router, http.MethodPost, "/api/designs", `{"name":"Assets","frames":[],"theme":{}}`)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	doc := decodeDesign(t, created)

	upload := do(t, router, http.MethodPost, "/api/designs/"+doc.ID+"/assets", assetBody("hero.png", "image/png", pngBytes(640, 480)))
	require.Equal(t, http.StatusCreated, upload.Code, upload.Body.String())
	asset := decodeAsset(t, upload)
	require.NotEmpty(t, asset.ID)
	require.Equal(t, doc.ID, asset.DesignID)
	require.Equal(t, "hero.png", asset.Name)
	require.Equal(t, "image/png", asset.Mime)
	require.Equal(t, 640, asset.Width)
	require.Equal(t, 480, asset.Height)
	require.Equal(t, "reference", asset.Fidelity)
	require.Equal(t, "/api/designs/"+doc.ID+"/assets/"+asset.ID, asset.URL)

	listed := do(t, router, http.MethodGet, "/api/designs/"+doc.ID+"/assets", "")
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())
	var listPayload struct {
		Assets []designs.Asset `json:"assets"`
	}
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &listPayload))
	require.Len(t, listPayload.Assets, 1)
	require.Equal(t, asset.ID, listPayload.Assets[0].ID)

	bytePath := "/api/designs/" + doc.ID + "/assets/" + asset.ID
	bytes := do(t, router, http.MethodGet, bytePath, "")
	require.Equal(t, http.StatusOK, bytes.Code)
	require.Equal(t, "image/png", bytes.Header().Get("Content-Type"))
	require.Equal(t, "private, max-age=3600", bytes.Header().Get("Cache-Control"))
	require.Equal(t, asset.ID, bytes.Header().Get("ETag"))
	require.Equal(t, pngBytes(640, 480), bytes.Body.Bytes())

	conditional := httptest.NewRequest(http.MethodGet, bytePath, nil)
	conditional.Header.Set("If-None-Match", asset.ID)
	notModified := httptest.NewRecorder()
	router.ServeHTTP(notModified, conditional)
	require.Equal(t, http.StatusNotModified, notModified.Code)
	require.Empty(t, notModified.Body.Bytes())

	all := do(t, router, http.MethodGet, "/api/design-assets", "")
	require.Equal(t, http.StatusOK, all.Code, all.Body.String())
	require.NoError(t, json.Unmarshal(all.Body.Bytes(), &listPayload))
	require.Len(t, listPayload.Assets, 1)

	deleted := do(t, router, http.MethodDelete, bytePath, "")
	require.Equal(t, http.StatusNoContent, deleted.Code)
	missing := do(t, router, http.MethodGet, bytePath, "")
	require.Equal(t, http.StatusNotFound, missing.Code)
	require.Equal(t, http.StatusNoContent, do(t, router, http.MethodDelete, bytePath, "").Code, "delete is idempotent")
}

func TestAssetsAreScopedToTheirDesign(t *testing.T) {
	router := newRouter(t)
	owner := decodeDesign(t, do(t, router, http.MethodPost, "/api/designs", `{"name":"Owner","frames":[],"theme":{}}`))
	other := decodeDesign(t, do(t, router, http.MethodPost, "/api/designs", `{"name":"Other","frames":[],"theme":{}}`))
	asset := decodeAsset(t, do(t, router, http.MethodPost, "/api/designs/"+owner.ID+"/assets", assetBody("hero.png", "image/png", pngBytes(10, 10))))

	throughOther := "/api/designs/" + other.ID + "/assets/" + asset.ID
	require.Equal(t, http.StatusNotFound, do(t, router, http.MethodGet, throughOther, "").Code)
	require.Equal(t, http.StatusNoContent, do(t, router, http.MethodDelete, throughOther, "").Code, "delete is idempotent for the wrong design")

	throughOwner := "/api/designs/" + owner.ID + "/assets/" + asset.ID
	require.Equal(t, http.StatusOK, do(t, router, http.MethodGet, throughOwner, "").Code, "asset of A must survive a delete through B")

	require.Equal(t, http.StatusNoContent, do(t, router, http.MethodDelete, "/api/designs/"+owner.ID, "").Code)
	all := do(t, router, http.MethodGet, "/api/design-assets", "")
	require.Equal(t, http.StatusOK, all.Code)
	var payload struct {
		Assets []designs.Asset `json:"assets"`
	}
	require.NoError(t, json.Unmarshal(all.Body.Bytes(), &payload))
	require.Empty(t, payload.Assets, "deleting a design removes its assets")
}

func TestAssetUploadValidation(t *testing.T) {
	router := newRouter(t)
	doc := decodeDesign(t, do(t, router, http.MethodPost, "/api/designs", `{"name":"Assets","frames":[],"theme":{}}`))
	path := "/api/designs/" + doc.ID + "/assets"

	oversize := make([]byte, designs.MaxAssetBytes+1)
	cases := map[string]string{
		"empty name":     assetBody("   ", "image/png", pngBytes(1, 1)),
		"non-image mime": assetBody("x", "application/pdf", pngBytes(1, 1)),
		"invalid base64": `{"name":"x","mime":"image/png","dataBase64":"@@not-base64@@"}`,
		"empty data":     `{"name":"x","mime":"image/png","dataBase64":""}`,
		"over size":      assetBody("x", "image/png", oversize),
		"bad fidelity":   `{"name":"x","mime":"image/png","dataBase64":"` + base64.StdEncoding.EncodeToString(pngBytes(1, 1)) + `","fidelity":"rendered"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(t, router, http.MethodPost, path, body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"error"`)
			require.Contains(t, rec.Body.String(), `"invalid_argument"`)
		})
	}

	missingDesign := do(t, router, http.MethodPost, "/api/designs/design-nope/assets", assetBody("x", "image/png", pngBytes(1, 1)))
	require.Equal(t, http.StatusNotFound, missingDesign.Code, missingDesign.Body.String())
}

func TestUnknownDesignBehaviour(t *testing.T) {
	router := newRouter(t)

	get := do(t, router, http.MethodGet, "/api/designs/design-nope", "")
	require.Equal(t, http.StatusNotFound, get.Code)
	require.Contains(t, get.Body.String(), `"not_found"`)

	save := do(t, router, http.MethodPut, "/api/designs/design-nope", `{"name":"x","frames":[],"theme":{}}`)
	require.Equal(t, http.StatusNotFound, save.Code)

	assets := do(t, router, http.MethodGet, "/api/designs/design-nope/assets", "")
	require.Equal(t, http.StatusNotFound, assets.Code)

	deleted := do(t, router, http.MethodDelete, "/api/designs/design-nope", "")
	require.Equal(t, http.StatusNoContent, deleted.Code, "delete is idempotent")
}
