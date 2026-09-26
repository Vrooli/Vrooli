// Package authoring provides a typed, confirm-before-write seam for agent
// profiles. It deliberately does not call a model or write arbitrary JSON.
package authoring

import (
	"fmt"
	"strings"
)

type Draft struct {
	ID, DisplayName, Description string
	Scopes                       []string
	OwnerOnlyScopes              []string
}

type Writer interface {
	WriteAgent(Draft) error
}

type Service struct{ writer Writer }

func New(writer Writer) *Service { return &Service{writer: writer} }

func (s *Service) Draft(description string) (Draft, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return Draft{}, fmt.Errorf("agent description is required")
	}
	name := strings.Fields(description)[0]
	return Draft{DisplayName: name, Description: description, Scopes: []string{"read"}}, nil
}

// Confirm is the only write path. A draft is returned by Draft and remains
// inert until the operator explicitly confirms it.
func (s *Service) Confirm(d Draft) error {
	if s.writer == nil {
		return fmt.Errorf("agent writer unavailable")
	}
	if strings.TrimSpace(d.DisplayName) == "" || strings.TrimSpace(d.Description) == "" {
		return fmt.Errorf("draft identity and description are required")
	}
	for _, scope := range d.OwnerOnlyScopes {
		if strings.TrimSpace(scope) != "" {
			return fmt.Errorf("owner-only scopes require an explicit owner review")
		}
	}
	return s.writer.WriteAgent(d)
}
