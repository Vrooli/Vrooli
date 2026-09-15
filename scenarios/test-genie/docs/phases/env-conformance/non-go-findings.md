# Non-Go environment findings

The Go census is authoritative for producer resolution. Non-Go sources are
tracked separately because shell, Python, and Vite each have different parser
and runtime semantics.

The current lexical inventory on 2026-08-23 is:

| Surface | Matches | Status |
| --- | ---: | --- |
| Shell (`$VAR`/`${VAR}`) | 1,424 | follow-up parser required |
| Vite/JavaScript (`import.meta.env`) | 76 | runtime `VITE_*` reads are rejected outside `vite.config.*` |
| Python (`os.environ`/`getenv`) | 5 | follow-up parser required |

These are inventory counts, not producer classifications: shell and Python
matching still includes locally-scoped variables and Vite includes build-mode
builtins. The `UI_RUNTIME_CONFIG` structure-health rule provides the first
complete non-Go conformance boundary; the shell and Python counts are an
explicit follow-on rather than silently being reported as class C.
