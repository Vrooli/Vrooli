package planview

import "context"

// CaptureEntry is the minimal capture view the classify source consumes.
type CaptureEntry struct {
	ID              string
	Text            string
	Status          string
	ClassifiedItems int
	CreatedAt       string
}

// CaptureLister lists captures with their classification summary.
type CaptureLister interface {
	ListCaptures() ([]CaptureEntry, error)
}

// ClassifySource enumerates KindClassify gates: captures whose
// classification finished and now awaits human confirmation. Captures
// still classifying are in flight (an agent activity) and belong to the
// Now column, not the gate list.
type ClassifySource struct {
	Captures CaptureLister
}

// Name identifies the source in degradation logs.
func (s ClassifySource) Name() string { return "classify" }

// classifyTitleLimit bounds the capture-text excerpt used as the gate title.
const classifyTitleLimit = 80

// Enumerate implements Source.
func (s ClassifySource) Enumerate(_ context.Context) ([]Gate, error) {
	caps, err := s.Captures.ListCaptures()
	if err != nil {
		return nil, err
	}
	var out []Gate
	for _, cap := range caps {
		if cap.Status != "classified" || cap.ClassifiedItems == 0 {
			continue
		}
		title := cap.Text
		if len(title) > classifyTitleLimit {
			title = title[:classifyTitleLimit]
		}
		if title == "" {
			title = cap.ID
		}
		out = append(out, Gate{
			ID:             GateID(KindClassify, "capture", cap.ID),
			Kind:           KindClassify,
			OwnerType:      "capture",
			OwnerName:      cap.ID,
			OwnerTitle:     title,
			Count:          cap.ClassifiedItems,
			DecidableSince: cap.CreatedAt,
		})
	}
	return out, nil
}
