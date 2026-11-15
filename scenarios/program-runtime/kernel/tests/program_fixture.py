"""Run repository programs in tests with the kernel's real `program` helper.

`run` executes a program under .vrooli/program-runtime the way standalone.py
does, with test doubles for its bindings passed as keyword globals. `Child` is
the handle a nested `lib.program_runtime.<name>()` call returns: it carries one
envelope and the child's real output contract, so `program.child_envelope`
validates the fixture exactly as it would validate the real child.
"""
from __future__ import annotations

import json
import sys
from pathlib import Path

KERNEL = Path(__file__).resolve().parents[1]
PROGRAMS = KERNEL.parent / ".vrooli" / "program-runtime"
sys.path.insert(0, str(KERNEL))
from host.standalone import execute  # noqa: E402


def run(name: str, inputs: dict | None = None, **bindings) -> dict:
    """The envelope program-runtime.<name> reports for inputs."""
    return execute(str(PROGRAMS / f"{name}.py"), inputs, **bindings)[1]


class Child:
    """A nested call's result holding one envelope (phase defaults to report)."""

    def __init__(self, name: str, envelope: dict) -> None:
        contract = json.loads((PROGRAMS / f"{name}.json").read_text())
        self.envelope = {"phase": "report", **envelope}
        self.metadata = {"program": f"program-runtime.{name}", "version": contract.get("version", "1"),
                         "digest": "sha256:fixture", "output_contract": contract.get("outputs", {})}

    def head(self, _n: int) -> list:
        return [self.envelope]

    def count(self) -> int:
        return 1

    def meta(self) -> dict:
        return self.metadata
