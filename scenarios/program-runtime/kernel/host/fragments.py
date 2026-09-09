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
    safe = {"__builtins__": {"len": len, "min": min, "max": max, "sum": sum, "str": str, "int": int,
                            "float": float, "bool": bool, "dict": dict, "list": list, "range": range,
                            "sorted": sorted, "enumerate": enumerate, "any": any, "all": all}}
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
