package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestNewPageMatcher(t *testing.T) {
	page1ID := uuid.New()
	page2ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
			{ID: page2ID, IsInitial: false, OpenerID: &page1ID, URLPattern: ".*popup.*"},
		},
	}

	config := DefaultMultiPageConfig()
	pm := NewPageMatcher(plan, config, nil)

	assert.NotNil(t, pm)
	assert.Equal(t, 1, len(pm.pendingPages)) // Only non-initial pages are pending
	assert.Contains(t, pm.pendingPages, page2ID)
}

func TestPageMatcher_RegisterInitialPage(t *testing.T) {
	pageID := uuid.New()
	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: pageID, IsInitial: true, StartURL: "https://example.com"},
		},
	}

	config := DefaultMultiPageConfig()
	pm := NewPageMatcher(plan, config, nil)

	err := pm.RegisterInitialPage("driver-page-123", "https://example.com")
	require.NoError(t, err)

	driverID, ok := pm.GetDriverPageID(pageID)
	assert.True(t, ok)
	assert.Equal(t, "driver-page-123", driverID)
}

func TestPageMatcher_RegisterInitialPage_NoPlan(t *testing.T) {
	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages:         []contracts.PageDefinition{}, // No pages
	}

	config := DefaultMultiPageConfig()
	pm := NewPageMatcher(plan, config, nil)

	err := pm.RegisterInitialPage("driver-page-123", "https://example.com")
	assert.Error(t, err)
}

func TestPageMatcher_OnPageCreated_MatchByOpener(t *testing.T) {
	page1ID := uuid.New()
	page2ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
			{ID: page2ID, IsInitial: false, OpenerID: &page1ID},
		},
	}

	config := DefaultMultiPageConfig()
	config.MatchStrategy = MatchByOpener
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	// Simulate a new page being created by the initial page
	matchedID := pm.OnPageCreated("driver-page-2", "https://example.com/popup", "Popup", "driver-page-1")

	assert.NotNil(t, matchedID)
	assert.Equal(t, page2ID, *matchedID)

	// Verify the page is now mapped
	driverID, ok := pm.GetDriverPageID(page2ID)
	assert.True(t, ok)
	assert.Equal(t, "driver-page-2", driverID)

	// Pending pages should be empty now
	assert.False(t, pm.HasPendingPages())
}

func TestPageMatcher_OnPageCreated_MatchByURL(t *testing.T) {
	page1ID := uuid.New()
	page2ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
			{ID: page2ID, IsInitial: false, URLPattern: ".*checkout.*"},
		},
	}

	config := DefaultMultiPageConfig()
	config.MatchStrategy = MatchByURL
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	// Simulate a new page with matching URL pattern
	matchedID := pm.OnPageCreated("driver-page-2", "https://example.com/checkout?id=123", "Checkout", "")

	assert.NotNil(t, matchedID)
	assert.Equal(t, page2ID, *matchedID)
}

func TestPageMatcher_OnPageCreated_NoMatch(t *testing.T) {
	page1ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
		},
	}

	config := DefaultMultiPageConfig()
	config.EnableFallback = false // Disable fallback matching
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	// Simulate an unexpected page
	matchedID := pm.OnPageCreated("driver-page-2", "https://unexpected.com", "Unexpected", "")

	assert.Nil(t, matchedID)
}

func TestPageMatcher_OnPageCreated_FallbackSinglePending(t *testing.T) {
	page1ID := uuid.New()
	page2ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
			{ID: page2ID, IsInitial: false}, // No opener or URL pattern
		},
	}

	config := DefaultMultiPageConfig()
	config.EnableFallback = true // Enable fallback
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	// With fallback enabled and only one pending page, it should match
	matchedID := pm.OnPageCreated("driver-page-2", "https://any-url.com", "Any Page", "")

	assert.NotNil(t, matchedID)
	assert.Equal(t, page2ID, *matchedID)
}

func TestPageMatcher_OnPageClosed(t *testing.T) {
	pageID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: pageID, IsInitial: true, StartURL: "https://example.com"},
		},
	}

	config := DefaultMultiPageConfig()
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	// Close the page
	pm.OnPageClosed("driver-page-1")

	// Verify the page is marked as closed
	pages := pm.GetAllRuntimePages()
	require.Len(t, pages, 1)
	assert.True(t, pages[0].Closed)
}

func TestPageMatcher_WaitForPage_AlreadyMatched(t *testing.T) {
	pageID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: pageID, IsInitial: true, StartURL: "https://example.com"},
		},
	}

	config := DefaultMultiPageConfig()
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	// WaitForPage should return immediately since page is already matched
	ctx := context.Background()
	driverID, err := pm.WaitForPage(ctx, pageID)

	require.NoError(t, err)
	assert.Equal(t, "driver-page-1", driverID)
}

func TestPageMatcher_WaitForPage_Timeout(t *testing.T) {
	page1ID := uuid.New()
	page2ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
			{ID: page2ID, IsInitial: false},
		},
	}

	config := DefaultMultiPageConfig()
	config.PageWaitTimeout = 200 * time.Millisecond // Short timeout for test
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page but NOT the second page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	// WaitForPage should timeout
	ctx := context.Background()
	_, err = pm.WaitForPage(ctx, page2ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestPageMatcher_WaitForPage_ContextCancelled(t *testing.T) {
	page1ID := uuid.New()
	page2ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
			{ID: page2ID, IsInitial: false},
		},
	}

	config := DefaultMultiPageConfig()
	config.PageWaitTimeout = 10 * time.Second // Long timeout
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = pm.WaitForPage(ctx, page2ID)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestDefaultMultiPageConfig(t *testing.T) {
	config := DefaultMultiPageConfig()

	assert.Equal(t, 30*time.Second, config.PageWaitTimeout)
	assert.Equal(t, MatchByOpener, config.MatchStrategy)
	assert.True(t, config.EnableFallback)
}

func TestEnsureMultiPagePlan_V1ToV2(t *testing.T) {
	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersion, // v1
		WorkflowID:    uuid.New(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "node-1"},
			{Index: 1, NodeID: "node-2"},
		},
	}

	result := EnsureMultiPagePlan(plan)

	assert.Equal(t, contracts.ExecutionPlanSchemaVersionV2, result.SchemaVersion)
	assert.Len(t, result.Pages, 1)
	assert.True(t, result.Pages[0].IsInitial)

	// All instructions should have the same page ID
	assert.NotNil(t, result.Instructions[0].PageID)
	assert.NotNil(t, result.Instructions[1].PageID)
	assert.Equal(t, *result.Instructions[0].PageID, *result.Instructions[1].PageID)
}

func TestEnsureMultiPagePlan_AlreadyV2(t *testing.T) {
	pageID := uuid.New()
	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2, // Already v2
		WorkflowID:    uuid.New(),
		Pages: []contracts.PageDefinition{
			{ID: pageID, IsInitial: true},
		},
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, NodeID: "node-1", PageID: &pageID},
		},
	}

	result := EnsureMultiPagePlan(plan)

	// Should be unchanged
	assert.Equal(t, plan, result)
}

func TestEnsureMultiPagePlan_Nil(t *testing.T) {
	result := EnsureMultiPagePlan(nil)
	assert.Nil(t, result)
}

func TestPageEventHandler_HandlePageCreated(t *testing.T) {
	page1ID := uuid.New()
	page2ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
			{ID: page2ID, IsInitial: false, OpenerID: &page1ID},
		},
	}

	config := DefaultMultiPageConfig()
	pm := NewPageMatcher(plan, config, nil)

	// Register initial page
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	handler := NewPageEventHandler(pm, nil)

	// Handle page created event
	handler.HandlePageCreated("driver-page-2", "https://example.com/popup", "Popup", "driver-page-1")

	// Verify the page was matched
	driverID, ok := pm.GetDriverPageID(page2ID)
	assert.True(t, ok)
	assert.Equal(t, "driver-page-2", driverID)
}

func TestPageEventHandler_HandlePageClosed(t *testing.T) {
	pageID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: pageID, IsInitial: true, StartURL: "https://example.com"},
		},
	}

	config := DefaultMultiPageConfig()
	pm := NewPageMatcher(plan, config, nil)

	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	handler := NewPageEventHandler(pm, nil)

	// Handle page closed event
	handler.HandlePageClosed("driver-page-1")

	// Verify page is marked as closed
	pages := pm.GetAllRuntimePages()
	require.Len(t, pages, 1)
	assert.True(t, pages[0].Closed)
}

func TestGetAllRuntimePages(t *testing.T) {
	page1ID := uuid.New()
	page2ID := uuid.New()

	plan := &contracts.ExecutionPlan{
		SchemaVersion: contracts.ExecutionPlanSchemaVersionV2,
		Pages: []contracts.PageDefinition{
			{ID: page1ID, IsInitial: true, StartURL: "https://example.com"},
			{ID: page2ID, IsInitial: false, OpenerID: &page1ID},
		},
	}

	config := DefaultMultiPageConfig()
	pm := NewPageMatcher(plan, config, nil)

	// Register both pages
	err := pm.RegisterInitialPage("driver-page-1", "https://example.com")
	require.NoError(t, err)

	pm.OnPageCreated("driver-page-2", "https://example.com/popup", "Popup", "driver-page-1")

	pages := pm.GetAllRuntimePages()
	assert.Len(t, pages, 2)

	// Verify page data
	pageURLs := make(map[string]bool)
	for _, p := range pages {
		pageURLs[p.URL] = true
	}
	assert.True(t, pageURLs["https://example.com"])
	assert.True(t, pageURLs["https://example.com/popup"])
}
