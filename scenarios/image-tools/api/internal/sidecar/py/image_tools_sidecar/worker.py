"""Warm JSONL worker for one-shot-compatible sidecar modules.

The Go provider sends one JSON object per line:
  {"argv":["-m","image_tools_sidecar.depth","--model","...","--image","...","--out","..."]}

The worker invokes the same module main functions the one-shot path uses. Because
the process stays alive, ONNX sessions and the bounded SAM2 model/processor stay
resident across requests for the same model.
"""

from __future__ import annotations

import contextlib
import importlib
import io
import json
import sys
import traceback

from . import _common


def _run(argv):
    if len(argv) < 2 or argv[0] != "-m":
        raise ValueError("worker argv must start with -m <module>")
    module_name = argv[1]
    module = importlib.import_module(module_name)
    main = getattr(module, "main", None)
    if main is None:
        raise ValueError(f"module {module_name!r} has no main()")

    old_argv = sys.argv
    try:
        sys.argv = [module_name] + list(argv[2:])
        return main()
    finally:
        sys.argv = old_argv


# How much of a failed module's own output the Go side gets back. The worker's
# stdout is the JSONL protocol channel and its stderr is not read, so a module
# that prints its reason and exits ("this parser has no label for 'book'") would
# otherwise corrupt the protocol or reach the job as "exited with code 1".
_DETAIL_CHARS = 600
# The response kind the Go side maps to a refused-as-given (client fault) failure.
INVALID_INPUT_KIND = "invalid_input"


def _detail(captured: io.StringIO) -> str:
    """The reason a module printed before it failed (structured metrics lines excluded)."""
    lines = [line.strip() for line in captured.getvalue().splitlines() if line.strip()]
    lines = [line for line in lines if not line.startswith("{")]
    return " | ".join(lines)[-_DETAIL_CHARS:]


def main() -> None:
    active_model = None
    for line in sys.stdin:
        if not line.strip():
            continue
        captured = io.StringIO()
        try:
            payload = json.loads(line)
            argv = payload["argv"]
            model = argv[argv.index("--model") + 1] if "--model" in argv else None
            if model != active_model:
                _common._SESSION_CACHE.clear()
                active_model = model
            with contextlib.redirect_stdout(captured), contextlib.redirect_stderr(captured):
                result = _run(argv)
            response = {"ok": True}
            if isinstance(result, dict):
                response["meta"] = {str(key): str(value) for key, value in result.items() if value is not None}
        except SystemExit as exc:
            response = {"ok": False, "error": _detail(captured) or f"sidecar exited with code {exc.code}"}
            if exc.code == _common.EXIT_INVALID_INPUT:
                response["kind"] = INVALID_INPUT_KIND
        except Exception as exc:  # noqa: BLE001 - returned to Go as backend detail
            response = {
                "ok": False,
                "error": str(exc) or _detail(captured),
                "trace": traceback.format_exc(limit=6),
            }
            if isinstance(exc, _common.InputError):
                response["kind"] = INVALID_INPUT_KIND
        sys.stdout.write(json.dumps(response, separators=(",", ":")) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
