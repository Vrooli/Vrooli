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

- `NameError` and `AttributeError` are never relabelled. A kernel-bound name
  that is missing is a `kernel_runtime` failure, and a program that calls it
  `binding_error` runs convincingly under plain Python and lies about why it
  failed.
- Exactly one envelope is printed on every path. `run` owns the catch that
  guarantees it; `report` refuses to print twice.
"""
from __future__ import annotations

import json
import re
from typing import Any, Callable, Iterable

HELPER_VERSION = "1"

# The public surface of the bound `program` object. The Go preflight validates
# `program.<member>` attribute paths against this exact tuple (see
# `internal/programs/surface.go` and its test), so a typo such as
# `program.clasify` is refused before the program runs.
PUBLIC_MEMBERS = (
    "VERSION",
    "inputs",
    "envelope",
    "attach",
    "current",
    "fail",
    "guarded",
    "classify",
    "run",
    "report",
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
      "no running runtime ports", "connection refused", "dial tcp", "no such host"),
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

    VERSION = HELPER_VERSION
    BindingError = BindingError
    ScenarioUnreachable = ScenarioUnreachable
    Refused = Refused
    InvalidInput = InvalidInput
    AmbiguousResponse = AmbiguousResponse
    DeadlineExceeded = DeadlineExceeded
    RemoteError = RemoteError
    NoGovernedBinding = NoGovernedBinding

    def __init__(self) -> None:
        self._environment: dict[str, Any] | None = None
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

    # -- the helpers the copies used to define -----------------------------

    def fail(self, status: str, klass: str, detail: Any, where: str) -> str:
        """Record one bad path and route to the report phase."""
        envelope = self.current()
        envelope["status"] = status
        envelope.setdefault("errors", []).append({"class": str(klass), "detail": str(detail)[:ERROR_DETAIL_LIMIT], "where": str(where)})
        return "report"

    def guarded(self, call: Callable[[], Any]) -> Callable[[], Any]:
        """Wrap one read so a `gather` over several returns the exception instead of raising."""

        def run() -> Any:
            try:
                return call()
            except Exception as exc:  # noqa: BLE001 - the caller classifies
                return exc

        return run

    def classify(self, exc: BaseException) -> tuple[str, str]:
        """Map a caught exception to `(status, class)`.

        `NameError` and `AttributeError` are re-raised: a missing kernel-bound
        name is a `kernel_runtime` failure, never a transport class.
        """
        if isinstance(exc, (NameError, AttributeError)):
            raise exc
        if isinstance(exc, BindingError):
            return exc.classification()
        return classify_message(str(exc))

    def run(self, states: dict[str, Callable[[], str | None]], start: str = "validate") -> None:
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
                envelope["status"] = "failed"
                envelope.setdefault("errors", []).append({"class": "kernel_runtime", "detail": str(exc)[:ERROR_DETAIL_LIMIT], "where": envelope.get("phase") or state})
                state = "report"

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
