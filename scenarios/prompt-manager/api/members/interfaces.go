// Package members provides types and operations for member management.
package members

// MemberStore defines the interface for member storage operations.
// This is the testing seam for the members domain.
// Implementations: Store (file-based, production), MockStore (testing).
type MemberStore interface {
	// GetAll returns all members.
	GetAll() ([]Member, error)

	// FindByID finds a member by ID.
	FindByID(id string) (*Member, error)

	// Create adds a new member.
	Create(member Member) error

	// Update modifies an existing member.
	Update(member Member) error

	// Delete removes a member by ID.
	Delete(id string) error
}
