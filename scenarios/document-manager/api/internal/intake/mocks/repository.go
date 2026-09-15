package mocks

import "document-manager/internal/intake"

type Repository struct{ Documents []intake.Document }

func (r *Repository) Create(d intake.Document) (intake.Document, error) {
	r.Documents = append(r.Documents, d)
	return d, nil
}

func (r *Repository) Get(id string) (intake.Document, error) {
	for _, d := range r.Documents {
		if d.ID == id {
			return d, nil
		}
	}
	return intake.Document{}, intake.ErrNotFound{Key: id}
}

func (r *Repository) List(limit int) ([]intake.Document, error) {
	if limit <= 0 || limit > len(r.Documents) {
		limit = len(r.Documents)
	}
	return r.Documents[:limit], nil
}
func (r *Repository) ListSources() ([]string, error) { return nil, nil }
func (r *Repository) FindByHash(hash string) (intake.Document, error) {
	for _, d := range r.Documents {
		if d.ContentSHA256 == hash {
			return d, nil
		}
	}
	return intake.Document{}, intake.ErrNotFound{Key: hash}
}
