// Package domain provides core domain types for browser-automation-studio.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// PageStatus represents the lifecycle state of a page.
type PageStatus string

const (
	PageStatusActive PageStatus = "active"
	PageStatusClosed PageStatus = "closed"
)

// Page represents a browser page/tab within a recording session.
// Each session can have multiple pages, with one designated as the active page
// for frame streaming and input forwarding.
type Page struct {
	ID        uuid.UUID  `json:"id"`
	SessionID string     `json:"sessionId"`
	URL       string     `json:"url"`
	Title     string     `json:"title"`
	OpenerID  *uuid.UUID `json:"openerId,omitempty"` // Page that opened this one
	CreatedAt time.Time  `json:"createdAt"`
	ClosedAt  *time.Time `json:"closedAt,omitempty"`
	IsInitial bool       `json:"isInitial"` // First page in session
	Status    PageStatus `json:"status"`

	// DriverPageID is the Playwright driver's internal identifier for this page.
	// Used for communication with the driver but not exposed to clients.
	DriverPageID string `json:"-"`
}

// PageEventType represents the type of page lifecycle event.
type PageEventType string

const (
	PageEventCreated   PageEventType = "page_created"
	PageEventNavigated PageEventType = "page_navigated"
	PageEventClosed    PageEventType = "page_closed"
)

// PageEvent represents a page lifecycle event in the recording timeline.
// These events are interleaved with actions to form a unified timeline.
type PageEvent struct {
	ID        uuid.UUID     `json:"id"`
	Type      PageEventType `json:"type"`
	PageID    uuid.UUID     `json:"pageId"`
	URL       string        `json:"url,omitempty"`
	Title     string        `json:"title,omitempty"`
	OpenerID  *uuid.UUID    `json:"openerId,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// DriverPageEvent represents a page event received from the Playwright driver.
// This is the wire format for page lifecycle notifications from the driver.
type DriverPageEvent struct {
	SessionID          string `json:"sessionId"`
	DriverPageID       string `json:"driverPageId"` // Playwright's internal ID
	VrooliPageID       string `json:"vrooliPageId"` // Our UUID (echoed back by driver)
	EventType          string `json:"eventType"`    // "created" | "navigated" | "closed"
	URL                string `json:"url"`
	Title              string `json:"title"`
	OpenerDriverPageID string `json:"openerDriverPageId,omitempty"`
	Timestamp          string `json:"timestamp"`
}

// SetActivePageRequest is the request to switch which page receives
// frame streaming and input forwarding.
type SetActivePageRequest struct {
	SessionID    string `json:"sessionId"`
	DriverPageID string `json:"driverPageId"`
}

// PagesResponse is the response for listing pages in a session.
type PagesResponse struct {
	Pages        []*Page `json:"pages"`
	ActivePageID string  `json:"activePageId"`
}
