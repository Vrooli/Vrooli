package main

import "database/sql"

// scanner is the interface shared by *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanScheme scans a single row into a Scheme.
func scanScheme(sc scanner) (Scheme, error) {
	var s Scheme
	err := sc.Scan(&s.ID, &s.Name, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

// scanThought scans a single row into a Thought.
// Centralizes the 8-column scan order so adding/removing fields only requires
// one change site instead of five (Create, List, GetByID, Update, Export).
func scanThought(sc scanner) (Thought, error) {
	var t Thought
	err := sc.Scan(&t.ID, &t.SchemeID, &t.Title, &t.Body,
		&t.CanvasX, &t.CanvasY, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

// scanInformation scans a single row into an Information item.
func scanInformation(sc scanner) (Information, error) {
	var info Information
	err := sc.Scan(&info.ID, &info.SchemeID, &info.Type, &info.Content,
		&info.CanvasX, &info.CanvasY, &info.CreatedAt, &info.UpdatedAt)
	return info, err
}

// scanEdge scans a single row into a ThoughtEdge.
func scanEdge(sc scanner) (ThoughtEdge, error) {
	var e ThoughtEdge
	err := sc.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.Label, &e.CreatedAt)
	return e, err
}

// collectRows iterates sql.Rows, scanning each row with scanFn.
// Returns a non-nil empty slice when no rows are present.
func collectRows[T any](rows *sql.Rows, scanFn func(scanner) (T, error)) ([]T, error) {
	defer rows.Close()
	var items []T
	for rows.Next() {
		item, err := scanFn(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = make([]T, 0)
	}
	return items, rows.Err()
}
