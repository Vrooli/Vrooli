package projects

import "errors"

const (
	StatusActive = "active"
	UnassignedID = "unassigned"
)

var ErrInvalidName = errors.New("project name must be non-empty and cannot be reserved 'unassigned'")

type Project struct {
	ID, Name, Status string
}
