# go-mislocated fixture

A tiny Go module where `wrongdir/m.go` declares `package right` despite
living under `wrongdir/`. `golang.org/x/tools/go/packages` loads each
directory as a single package, so the loader reports the mislocated file
as a list-error surfaced through `packages.Package.Errors`. Our
extractor classifies that as an `unresolved_import`/`type_check_failure`
warning rather than aborting the whole extraction.

The exact shape of the loader's error message can vary across Go and
`x/tools` versions, which is why the expected graph captures only the
structural nodes/edges that survive — the warning text itself is not
required to be byte-stable. If the loader's behaviour changes enough to
break the fixture, regenerate `expected-graph.json` with
`UPDATE_FIXTURES=1` and review the diff.
