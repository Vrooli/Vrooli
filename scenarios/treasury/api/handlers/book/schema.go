// Package book is the API composition boundary for custody books.
package book

import domain "treasury/internal/book"

func Schema() string { return domain.Schema() }
