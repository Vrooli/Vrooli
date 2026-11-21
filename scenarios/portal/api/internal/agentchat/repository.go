package agentchat

import (
	"context"
	"errors"

	"portal/internal/integrations/agentmanager"
)

var (
	ErrAlreadyAdmitted = errors.New("agent message already admitted; reconcile its existing run")
	ErrBindingConflict = errors.New("agent run binding conflict")
)

type Binding struct{ ID, ChatID, MessageID, TaskID, RunID, BriefID string }

type AdmissionPage struct {
	Bindings      []Binding
	NextPageToken string
}

var ErrInvalidPage = errors.New("invalid admission page token or size")

type Repository interface {
	List(context.Context, string, int) (AdmissionPage, error)
	Reserve(context.Context, string, string) (Binding, error)
	SetBriefID(context.Context, string, string) error
	Bind(context.Context, string, agentmanager.Session) error
	Get(context.Context, string, string) (Binding, error)
}
