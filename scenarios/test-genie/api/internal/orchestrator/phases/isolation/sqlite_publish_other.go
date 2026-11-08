//go:build !linux

package isolation

import "errors"

func publishSQLiteDirectory(string, string) error {
	return errors.New("private SQLite backing publication requires Linux no-replace directory rename")
}
