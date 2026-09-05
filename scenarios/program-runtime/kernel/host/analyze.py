"""Static name analysis for one submitted program.

The Go pre-flight resolver calls this script instead of re-implementing Python
scoping. `symtable` is the compiler's own scope analysis, so a `def`, `lambda`,
comprehension, `for` target, `with ... as`, `except ... as`, walrus, match
capture, and closure are all classified correctly rather than approximated.

Contract: read source on stdin, write one JSON object on stdout.

    {"ok": true,
     "free": [{"name": "test_geni", "line": 3}],
     "imports": [{"name": "json", "line": 1}],
     "shadowed": [{"name": "search_hub", "line": 7}],
     "attributes": [{"name": "browser_automation_studio.capture.run", "line": 4}]}

    {"ok": false, "syntax_error": {"message": "...", "line": 4}}

`free` are names the program reads that nothing in the program binds and that
are not builtins — the exact set the kernel would resolve through globals. `Go`
decides severity by joining them against the governed binding registry; this
script never decides policy.

**The analysis is deliberately conservative.** A false negative is harmless
because the kernel still raises at runtime. A false positive refuses a correct
program, which is the defect this script replaced. When a construct cannot be
classified with certainty, it is omitted from `free`.
"""
from __future__ import annotations

import ast
import json
import symtable
import sys

try:
    from safebuiltins import SAFE_BUILTIN_NAMES
except ImportError:  # executed as a bare path without the package dir on sys.path
    sys.path.insert(0, __file__.rsplit("/", 1)[0])
    from safebuiltins import SAFE_BUILTIN_NAMES

_BUILTINS = frozenset(SAFE_BUILTIN_NAMES)


def _bound_at_module(table: symtable.SymbolTable) -> set[str]:
    """Names the module scope binds: assignments, defs, classes, and imports."""
    bound = set()
    for symbol in table.get_symbols():
        if symbol.is_assigned() or symbol.is_imported() or symbol.is_parameter():
            bound.add(symbol.get_name())
    return bound


def _free_names(table: symtable.SymbolTable, module_bound: set[str], out: set[str]) -> None:
    """Collect names that will be looked up in globals and are bound nowhere.

    In the module table a name qualifies when it is referenced and never bound.
    In a nested table `is_global()` means the compiler resolved the name to
    module scope, so it qualifies only when the module never binds it either.
    """
    is_module = table.get_type() == "module"
    for symbol in table.get_symbols():
        name = symbol.get_name()
        if not symbol.is_referenced():
            continue
        if name in module_bound or name in _BUILTINS:
            continue
        if name.startswith("__") and name.endswith("__"):
            continue
        if is_module:
            if not (symbol.is_assigned() or symbol.is_imported() or symbol.is_parameter()):
                out.add(name)
            continue
        # A nested scope: only a resolved-to-global reference can reach the
        # kernel globals. Locals, parameters, and closure variables cannot.
        if symbol.is_global() and not symbol.is_local() and not symbol.is_parameter():
            out.add(name)
    for child in table.get_children():
        _free_names(child, module_bound, out)


def _first_line(tree: ast.AST, names: set[str], want_store: bool) -> dict[str, int]:
    """Map each name to the first line where it is read (or written)."""
    lines: dict[str, int] = {}
    for node in ast.walk(tree):
        if not isinstance(node, ast.Name) or node.id not in names:
            continue
        stored = isinstance(node.ctx, ast.Store)
        if stored != want_store:
            continue
        line = getattr(node, "lineno", 0)
        if node.id not in lines or line < lines[node.id]:
            lines[node.id] = line
    return lines


def _imports(tree: ast.AST) -> list[dict[str, int | str]]:
    """Report imported module roots without deciding whether they are allowed.

    The bound name is deliberately not used here: ``import vrooli as v`` still
    imports the protected ``vrooli`` module root, while ``from json import
    loads`` imports the admitted ``json`` root. Policy remains on the Go side.
    """
    imports: list[dict[str, int | str]] = []
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                imports.append({"name": alias.name.split(".", 1)[0], "line": int(node.lineno)})
        elif isinstance(node, ast.ImportFrom) and node.module:
            imports.append({"name": node.module.split(".", 1)[0], "line": int(node.lineno)})
    return sorted(imports, key=lambda item: (int(item["line"]), str(item["name"])))


def _attribute_chain(node: ast.AST) -> str | None:
    parts: list[str] = []
    current = node
    while isinstance(current, ast.Attribute):
        parts.append(current.attr)
        current = current.value
    if not isinstance(current, ast.Name):
        return None
    parts.append(current.id)
    return ".".join(reversed(parts))


def _attributes(tree: ast.AST, module_bound: set[str]) -> list[dict[str, int | str]]:
    found: dict[str, int] = {}
    for node in ast.walk(tree):
        if not isinstance(node, ast.Attribute) or isinstance(node.ctx, ast.Store):
            continue
        chain = _attribute_chain(node)
        if not chain:
            continue
        root = chain.split(".", 1)[0]
        if root in module_bound:
            continue
        line = int(getattr(node, "lineno", 0))
        found.setdefault(chain, line)
    return [{"name": name, "line": line} for name, line in sorted(found.items())]


def analyze(source: str) -> dict:
    try:
        tree = ast.parse(source, "<program>", "exec")
        table = symtable.symtable(source, "<program>", "exec")
    except SyntaxError as exc:
        return {"ok": False, "syntax_error": {"message": str(exc.msg or exc), "line": int(exc.lineno or 0)}}
    except (ValueError, MemoryError, RecursionError) as exc:
        # An un-analyzable program is not a refusable program.
        return {"ok": True, "free": [], "shadowed": [], "degraded": f"{type(exc).__name__}: {exc}"}

    module_bound = _bound_at_module(table)
    free: set[str] = set()
    _free_names(table, module_bound, free)
    free_lines = _first_line(tree, free, want_store=False)

    shadow_lines = _first_line(tree, module_bound, want_store=True)

    return {
        "ok": True,
        "bound": [{"name": name} for name in sorted(module_bound)],
        "free": [{"name": name, "line": free_lines.get(name, 0)} for name in sorted(free)],
        "imports": _imports(tree),
        "shadowed": [{"name": name, "line": line} for name, line in sorted(shadow_lines.items())],
        "attributes": _attributes(tree, module_bound),
    }


def main() -> None:
    source = sys.stdin.read()
    try:
        result = analyze(source)
    except Exception as exc:  # noqa: BLE001 - never refuse a program because analysis broke
        result = {"ok": True, "free": [], "shadowed": [], "degraded": f"{type(exc).__name__}: {exc}"}
    sys.stdout.write(json.dumps(result, separators=(",", ":")))
    sys.stdout.flush()


if __name__ == "__main__":
    main()
