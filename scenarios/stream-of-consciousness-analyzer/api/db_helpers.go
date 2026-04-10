package main

import "database/sql"

// deleteByID executes a DELETE statement and returns sql.ErrNoRows if no row was affected.
// This consolidates the identical delete-and-check pattern used by all services.
func deleteByID(db *sql.DB, query string, id string) error {
	result, err := db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
