"""Validated, bounded code fragments and portable baseline artifacts."""
import ast
import hashlib
import json
import sys


class StepFailed(RuntimeError):
    def __init__(self, reason, **fields):
        detail = [{"class": item.get("class"), "detail": item.get("detail", "")[:120]} for item in fields.get("errors", []) if isinstance(item, dict)]
        super().__init__(reason + (": " + json.dumps(detail)[:600] if detail else ""))
        self.reason = reason
        self.fields = fields


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()).hexdigest()


def normalize(source):
    if not isinstance(source, str) or len(source.encode()) > 16384:
        raise StepFailed("fragment_size")
    tree = ast.parse(source, "<fragment>", "exec")
    if len(tree.body) != 1 or not isinstance(tree.body[0], ast.FunctionDef):
        raise StepFailed("fragment_shape")
    function = tree.body[0]
    args = function.args
    if (function.name != "step" or [arg.arg for arg in args.args] != ["inputs", "bindings"]
            or args.posonlyargs or args.kwonlyargs or args.vararg or args.kwarg
            or args.defaults or function.decorator_list or function.returns
            or any(arg.annotation for arg in args.args)):
        raise StepFailed("fragment_signature")
    forbidden = (ast.Import, ast.ImportFrom, ast.ClassDef, ast.AsyncFunctionDef, ast.Global,
                 ast.Nonlocal, ast.Lambda, ast.With, ast.AsyncWith, ast.Await, ast.Yield, ast.YieldFrom)
    for node in ast.walk(function):
        if isinstance(node, forbidden) or (isinstance(node, ast.FunctionDef) and node is not function):
            raise StepFailed("fragment_unsafe_syntax")
        if isinstance(node, ast.Attribute) and (node.attr.startswith("_") or node.attr in ("format", "format_map") or isinstance(node.ctx, (ast.Store, ast.Del))):
            raise StepFailed("fragment_private_access")
        if isinstance(node, ast.Name) and node.id.startswith("_"):
            raise StepFailed("fragment_private_access")
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and node.func.id in ("eval", "exec", "getattr", "setattr", "open", "compile", "globals", "locals", "type", "step"):
            raise StepFailed("fragment_unsafe_call")
    return ast.unparse(tree).strip()


def wrap(fragment):
    normalized = normalize(fragment)
    # `isinstance` and `set` are here because normalization fragments — the common
    # Act shape — are defensive by nature. Without them a generated candidate that
    # writes idiomatic Python fails at runtime with NameError, burning an attempt to
    # rediscover the sandbox. Neither widens reach: the fragment can only name types
    # already exposed here, and `type` remains a forbidden call.
    safe = {"__builtins__": {"len": len, "min": min, "max": max, "sum": sum, "str": str, "int": int,
                            "float": float, "bool": bool, "dict": dict, "list": list, "range": range,
                            "sorted": sorted, "enumerate": enumerate, "any": any, "all": all,
                            "isinstance": isinstance, "set": set}}
    safe["StepFailed"] = StepFailed
    exec(compile(normalized, "<fragment>", "exec"), safe, safe)
    function = safe["step"]
    def bounded(inputs, bindings):
        previous = sys.gettrace()
        remaining = 20000
        def trace(frame, event, arg):
            nonlocal remaining
            if frame.f_code.co_filename == "<fragment>" and event in ("line", "call"):
                remaining -= 1
                if remaining <= 0:
                    raise StepFailed("fragment_execution_budget")
            return trace
        try:
            sys.settrace(trace)
            # Generated code cannot rewrite values captured by the independent verifier.
            isolated_inputs = json.loads(json.dumps(inputs, allow_nan=False))
            return function(isolated_inputs, bindings)
        finally:
            sys.settrace(previous)
    return bounded


def baseline_artifact(fragment, compatibility, reviewed_by, evidence, fixtures):
    """Only curated fixtures enter a portable asset; callers never export raw traces."""
    normalized = normalize(fragment)
    if not isinstance(reviewed_by, str) or not reviewed_by.strip() or len(reviewed_by) > 128:
        raise ValueError("baseline requires reviewed_by")
    if not isinstance(evidence, list) or not 1 <= len(evidence) <= 16 or any(not isinstance(item, str) or not item.strip() or len(item) > 256 for item in evidence):
        raise ValueError("baseline requires bounded review evidence")
    if not isinstance(fixtures, list) or not 1 <= len(fixtures) <= 16:
        raise ValueError("baseline requires 1..16 curated fixtures")
    artifact = {"version": 1, "fragment": normalized, "compatibility": compatibility,
                "reviewed_by": reviewed_by, "evidence": evidence, "fixtures": fixtures}
    encoded = json.dumps(artifact, sort_keys=True, allow_nan=False)
    if len(encoded.encode()) > 32768:
        raise ValueError("baseline exceeds 32 KiB")
    # Assets must be portable. This complements review, not a claim of complete secret detection.
    import re
    if re.search(r'(?:/home/|/Users/|[A-Z]:\\\\Users\\\\|-----BEGIN .*PRIVATE KEY|sk-[A-Za-z0-9]{20}|Bearer [A-Za-z0-9._-]{16})', encoded):
        raise ValueError("baseline contains private or machine-specific material")
    for fixture in fixtures:
        if not isinstance(fixture, dict) or (set(fixture) - {"inputs", "expected", "calls"} or not {"inputs", "expected"}.issubset(fixture)) or not isinstance(fixture["inputs"], dict):
            raise ValueError("baseline fixture requires inputs and expected")
        calls = fixture.get("calls", [])
        if not isinstance(calls, list) or len(calls) > 32:
            raise ValueError("baseline fixture calls must be bounded")
        for call in calls:
            if (not isinstance(call, dict) or set(call) - {"binding_id", "arguments", "rows", "metadata"}
                    or not isinstance(call.get("binding_id"), str)
                    or not isinstance(call.get("arguments"), dict)
                    or not isinstance(call.get("rows"), list)):
                raise ValueError("baseline fixture requires recorded binding arguments and rows")
    artifact["digest"] = digest(artifact)
    return artifact


def validate_baseline(artifact, compatibility):
    if not isinstance(artifact, dict):
        raise StepFailed("baseline_invalid")
    rebuilt = baseline_artifact(artifact.get("fragment"), artifact.get("compatibility"),
                                artifact.get("reviewed_by"), artifact.get("evidence"), artifact.get("fixtures"))
    if rebuilt != artifact or artifact["compatibility"] != compatibility:
        raise StepFailed("baseline_incompatible")
    return artifact["fragment"]


class ReplayBindings:
    """Exact ordered recorded binding calls, never live tools during publication."""
    def __init__(self, calls, path="", state=None):
        self._calls, self._path = calls, path
        self._state = state if state is not None else [0]

    def __getattr__(self, name):
        if name.startswith("_"):
            raise AttributeError(name)
        return ReplayBindings(self._calls, (self._path + "/" + name.replace("_", "-")).strip("/"), self._state)

    def __call__(self, **arguments):
        try:
            from .engine import Handle
        except ImportError:
            from engine import Handle
        if self._state[0] >= len(self._calls):
            raise ValueError("unexpected fixture binding call")
        call = self._calls[self._state[0]]
        if call.get("binding_id") != self._path or call.get("arguments", {}) != arguments:
            raise ValueError("fixture binding request mismatch")
        self._state[0] += 1
        return Handle(call.get("rows", []), metadata=call.get("metadata", {}))

    def assert_consumed(self):
        if self._state[0] != len(self._calls):
            raise ValueError("fixture binding calls were not consumed")


def _learning_state_path(owner, provenance, partition_name):
    import os
    from pathlib import Path
    from urllib.parse import urlsplit
    endpoint = getattr(owner, "_bridge_url", "")
    if not endpoint:
        return None
    root = Path(os.environ.get("PROGRAM_RUNTIME_LEARNING_STATE_DIR", "") or
                str(Path(os.environ.get("XDG_STATE_HOME", str(Path.home() / ".local/state"))) / "program-runtime/learning"))
    root.mkdir(mode=0o700, parents=True, exist_ok=True)
    address = urlsplit(endpoint.rsplit("/bindings/", 1)[0])
    runtime_identity = [address.scheme, address.hostname, address.path]
    cohort = "live" if provenance in ("agent", "operator") else provenance
    path = root / (digest([runtime_identity, partition_name, cohort]) + ".sqlite3")
    descriptor = os.open(path, os.O_CREAT | os.O_WRONLY, 0o600)
    os.close(descriptor)
    return path


def learning_quarantine(owner, provenance, step_key, hashes=None):
    """Known negative evidence survives backend outages and program boundaries."""
    import sqlite3
    path = _learning_state_path(owner, provenance, "artifact-quarantine")
    if path is None:
        return []
    with sqlite3.connect(path, timeout=5) as db:
        db.execute("CREATE TABLE IF NOT EXISTS rejected(step_key TEXT, hash TEXT, PRIMARY KEY(step_key,hash))")
        if hashes:
            db.executemany("INSERT OR IGNORE INTO rejected(step_key,hash) VALUES(?,?)", [(step_key, value.removeprefix("sha256:")) for value in hashes if isinstance(value, str) and len(value.removeprefix("sha256:")) == 64])
        return [row[0] for row in db.execute("SELECT hash FROM rejected WHERE step_key=?", (step_key,))]


def durable_learning_write(owner, context, provenance, action, request, send, program_name=""):
    """Write-before-send journal: a lost bridge response is safely replayed later.

    Journals are private to the runtime user and partitioned by endpoint, program
    and provenance. Session credentials are never persisted. Server handlers must
    make receipt/observation identities idempotent.
    """
    import sqlite3
    db_path = _learning_state_path(owner, provenance, "delivery")
    if db_path is None:
        return {"delivery": "unavailable", "reason": "bridge_unavailable"}
    payload = json.dumps(request, sort_keys=True, allow_nan=False)
    if len(payload.encode()) > 256 * 1024:
        raise ValueError("learning event exceeds 256 KiB")
    event_id = digest([action, request])
    with sqlite3.connect(db_path, timeout=5) as db:
        db.execute("CREATE TABLE IF NOT EXISTS events (seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT UNIQUE, action TEXT NOT NULL, payload TEXT NOT NULL)")
        db.execute("CREATE TABLE IF NOT EXISTS rejected_events(id TEXT PRIMARY KEY, action TEXT NOT NULL, payload TEXT NOT NULL, error TEXT NOT NULL)")
        rejected = db.execute("SELECT error FROM rejected_events WHERE id=?", (event_id,)).fetchone()
        if rejected:
            return {"delivery": "rejected", "event_id": event_id, "last_error": rejected[0]}
        count = db.execute("SELECT count(*) FROM events").fetchone()[0]
        if action and count >= 4096 and not db.execute("SELECT 1 FROM events WHERE id=?", (event_id,)).fetchone():
            raise RuntimeError("learning outbox full; evidence not discarded")
        if action:
            db.execute("INSERT OR IGNORE INTO events(id,action,payload) VALUES(?,?,?)", (event_id, action, payload))
        db.commit()
        result = {"delivery": "pending", "event_id": event_id}
        for identity, verb, encoded in db.execute("SELECT id,action,payload FROM events ORDER BY seq LIMIT 128").fetchall():
            try:
                response = send(verb, **json.loads(encoded))
            except Exception as exc:
                error = str(exc)[:160]
                status = getattr(getattr(exc, "__cause__", None), "code", None)
                permanent = status in (400, 403) or (status == 409 and any(marker in str(exc) for marker in (
                    "sql: no rows", "observation identity", "invalid feedback", "bounded feedback", "cohort mismatch")))
                if permanent:
                    # Preserve refused evidence for inspection without poisoning unrelated work.
                    db.execute("INSERT OR REPLACE INTO rejected_events(id,action,payload,error) VALUES(?,?,?,?)", (identity, verb, encoded, error))
                    db.execute("DELETE FROM events WHERE id=?", (identity,))
                    db.commit()
                    if identity == event_id:
                        result = {"delivery": "rejected", "event_id": event_id, "last_error": error}
                    continue
                result["last_error"] = error
                break
            db.execute("DELETE FROM events WHERE id=?", (identity,))
            db.commit()
            if identity == event_id:
                result = {**(response if isinstance(response, dict) else {}), "delivery": "delivered", "event_id": event_id}
        if not action:
            result["delivery"] = "pending" if db.execute("SELECT 1 FROM events LIMIT 1").fetchone() else "delivered"
        return result
