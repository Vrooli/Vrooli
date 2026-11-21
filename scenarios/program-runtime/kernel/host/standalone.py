"""Run a binding-free declared program without Program Runtime.

    python3 -I standalone.py <contract.json> [key=value ...] [--inputs-json '{...}']

This is the outage path for research programs (and any binding-free contract):
`program-runtime library run research/...` uses it when the API cannot be
reached. It binds the same `program` helper the kernel binds, resolves the
declared inputs (unknown, required, default, basic type), executes the
program's source and prints its envelope. Exit 0 when the envelope status is
ok or partial, 1 for any other status, 3 when the program needs the kernel.

Governed bindings and runtime verbs (ai, gather, lib, tasks, learn, ...) exist
only inside Program Runtime. A program that declares bindings or names a
runtime verb is refused before it runs; read its .py and do its steps by hand.
"""
from __future__ import annotations

import ast
import builtins
import contextlib
import io
import json
import math
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from program_helper import SUPPORTED_HELPER_VERSIONS, ProgramHelper  # noqa: E402

# The kernel surface a standalone run cannot provide (engine._RUNTIME_VERB_NAMES
# plus the project namespace and Handle).
KERNEL_NAMES = frozenset({"discover", "recall", "guide", "validate", "capture", "ai", "agent", "gather", "describe",
                          "reachable", "lib", "tasks", "learn", "vrooli", "__vrooli__", "Handle"})
TYPES = {"string": (str,), "boolean": (bool,), "array": (list,), "object": (dict,)}


def kernel_names(source: str) -> list[str]:
    """Kernel-only names the source reads and never defines itself."""
    tree = ast.parse(source)
    defined, used = set(), set()
    for node in ast.walk(tree):
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
            defined.add(node.name)
        elif isinstance(node, ast.Name):
            (defined if isinstance(node.ctx, ast.Store) else used).add(node.id)
        elif isinstance(node, (ast.Import, ast.ImportFrom)):
            defined.update((alias.asname or alias.name).split(".")[0] for alias in node.names)
        elif isinstance(node, ast.arg):
            defined.add(node.arg)
    return sorted((used - defined) & KERNEL_NAMES)


def type_ok(value, kind: str) -> bool:
    if kind in ("number", "integer"):
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            return False
        return kind == "number" or (math.isfinite(value) and float(value).is_integer())
    return isinstance(value, TYPES.get(kind, ()))


def resolve_inputs(declared: dict, provided: dict) -> dict:
    """Mirror contracts.ResolveInputs: the kernel's admission checks."""
    unknown = sorted(set(provided) - set(declared))
    if unknown:
        raise ValueError(f"unknown input {unknown[0]!r}")
    resolved = {}
    for name, spec in declared.items():
        if name in provided:
            value = provided[name]
        elif "default" in spec:
            value = spec["default"]
        elif spec.get("required"):
            raise ValueError(f"missing required input {name!r}")
        else:
            continue
        if not type_ok(value, spec.get("type", "")):
            raise ValueError(f"input {name!r} must have type {spec.get('type')}")
        if spec.get("enum") and value not in spec["enum"]:
            raise ValueError(f"input {name!r} is outside its declared enum")
        resolved[name] = value
    return resolved


def parse_args(argv: list[str]) -> tuple[str, dict]:
    if not argv or argv[0].startswith("-"):
        raise ValueError("usage: standalone.py <contract.json> [key=value ...] [--inputs-json '{...}']")
    provided, rest = {}, argv[1:]
    while rest:
        arg = rest.pop(0)
        if arg == "--inputs-json":
            provided.update(json.loads(rest.pop(0)))
            continue
        key, sep, raw = arg.partition("=")
        if not sep or not key:
            raise ValueError(f"input must be key=value, got {arg!r}")
        try:
            provided[key] = json.loads(raw)
        except ValueError:
            provided[key] = raw
    return argv[0], provided


def refusal(program: str, detail: str) -> dict:
    return {"program": program, "version": "", "status": "unavailable", "phase": "validate", "inputs": {}, "signals": {},
            "errors": [{"class": "needs_program_runtime", "detail": detail, "where": "validate"}], "evidence": []}


def last_envelope(text: str) -> dict:
    """The envelope a run reported: its last JSON object line."""
    for line in reversed(text.strip().splitlines()):
        try:
            value = json.loads(line)
        except ValueError:
            continue
        if isinstance(value, dict):
            return value
    return {}


def execute(source_path: str, inputs: dict | None = None, contract: dict | None = None, **bindings) -> tuple[str, dict]:
    """Run a program's source with the kernel's own `program` helper; return its stdout and envelope.

    `contract` defaults to the JSON beside the source. Extra keyword arguments
    are bound as globals, which is how tests stand in for governed bindings.
    """
    if contract is None:
        contract_path = os.path.splitext(source_path)[0] + ".json"
        contract = {}
        if os.path.exists(contract_path):
            with open(contract_path, encoding="utf-8") as handle:
                contract = json.load(handle)
    version = str((contract.get("helpers") or {}).get("program") or "1")
    if version not in SUPPORTED_HELPER_VERSIONS:
        raise ValueError(f"program helper version {version!r} is not supported")
    helper = ProgramHelper(version, contract.get("outputs") if isinstance(contract.get("outputs"), dict) else {})
    environment = {"__name__": "<program>", "__builtins__": builtins, **bindings, "program": helper, "inputs": dict(inputs or {})}
    helper._bind(environment)
    with open(source_path, encoding="utf-8") as handle:
        source = handle.read()
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(source, source_path, "exec"), environment)  # noqa: S102 - a declared local program, the kernel's own model
    text = output.getvalue()
    return text, last_envelope(text)


def run(argv: list[str]) -> int:
    contract_path, provided = parse_args(argv)
    with open(contract_path, encoding="utf-8") as handle:
        contract = json.load(handle)
    source_path = os.path.splitext(contract_path)[0] + ".py"
    with open(source_path, encoding="utf-8") as handle:
        source = handle.read()
    needs = [binding.get("id", "?") for binding in contract.get("bindings", [])] + kernel_names(source)
    if needs:
        detail = f"needs Program Runtime for {', '.join(needs)}; read {source_path} and do its steps by hand"
        print(json.dumps(refusal(str(contract.get("name", "")), detail)))
        return 3
    text, envelope = execute(source_path, resolve_inputs(contract.get("inputs") or {}, provided), contract)
    sys.stdout.write(text)
    return 0 if envelope.get("status") in ("ok", "partial") else 1


if __name__ == "__main__":
    try:
        sys.exit(run(sys.argv[1:]))
    except (OSError, ValueError, SyntaxError) as exc:
        print(f"standalone: {exc}", file=sys.stderr)
        sys.exit(2)
