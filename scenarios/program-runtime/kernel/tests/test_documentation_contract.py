import re
import textwrap
from pathlib import Path

from host.engine import Handle


ROOT = Path(__file__).parents[4]
SKILL = ROOT / "scenarios" / "prompt-manager" / "store" / "skills" / "packs" / "core" / "program-runtime" / "SKILL.md"
EXAMPLES = ROOT / "scenarios" / "program-runtime" / "docs" / "examples"


class DocumentationSurface:
    """Bounded stand-in used to execute documentation syntax without side effects."""

    def __getattr__(self, _name):
        return self

    def __call__(self, *_args, **_kwargs):
        return Handle([{"kind": "documentation", "shape": "documentation", "execution_id": "doc-execution"}])

    def gather(self, *calls, **_kwargs):
        return [call() for call in calls]

    def discover(self, _intent):
        return Handle([{"binding_id": "", "reason": "documentation fixture", "null_verdict": True}])


def documented_programs():
    text = SKILL.read_text(encoding="utf-8")
    fences = re.findall(r"```python\n(.*?)```", text, flags=re.DOTALL)
    return [(f"SKILL.md fence {index}", textwrap.dedent(source)) for index, source in enumerate(fences, 1)] + [
        (path.name, path.read_text(encoding="utf-8"))
        for path in sorted(EXAMPLES.glob("*.py"))
    ]


def test_every_documented_program_compiles_and_executes():
    failures = []
    for label, source in documented_programs():
        try:
            code = compile(source, label, "exec")
            exec(code, {"Handle": Handle, "vrooli": DocumentationSurface(), "__name__": "documentation_test"})
        except Exception as exc:  # noqa: BLE001 - report every broken example together
            failures.append(f"{label}: {type(exc).__name__}: {exc}")
    assert not failures, "\n".join(failures)
