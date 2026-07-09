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
		CreatedBy:   result.Initiative.CreatedBy,
		PlanRef:     planRefToBacklog(result.Initiative.PlanRef),
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
		CreatedBy:   spec.CreatedBy,
		PlanRef:     planRefFromBacklog(spec.PlanRef),
	})
	return err
}

func (a *backlogAssignerAdapter) Update(spec backlog.InitiativeSpec) error {
	title := spec.Title
	description := spec.Description
	status := spec.Status
	priority := spec.Priority
	deps := append([]string(nil), spec.DependsOn...)
	planRef := planRefFromBacklog(spec.PlanRef)
	_, err := a.service.Update(spec.Name, UpdateRequest{
		Title:       &title,
		Description: &description,
		Status:      &status,
		Priority:    &priority,
		DependsOn:   &deps,
		PlanRef:     planRef,
		PlanRefSet:  true,
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
		CreatedBy:   snapshot.CreatedBy,
		PlanRef:     planRefFromBacklog(snapshot.PlanRef),
	})
}

func planRefFromBacklog(ref *backlog.PlanRef) *PlanRef {
	if ref == nil {
		return nil
	}
	return &PlanRef{
		Provider: ref.Provider,
		PlanID:   ref.PlanID,
		Slug:     ref.Slug,
		Role:     ref.Role,
	}
}

func planRefToBacklog(ref *PlanRef) *backlog.PlanRef {
	if ref == nil {
		return nil
	}
	return &backlog.PlanRef{
		Provider: ref.Provider,
		PlanID:   ref.PlanID,
		Slug:     ref.Slug,
		Role:     ref.Role,
	}
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
