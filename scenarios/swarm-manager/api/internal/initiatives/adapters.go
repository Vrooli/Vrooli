package initiatives

import (
	"swarm-manager/internal/backlog"
)

// backlogAssignerAdapter bridges Service to the backlog.InitiativeAssigner
// interface, avoiding a direct import cycle (backlog cannot import initiatives).
type backlogAssignerAdapter struct {
	service *Service
}

// NewBacklogAssignerAdapter creates a backlog.InitiativeAssigner backed by the given Service.
func NewBacklogAssignerAdapter(service *Service) *backlogAssignerAdapter {
	return &backlogAssignerAdapter{service: service}
}

func (a *backlogAssignerAdapter) Get(name string) (*backlog.InitiativeSnapshot, error) {
	result, err := a.service.Get(name)
	if err != nil {
		return nil, err
	}
	return &backlog.InitiativeSnapshot{
		Name:        result.Initiative.Name,
		Title:       result.Initiative.Title,
		Description: result.Initiative.Description,
		Status:      result.Initiative.Status,
		Priority:    result.Initiative.Priority,
		DependsOn:   append([]string(nil), result.Initiative.DependsOn...),
		Items:       append([]string(nil), result.Initiative.Items...),
	}, nil
}

func (a *backlogAssignerAdapter) Create(spec backlog.InitiativeSpec) error {
	_, err := a.service.Create(CreateRequest{
		Name:        spec.Name,
		Title:       spec.Title,
		Description: spec.Description,
		Status:      spec.Status,
		Priority:    spec.Priority,
		DependsOn:   append([]string(nil), spec.DependsOn...),
	})
	return err
}

func (a *backlogAssignerAdapter) Update(spec backlog.InitiativeSpec) error {
	title := spec.Title
	description := spec.Description
	status := spec.Status
	priority := spec.Priority
	deps := append([]string(nil), spec.DependsOn...)
	_, err := a.service.Update(spec.Name, UpdateRequest{
		Title:       &title,
		Description: &description,
		Status:      &status,
		Priority:    &priority,
		DependsOn:   &deps,
	})
	return err
}

func (a *backlogAssignerAdapter) Replace(snapshot backlog.InitiativeSnapshot) error {
	return a.service.Replace(Initiative{
		Name:        snapshot.Name,
		Title:       snapshot.Title,
		Description: snapshot.Description,
		Status:      snapshot.Status,
		Priority:    snapshot.Priority,
		DependsOn:   append([]string(nil), snapshot.DependsOn...),
		Items:       append([]string(nil), snapshot.Items...),
	})
}

func (a *backlogAssignerAdapter) Delete(name string) error {
	return a.service.Delete(name)
}

func (a *backlogAssignerAdapter) AddItems(name string, items []string) error {
	return a.service.AddItems(name, items)
}

func (a *backlogAssignerAdapter) RememberItem(initiativeName, ref string) error {
	return a.service.RememberItem(initiativeName, ref)
}

func (a *backlogAssignerAdapter) ForgetItem(initiativeName, ref string) error {
	return a.service.ForgetItem(initiativeName, ref)
}
