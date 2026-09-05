"""Warm JSONL worker for one-shot-compatible sidecar modules.

The Go provider sends one JSON object per line:
  {"argv":["-m","image_tools_sidecar.depth","--model","...","--image","...","--out","..."]}

The worker invokes the same module main functions the one-shot path uses. Because
the process stays alive, _common.make_session() keeps ONNX sessions cached across
requests for the same model path.
"""

from __future__ import annotations

import importlib
import json
import sys
import traceback


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
        main()
    finally:
        sys.argv = old_argv


def main() -> None:
    for line in sys.stdin:
        if not line.strip():
            continue
        try:
            payload = json.loads(line)
            _run(payload["argv"])
            response = {"ok": True}
        except SystemExit as exc:
            response = {"ok": False, "error": f"sidecar exited with code {exc.code}"}
        except Exception as exc:  # noqa: BLE001 - returned to Go as backend detail
            response = {
                "ok": False,
                "error": str(exc),
                "trace": traceback.format_exc(limit=6),
            }
        sys.stdout.write(json.dumps(response, separators=(",", ":")) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
