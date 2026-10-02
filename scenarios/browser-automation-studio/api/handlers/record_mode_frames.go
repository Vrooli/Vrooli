package handlers

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image/jpeg"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/performance"
)

const maxFrameHeaderBytes = 16 * 1024

type driverFrameHeader struct {
	Version    int                      `json:"version"`
	Source     *driver.FrameSource      `json:"source"`
	CapturedAt time.Time                `json:"captured_at"`
	Timing     *performance.FrameHeader `json:"timing,omitempty"`
}

// Source identity is mandatory and independent of optional timing telemetry.
func decodeDriverFrame(data []byte) ([]byte, *driverFrameHeader, error) {
	invalid := errors.New("invalid recording frame envelope")
	if len(data) < 4 {
		return nil, nil, invalid
	}
	length := int(binary.BigEndian.Uint32(data[:4]))
	if length == 0 || length > maxFrameHeaderBytes || length > len(data)-6 {
		return nil, nil, invalid
	}
	var header driverFrameHeader
	if err := json.Unmarshal(data[4:4+length], &header); err != nil {
		return nil, nil, err
	}
	payload := data[4+length:]
	if header.Version != 1 || header.Source == nil ||
		(header.Source.StreamKind != "recording" && header.Source.StreamKind != "execution") ||
		header.CapturedAt.IsZero() || payload[0] != 0xff || payload[1] != 0xd8 {
		return nil, nil, invalid
	}
	return payload, &header, nil
}

// Consult the current session on every admission, including after HTTP capture.
func (h *Handler) framePage(sessionID string, source *driver.FrameSource) (uuid.UUID, bool) {
	pageID, _, _, accepted := h.framePageMetadata(sessionID, source)
	return pageID, accepted
}

func (h *Handler) framePageMetadata(sessionID string, source *driver.FrameSource) (uuid.UUID, string, string, bool) {
	owner, ok := h.recordModeService.GetSession(sessionID)
	if !ok || owner == nil {
		return uuid.Nil, "", "", false
	}
	pageID, accepted := owner.FramePage(source)
	if !accepted {
		return uuid.Nil, "", "", false
	}
	page, ok := owner.Pages().GetPage(pageID)
	if !ok || page.Status != domain.PageStatusActive {
		return uuid.Nil, "", "", false
	}
	return pageID, page.Title, page.URL, true
}

func viewerFrame(sessionID string, pageID uuid.UUID, title, pageURL string, capturedAt time.Time, jpeg []byte) []byte {
	header, _ := json.Marshal(struct {
		Version    int       `json:"version"`
		SessionID  string    `json:"session_id"`
		PageID     uuid.UUID `json:"page_id"`
		PageTitle  string    `json:"page_title"`
		PageURL    string    `json:"page_url"`
		CapturedAt time.Time `json:"captured_at"`
	}{1, sessionID, pageID, title, pageURL, capturedAt})
	packet := make([]byte, 4+len(header)+len(jpeg))
	binary.BigEndian.PutUint32(packet, uint32(len(header)))
	copy(packet[4:], header)
	copy(packet[4+len(header):], jpeg)
	return packet
}

func jpegDimensions(data []byte) (int, int) {
	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return config.Width, config.Height
}
