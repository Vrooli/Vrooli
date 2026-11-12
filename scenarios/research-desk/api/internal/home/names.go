package home

import "regexp"

// NamePattern is the one rule for the names of topic folders, commissions and
// reviews: packet.py's NAME, which a test reads back (rulings E3). The
// lister, the charter links, the verdict heading and input validation all
// use it.
const NamePattern = `[A-Za-z0-9][A-Za-z0-9._-]{0,127}`

// FolderID matches a whole name (FILE-CONTRACT.md §2).
var FolderID = regexp.MustCompile(`^` + NamePattern + `$`)
