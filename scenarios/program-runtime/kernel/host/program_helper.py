"""The `program` helper: scaffolding every contract program used to copy.

Before this module existed, `program-contracts.md` told authors to copy a
25-line `classify_transport` table, a `fail` helper, an `inputs` guard, a
`guarded` wrapper and the driver loop into every program verbatim, because a
program is one file and the kernel withholds shared imports. Sixty-one programs
carried the copy, eleven of them had drifted, and every bridge failure was an
untyped `RuntimeError` string the copies classified by substring.

This module is the one copy. The kernel binds an instance as `program` next to
`ai`, `gather` and `lib`; the bridge raises the typed exceptions below instead
of strings; `program.classify` reads the exception's class instead of matching
text. The table in `classify_message` remains for messages that reach a program
without a class (an older bridge body, a reachability probe, a projection verb)
and is the only substring table left in the runtime.

Two invariants the old copies encoded and this module keeps:

- `NameError` stays distinct as `unresolved_name`; no binding alias or retry is
  guessed. `AttributeError` remains a `kernel_runtime` failure.
- Exactly one envelope is printed on every path. `run` owns the catch that
  guarantees it; `report` refuses to print twice.
"""
from __future__ import annotations

import json
import re
import urllib.error
import urllib.request
from typing import Any, Callable, Iterable

from contracts import _json_equal, _matches_type
try:
    from .bridge_transport import open_bridge_request as _open_bridge_request
except ImportError:  # launched by absolute path without the host package on sys.path
    from bridge_transport import open_bridge_request as _open_bridge_request

HELPER_VERSION = "2"
SUPPORTED_HELPER_VERSIONS = {"1", "2"}

# The public surface of the bound `program` object. The Go preflight validates
# `program.<member>` attribute paths against this exact tuple (see
# `internal/programs/surface.go` and its test), so a typo such as
# `program.clasify` is refused before the program runs.
PUBLIC_MEMBERS = (
    "VERSION",
    "inputs",
    "child_envelope",
    "deadline_refusal", "fail_exception",
    "envelope",
    "attach",
    "current",
    "fail",
    "guarded",
    "classify",
    "select_field",
    "coerce_int",
    "coerce_float",
    "run",
    "report",
    "write_artifact",
    "BindingError",
    "ScenarioUnreachable",
    "Refused",
    "InvalidInput",
    "AmbiguousResponse",
    "DeadlineExceeded",
    "RemoteError",
    "NoGovernedBinding",
)

ERROR_DETAIL_LIMIT = 240


class BindingError(RuntimeError):
    """A governed binding call produced no rows.

    Carries the closed `(status, class)` pair a contract envelope records, the
    binding id, and the remote HTTP status when the target scenario answered.
    Subclasses fix the pair; the base class is the generic `binding_error`.
    """

    status = "failed"
    klass = "binding_error"

    def __init__(self, detail: Any, *, binding_id: str = "", status: str | None = None, klass: str | None = None, http_status: int = 0) -> None:
        super().__init__(str(detail))
        self.detail = str(detail)
        self.binding_id = binding_id
        if status:
            self.status = status
        if klass:
            self.klass = klass
        self.http_status = int(http_status or 0)

    def classification(self) -> tuple[str, str]:
        return (self.status, self.klass)

    def as_error(self, where: str = "") -> dict[str, Any]:
        return {"class": self.klass, "detail": self.detail[:ERROR_DETAIL_LIMIT], "where": where}


class ScenarioUnreachable(BindingError):
    """The target scenario, or the bridge to it, could not be reached. Retry later; never read as zero."""

    status = "unavailable"
    klass = "scenario_unreachable"


class Refused(BindingError):
    """Governance stopped the call: no grant, not run-eligible, or a spend ceiling."""

    status = "refused"
    klass = "no_grant"


class InvalidInput(BindingError, TypeError):
    """The arguments do not match the binding's proto request."""

    klass = "invalid_input"


class AmbiguousResponse(BindingError, ValueError):
    """The response has no determinable primary row field; pass `rows=`."""

    klass = "ambiguous_response"


class DeadlineExceeded(BindingError, TimeoutError):
    """A budget or transport deadline ended the call."""

    klass = "deadline_exceeded"


class RemoteError(BindingError):
    """The target scenario answered with an error status. The scenario ran; the outcome is bad."""

    klass = "remote_error"


class NoGovernedBinding(BindingError):
    """The name resolves to no governed binding. Permanent until someone adds it; never `unavailable`."""

    klass = "no_governed_binding"


_EXCEPTION_FOR_CLASS: dict[str, type[BindingError]] = {
    "scenario_unreachable": ScenarioUnreachable,
    "no_grant": Refused,
    "not_run_eligible": Refused,
    "inference_spend_exceeded": Refused,
    "delegated_run_spend_exceeded": Refused,
    "invalid_input": InvalidInput,
    "ambiguous_response": AmbiguousResponse,
    "deadline_exceeded": DeadlineExceeded,
    "remote_error": RemoteError,
    "no_governed_binding": NoGovernedBinding,
    "invalid_response": BindingError,
    "binding_error": BindingError,
}

_REMOTE_STATUS = re.compile(r"remote status (\d{3})")

# Message needles for text that reaches a program without a class. Order
# matters: unreachable first, deadline late, because several needles are
# substrings of ordinary deadline messages. This mirrors the Go bridge table
# in `internal/bindings/failure.go`; the Go side is authoritative for bridge
# bodies and this table only covers kernel-local and legacy messages.
_MESSAGE_TABLE: tuple[tuple[tuple[str, ...], str, str], ...] = (
    (("is unreachable", "bridge unavailable", "bridge is unavailable", "scenario_not_running",
      "no running runtime ports", "connection refused", "dial tcp", "no such host",
      "invalid_port", "invalid vrooli_api_port", "vrooli_not_found"),
     "unavailable", "scenario_unreachable"),
    (("requires an explicit grant", "requires explicit confirmation"), "refused", "no_grant"),
    (("not run eligible", "run_eligible"), "refused", "not_run_eligible"),
    (("inference_spend_exceeded", "inference spend"), "refused", "inference_spend_exceeded"),
    (("delegation_spend_exceeded", "delegated run spend", "delegated_run_spend"), "refused", "delegated_run_spend_exceeded"),
    (("no determinable primary response field", "rows must be one of", "rows must name one of"), "failed", "ambiguous_response"),
    (("accepts named proto fields", "invalid arguments for", "no proto field matches", "decode binding arguments",
      "client-side only", "has an empty proto path", "unknown field"), "failed", "invalid_input"),
    (("is not governed", "does not resolve to a governed binding", "binding does not exist"), "failed", "no_governed_binding"),
    (("deadline", "timed out", "budget exhausted"), "failed", "deadline_exceeded"),
)


def classify_message(text: str) -> tuple[str, str]:
    """Map a bridge message to `(status, class)` when no class travelled with it."""
    lowered = str(text).lower()
    match = _REMOTE_STATUS.search(lowered)
    if match:
        return _remote_classification(int(match.group(1)))
    for needles, status, klass in _MESSAGE_TABLE:
        if any(needle in lowered for needle in needles):
            return (status, klass)
    return ("failed", "binding_error")


def _remote_classification(http_status: int) -> tuple[str, str]:
    # A gateway answer means the scenario is not serving; retry later. Any
    # other status means the scenario ran and its answer is the outcome.
    if http_status in (502, 503, 504):
        return ("unavailable", "scenario_unreachable")
    return ("failed", "remote_error")


def exception_for(klass: str, detail: Any, *, binding_id: str = "", status: str = "", http_status: int = 0) -> BindingError:
    """Build the typed exception for a closed class; an unknown class is the generic `binding_error`."""
    cls = _EXCEPTION_FOR_CLASS.get(str(klass), BindingError)
    resolved_class = str(klass) if str(klass) in _EXCEPTION_FOR_CLASS else BindingError.klass
    return cls(detail, binding_id=binding_id, status=status or None, klass=resolved_class, http_status=http_status)


_OUTPUT_SCHEMA_KEYWORDS = {
    "$schema", "$id", "title", "description", "type", "const", "enum", "required",
    "properties", "additionalProperties", "items", "minItems", "maxItems", "minLength",
    "maxLength", "minimum", "maximum",
}


def validate_output_schema(value: Any, schema: Any) -> list[str]:
    """Validate the JSON Schema subset used by declared program output schemas.

    Unknown schema keywords fail closed. The checked-in Program Runtime output
    schemas use this subset; this avoids a runtime-only third-party dependency.
    """
    issues: list[str] = []

    def visit(candidate: Any, rule: Any, path: str) -> None:
        if not isinstance(rule, dict):
            issues.append(f"schema at {path} is not an object")
            return
        unknown = set(rule) - _OUTPUT_SCHEMA_KEYWORDS
        if unknown:
            issues.append(f"unsupported output schema keyword {sorted(unknown)[0]} at {path}")
            return

        declared_type = rule.get("type")
        if declared_type is not None:
            types = declared_type if isinstance(declared_type, list) else [declared_type]
            if not any(isinstance(kind, str) and _matches_type(candidate, kind) for kind in types):
                issues.append(f"{path} does not match declared type")
                return
        if "const" in rule and not _json_equal(candidate, rule["const"]):
            issues.append(f"{path} does not match declared constant")
        if "enum" in rule and not any(_json_equal(candidate, choice) for choice in rule["enum"]):
            issues.append(f"{path} is outside the declared enum")
        if isinstance(candidate, str):
            if "minLength" in rule and len(candidate) < rule["minLength"]:
                issues.append(f"{path} is shorter than minLength")
            if "maxLength" in rule and len(candidate) > rule["maxLength"]:
                issues.append(f"{path} is longer than maxLength")
        if isinstance(candidate, (int, float)) and not isinstance(candidate, bool):
            if "minimum" in rule and candidate < rule["minimum"]:
                issues.append(f"{path} is below minimum")
            if "maximum" in rule and candidate > rule["maximum"]:
                issues.append(f"{path} is above maximum")
        if isinstance(candidate, list):
            if "minItems" in rule and len(candidate) < rule["minItems"]:
                issues.append(f"{path} has fewer than minItems")
            if "maxItems" in rule and len(candidate) > rule["maxItems"]:
                issues.append(f"{path} has more than maxItems")
            if "items" in rule:
                for index, item in enumerate(candidate):
                    visit(item, rule["items"], f"{path}[{index}]")
        if isinstance(candidate, dict):
            required = rule.get("required", [])
            if not isinstance(required, list) or any(not isinstance(key, str) for key in required):
                issues.append(f"schema required at {path} is not a string array")
                return
            for key in required:
                if key not in candidate:
                    issues.append(f"{path}.{key} is required")
            properties = rule.get("properties", {})
            if not isinstance(properties, dict):
                issues.append(f"schema properties at {path} is not an object")
                return
            for key, child_rule in properties.items():
                if key in candidate:
                    visit(candidate[key], child_rule, f"{path}.{key}")
            additional = rule.get("additionalProperties", True)
            extras = set(candidate) - set(properties)
            if additional is False and extras:
                issues.append(f"{path} contains undeclared property {sorted(extras)[0]}")
            elif isinstance(additional, dict):
                for key in sorted(extras):
                    visit(candidate[key], additional, f"{path}.{key}")

    visit(value, schema, "$")
    return issues


def bridge_error(binding_id: str, transport_status: int, body: Any) -> BindingError:
    """Turn a bridge error response into the typed exception the program catches.

    `body` is the decoded JSON object the Go bridge writes (`error`, `class`,
    `status`, `http_status`) or, from an older bridge, the raw text. A body
    without a class is classified from its message so the program never sees a
    bare string either way.
    """
    payload: dict[str, Any] = body if isinstance(body, dict) else {}
    detail = str(payload.get("error", body if not isinstance(body, dict) else "")).strip() or f"binding {binding_id} failed with status {transport_status}"
    remote_status = int(payload.get("http_status", 0) or 0)
    klass = str(payload.get("class", "")).strip()
    status = str(payload.get("status", "")).strip()
    if not klass:
        status, klass = classify_message(detail)
        if not remote_status:
            match = _REMOTE_STATUS.search(detail.lower())
            remote_status = int(match.group(1)) if match else 0
    return exception_for(klass, detail, binding_id=binding_id, status=status, http_status=remote_status)


def new_envelope(program: str, version: str = "1", *, phase: str = "validate", inputs: dict[str, Any] | None = None) -> dict[str, Any]:
    return {
        "program": str(program),
        "version": str(version),
        "status": "failed",
        "phase": phase,
        "inputs": dict(inputs or {}),
        "signals": {},
        "errors": [],
        "evidence": [],
    }


class ProgramHelper:
    """The object bound as `program` in every program environment.

    One instance per environment: session programs share the session's helper
    across submissions; a nested `lib.<scenario>.<name>()` call gets its own,
    so a child's envelope and report state never leak into the parent.
    """

    BindingError = BindingError
    ScenarioUnreachable = ScenarioUnreachable
    Refused = Refused
    InvalidInput = InvalidInput
    AmbiguousResponse = AmbiguousResponse
    DeadlineExceeded = DeadlineExceeded
    RemoteError = RemoteError
    NoGovernedBinding = NoGovernedBinding

    @property
    def VERSION(self) -> str:
        return self._version

    def __init__(self, version: str = HELPER_VERSION, output_contract: dict[str, Any] | None = None, invocation_context: Any = None) -> None:
        self._version = version
        self._output_contract = dict(output_contract or {})
        self._environment: dict[str, Any] | None = None
        self._invocation_context = invocation_context
        self._envelope: dict[str, Any] | None = None
        self._reported = False

    def _bind(self, environment: dict[str, Any]) -> None:
        self._environment = environment

    def _reset(self) -> None:
        """A session's helper is reused across submissions; each submission reports once."""
        self._envelope = None
        self._reported = False

    # -- inputs and envelope ---------------------------------------------

    def inputs(self, default: dict[str, Any] | None = None) -> dict[str, Any]:
        """The inputs injected by `library run` or a caller's `inputs = {...}` preamble, else `{}`."""
        environment = self._environment or {}
        value = dict.get(environment, "inputs") if isinstance(environment, dict) else None
        if isinstance(value, dict):
            return value
        return dict(default or {})

    def child_envelope(self, handle: Any, *, invalid_class: str = "invalid_response", where: str | None = None, expected_digest: str | None = None, adopt: tuple[str, ...] = (), row_groups: dict[str, list[str]] | None = None) -> dict[str, Any]:
        """Read one nested contract envelope and optionally enforce its pinned artifact digest."""
        if self.VERSION != "2":
            raise RuntimeError("program.child_envelope requires helpers.program version 2")
        if expected_digest is not None and not re.fullmatch(r"[0-9a-f]{64}", expected_digest):
            raise ValueError("expected_digest must be a lowercase SHA-256 artifact digest")
        adoptable = {"inputs", "status", "signals", "errors", "evidence"}
        if not isinstance(adopt, tuple) or any(field not in adoptable for field in adopt):
            raise ValueError("adopt must be a tuple of declared child envelope fields")

        def adopt_child(child: dict[str, Any]) -> dict[str, Any]:
            if row_groups is not None:
                child = dict(child)
                signals = dict(child.get("signals") or {})
                signals["rows"] = [dict(source, reading=dict(source["reading"], cohorts=[
                    dict({"operation": cohort.get("operation"), "context": cohort.get("context")},
                         **{key: cohort.get(key) for key in row_groups[source["row"]]})
                    for cohort in source["reading"].get("cohorts", [])[:1]])
                    if source.get("reading") is not None else source)
                    for source in signals.get("rows", []) if source["row"] in row_groups]
                child["signals"] = signals
            if adopt:
                parent = self.current()
                for field in adopt:
                    parent[field] = child[field]
            return child

        parent_errors = self._enum(self._output_contract, "errors", "classes")
        if invalid_class not in parent_errors:
            raise ValueError(f"{invalid_class!r} must be declared in the parent errors.classes")
        parent_statuses = self._enum(self._output_contract, "status", "enum")
        if "failed" not in parent_statuses:
            raise ValueError("parent status.enum must declare 'failed' to normalize an invalid child envelope")
        if where is None:
            current = self._envelope if isinstance(self._envelope, dict) else None
            if current is None and isinstance(self._environment, dict):
                candidate = dict.get(self._environment, "envelope")
                current = candidate if isinstance(candidate, dict) else None
            phase = current.get("phase") if current else None
            where = phase if isinstance(phase, str) and phase else "collect"

        raw: Any = None
        issues: list[str] = []
        empty_result = False
        try:
            count = handle.count() if callable(getattr(handle, "count", None)) else None
            rows = handle.head(1)
            if count is not None and count != 1:
                issues.append("nested result must contain exactly one envelope")
                empty_result = count == 0 and isinstance(rows, list) and not rows
            if not isinstance(rows, list) or len(rows) != 1:
                issues.append("nested result must contain exactly one envelope")
                empty_result = empty_result or (isinstance(rows, list) and not rows)
            else:
                raw = rows[0]
        except Exception as exc:  # malformed handles are protocol failures
            issues.append(f"could not read nested result: {type(exc).__name__}")

        metadata: dict[str, Any] = {}
        try:
            value = handle.meta()
            if isinstance(value, dict):
                metadata = value
        except Exception:
            pass
        if expected_digest is not None and metadata.get("digest") != expected_digest:
            issues.append("child artifact digest is absent or does not match its declared pin")
        child_contract = metadata.get("output_contract") if isinstance(metadata.get("output_contract"), dict) else {}
        child_statuses = self._enum(child_contract, "status", "enum")
        child_phases = self._enum(child_contract, "phase", "enum")
        child_errors = set(self._enum(child_contract, "errors", "classes"))
        if not child_statuses or not child_phases or not child_errors:
            issues.append("child output contract metadata is missing declared status, phase, or error enums")

        if not isinstance(raw, dict):
            issues.append("child envelope must be an object")
            raw = {}
        if not isinstance(raw.get("status"), str) or (child_statuses and raw.get("status") not in child_statuses):
            issues.append("child status is missing or outside its declared enum")
        if not isinstance(raw.get("phase"), str) or (child_phases and raw.get("phase") not in child_phases):
            issues.append("child phase is missing or outside its declared enum")
        signals = raw.get("signals")
        if not isinstance(signals, dict):
            issues.append("child signals must be an object")
            signals = {}
        raw_errors = raw.get("errors")
        if not isinstance(raw_errors, list):
            issues.append("child errors must be an array")
            raw_errors = []
        clean_errors: list[dict[str, Any]] = []
        for item in raw_errors:
            if not isinstance(item, dict) or not isinstance(item.get("class"), str):
                issues.append("child error item must declare a string class")
                continue
            if child_errors and item["class"] not in child_errors:
                issues.append("child error class is outside its declared enum")
                continue
            if any(key in item and not isinstance(item[key], str) for key in ("detail", "where")):
                issues.append("child error detail and where must be strings")
                continue
            clean_errors.append(dict(item))
        evidence = raw.get("evidence")
        if not isinstance(evidence, list):
            issues.append("child evidence must be an array")
            evidence = []

        if not issues:
            return adopt_child(dict(raw))

        parent_statuses = self._enum(self._output_contract, "status", "enum")
        failure_status = "failed"
        child_program = raw.get("program") if isinstance(raw.get("program"), str) else metadata.get("program", "program-runtime.child")
        child_version = raw.get("version") if isinstance(raw.get("version"), str) else metadata.get("version", "1")
        child_phase = raw.get("phase") if isinstance(raw.get("phase"), str) and (not child_phases or raw.get("phase") in child_phases) else "report"
        empty_class = next((candidate for candidate in ("missing_result", "no_output") if candidate in parent_errors), invalid_class)
        error_class = empty_class if empty_result else invalid_class
        normalized = {
            "program": child_program,
            "version": child_version,
            "status": failure_status,
            "phase": child_phase,
            "inputs": raw.get("inputs") if isinstance(raw.get("inputs"), dict) else {},
            "signals": signals,
            "errors": clean_errors[:4] + [{
                "class": error_class,
                "detail": "; ".join(dict.fromkeys(issues))[:ERROR_DETAIL_LIMIT],
                "where": str(where)[:80],
            }],
            "evidence": evidence[:10],
        }
        return adopt_child(normalized)

    @staticmethod
    def _enum(contract: dict[str, Any], section: str, field: str) -> list[str]:
        value = contract.get(section)
        if not isinstance(value, dict):
            return []
        choices = value.get(field)
        return [item for item in choices if isinstance(item, str)] if isinstance(choices, list) else []

    def envelope(self, program: str, version: str = "1", *, phase: str = "validate", inputs: dict[str, Any] | None = None) -> dict[str, Any]:
        """Create the envelope for this run and remember it for `fail`, `run` and `report`."""
        self._envelope = new_envelope(program, version, phase=phase, inputs=self.inputs() if inputs is None else inputs)
        self._reported = False
        return self._envelope

    def attach(self, envelope: dict[str, Any]) -> dict[str, Any]:
        """Adopt an envelope the program built itself."""
        if not isinstance(envelope, dict):
            raise TypeError("program.attach expects the envelope dict")
        self._envelope = envelope
        return envelope

    def current(self) -> dict[str, Any]:
        if self._envelope is not None:
            return self._envelope
        environment = self._environment or {}
        candidate = dict.get(environment, "envelope") if isinstance(environment, dict) else None
        if isinstance(candidate, dict):
            self._envelope = candidate
            return candidate
        raise RuntimeError("no envelope: call program.envelope(name, version) or program.attach(envelope) before program.fail, program.run or program.report")

    def write_artifact(self, *, scenario: str, run_id: str, artifact_id: str, path: str, overwrite: bool = False) -> dict[str, Any]:
        """Stream one pinned Test Genie artifact into the admitted workspace."""
        if not all(isinstance(value, str) and value.strip() for value in (scenario, run_id, artifact_id, path)):
            raise ValueError("scenario, run_id, artifact_id and relative path are required")
        if type(overwrite) is not bool:
            raise TypeError("overwrite must be a boolean")
        environment = self._environment or {}
        root = dict.get(environment, "__vrooli__") if isinstance(environment, dict) else None
        bridge_url = str(getattr(root, "_bridge_url", "")).strip()
        session_id = str(getattr(root, "_session_id", "")).strip()
        context = self._invocation_context.get({}) if self._invocation_context is not None else {}
        program_id = str(context.get("program_id", "")).strip()
        if not bridge_url or not session_id or not program_id:
            raise RuntimeError("artifact stream effect requires an active governed Program Runtime execution")
        endpoint = bridge_url.rsplit("/bindings/execute", 1)[0] + "/artifacts/stream-to-file"
        request = urllib.request.Request(
            endpoint,
            data=json.dumps({
                "session_id": session_id,
                "program_id": program_id,
                "scenario": scenario.strip(),
                "run_id": run_id.strip(),
                "artifact_id": artifact_id.strip(),
                "path": path,
                "overwrite": overwrite,
            }, separators=(",", ":")).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with _open_bridge_request(request, timeout=120) as response:
                payload = json.loads(response.read(16 << 10).decode())
        except urllib.error.HTTPError as exc:
            detail = exc.read(4096).decode(errors="replace")
            try:
                detail = json.loads(detail).get("error", detail)
            except json.JSONDecodeError:
                pass
            raise RuntimeError(f"artifact stream effect refused: {detail}") from exc
        except (urllib.error.URLError, TimeoutError, OSError, json.JSONDecodeError) as exc:
            raise RuntimeError(f"artifact stream effect unavailable: {exc}") from exc
        if not isinstance(payload, dict) or payload.get("artifact_id") != artifact_id.strip():
            raise RuntimeError("artifact stream effect returned an invalid receipt")
        return payload

    # -- the helpers the copies used to define -----------------------------

    def fail(self, status: str, klass: str, detail: Any, where: str, detail_limit: int = ERROR_DETAIL_LIMIT) -> str:
        """Record one bad path and route to the report phase."""
        if type(detail_limit) is not int:
            raise TypeError("detail_limit must be an integer")
        max_detail = max(0, min(detail_limit, ERROR_DETAIL_LIMIT))
        envelope = self.current()
        envelope["status"] = status
        envelope.setdefault("errors", []).append({"class": str(klass), "detail": str(detail)[:max_detail], "where": str(where)})
        return "report"

    def guarded(self, call: Callable[[], Any]) -> Callable[[], Any]:
        """Wrap one read so a `gather` over several returns the exception instead of raising."""

        def run() -> Any:
            try:
                return call()
            except Exception as exc:  # noqa: BLE001 - the caller classifies
                return exc

        return run

    def deadline_refusal(self, exc: BaseException, *, where: str, detail_limit: int = ERROR_DETAIL_LIMIT) -> dict[str, str]:
        """Return the shared no-retry refusal for an expired owner deadline."""
        prefix = "Deadline expired; no retry or budget extension is automatic. Inspect owner evidence before caller-owned resubmission. Cause: "
        cause = exc.detail if isinstance(exc, BindingError) else str(exc)
        return {"class": "deadline_exceeded", "detail": (prefix + str(cause))[:max(0, min(detail_limit, ERROR_DETAIL_LIMIT))], "where": str(where)[:80]}

    def fail_exception(self, exc: BaseException, where: str, detail_limit: int = ERROR_DETAIL_LIMIT,
                       default_class: str = "binding_error", record: bool = True) -> str | dict[str, str]:
        """Record or return a bounded, no-retry failure response with owner cause."""
        if record and where == "report":
            raise exc
        cause = exc.detail if isinstance(exc, BindingError) else str(exc)
        status, klass, detail = "failed", default_class, cause
        if isinstance(exc, NameError):
            klass = "unresolved_name"
            detail = "Unresolved name; no alias substitution or retry is safe; correct it at its owning program or binding contract. Cause: " + cause
        elif isinstance(exc, AmbiguousResponse):
            klass = "ambiguous_response"
            detail = "Ambiguous response; no row field was guessed or retried. Pass rows=<owner-approved candidate> or ask the binding owner to declare it. Cause: " + cause
        elif isinstance(exc, Refused) and exc.klass == "no_grant":
            status, klass = "refused", "no_grant"
            detail = "No grant was supplied; permission was not inferred and the call was not retried. Ask the binding owner for the required grant. Cause: " + cause
        elif isinstance(exc, ScenarioUnreachable):
            status, klass = "unavailable", "scenario_unreachable"
            detail = "Scenario is unreachable; retain unavailable state and follow its owner's recovery path; no retry was attempted. Cause: " + cause
        elif isinstance(exc, DeadlineExceeded):
            klass = "deadline_exceeded"
            detail = self.deadline_refusal(exc, where=where, detail_limit=detail_limit)["detail"]
        else:
            if default_class == "kernel_runtime":
                status, klass = "failed", "kernel_runtime"
            else:
                try:
                    status, klass = self.classify(exc)
                except (NameError, AttributeError):
                    status, klass = "failed", "kernel_runtime"
        limit = max(0, min(detail_limit, ERROR_DETAIL_LIMIT))
        issue = {"class": klass, "detail": str(detail)[:limit], "where": str(where)}
        if not record:
            return issue
        return self.fail(status, klass, issue["detail"], where, detail_limit)

    def classify(self, exc: BaseException, *, owner: bool = False, include_remote: bool = False,
                 extra_unreachable: tuple[str, ...] = ()) -> tuple[str, str]:
        """Map a caught exception to `(status, class)`.

        `NameError` and `AttributeError` are re-raised: a missing kernel-bound
        name is a `kernel_runtime` failure, never a transport class.
        """
        if isinstance(exc, (NameError, AttributeError)):
            raise exc
        if not owner and isinstance(exc, BindingError):
            return exc.classification()
        if not owner:
            return classify_message(str(exc))
        lowered = str(exc).lower()
        if "lease contention" in lowered or "database is locked" in lowered: return "failed", "lease_contention"
        if "grant" in lowered or "run eligible" in lowered or "run_eligible" in lowered: return "refused", "refused"
        if any(token in lowered for token in ("unreachable", "connection refused", "no running runtime", "bridge unavailable", *extra_unreachable)):
            return "unavailable", "scenario_unreachable"
        if "deadline" in lowered: return "failed", "deadline_exceeded"
        if include_remote and any(token in lowered for token in ("remote", "5xx", "internal server error")):
            return "failed", "remote_error"
        return "failed", "binding_error"

    def select_field(self, row: Any, *names: str, default: Any = None) -> Any:
        """Return the first present, non-empty value among `names` on a mapping.

        Proto/JSON rows reach a program in camelCase or snake_case depending on
        the transport; this is the field access the copies wrote out as a local
        `value()` helper. A non-mapping `row` yields `default`.
        """
        if not isinstance(row, dict):
            return default
        for name in names:
            candidate = dict.get(row, name)
            if candidate is not None and candidate != "":
                return candidate
        return default

    def coerce_int(self, raw: Any, default: int = 0) -> int:
        """Best-effort int, falling back to `default` on None or a bad value."""
        try:
            return int(raw or 0)
        except (TypeError, ValueError):
            return default

    def coerce_float(self, raw: Any, default: float = 0.0) -> float:
        """Best-effort float, falling back to `default` on None or a bad value."""
        try:
            return float(raw or 0.0)
        except (TypeError, ValueError):
            return default

    def run(self, states: dict[str, Callable[[], str | None]], start: str = "validate", *,
            detail_limit: int = ERROR_DETAIL_LIMIT) -> None:
        """Drive the phase state machine with the one catch that guarantees an envelope.

        Returns nothing on purpose: the kernel echoes a program's final
        expression value, so a program ending in `program.run(...)` would
        print the envelope twice if this returned it.
        """
        envelope = self.current()
        state: str | None = start
        while state:
            try:
                step = states[state]
                state = step()
            except Exception as exc:  # noqa: BLE001 - the one catch that guarantees an envelope on every path
                if envelope.get("phase") == "report":
                    raise
                where = envelope.get("phase") or state
                state = self.fail_exception(exc, where, detail_limit=detail_limit, default_class="kernel_runtime")

    def report(self, envelope: dict[str, Any] | None = None) -> None:
        """Print the envelope exactly once as JSON through the environment's `print`."""
        target = self.current() if envelope is None else envelope
        if self._reported:
            raise ValueError("program.report() prints exactly one envelope per run")
        self._reported = True
        printer = print
        environment = self._environment
        if isinstance(environment, dict):
            builtins = dict.get(environment, "__builtins__")
            if isinstance(builtins, dict) and callable(builtins.get("print")):
                printer = builtins["print"]
        printer(json.dumps(target, allow_nan=False))


def public_members(helper: ProgramHelper | None = None) -> Iterable[str]:
    return PUBLIC_MEMBERS
