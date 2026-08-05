package infra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
)

const (
	denialWindowMinutes    = 15
	denialProbeTimeout     = 15 * time.Second
	credentialDenialMarker = "Credentials are not set, denying client"
)

type denialCounts struct {
	CredentialDenials int
	TotalDenials      int
	Readable          bool
}

// recentRDPDenials counts client denials without treating an empty window as
// evidence of health; no attempted connection is still an unknown signal.
func (c *RDPCheck) recentRDPDenials(ctx context.Context) denialCounts {
	ctx, cancel := context.WithTimeout(ctx, denialProbeTimeout)
	defer cancel()
	entries, err := journal.NewReader(c.executor).QueryLogs(ctx, journal.QueryOpts{
		UserUnit: []string{"gnome-remote-desktop"},
		Since:    fmt.Sprintf("%d minutes ago", denialWindowMinutes),
	})
	if err != nil {
		return denialCounts{}
	}
	counts := denialCounts{Readable: true}
	for _, entry := range entries {
		message := entry.Message
		if message == "" {
			message = entry.Raw
		}
		if strings.Contains(message, credentialDenialMarker) {
			counts.CredentialDenials++
		}
		if strings.Contains(message, "denying client") {
			counts.TotalDenials++
		}
	}
	return counts
}
