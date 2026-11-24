import contextlib
import io
import json
import runpy
import unittest
from pathlib import Path
from types import SimpleNamespace

ROOT = Path(__file__).resolve().parents[1]
REPO_ROOT = ROOT.parents[3]


class Handle:
    def __init__(self, rows=None, meta=None):
        self.rows = rows or []
        self.metadata = meta or {}

    def head(self, limit):
        return self.rows[:limit]

    def count(self):
        return len(self.rows)

    def meta(self):
        return self.metadata


class ProgramStub:
    def __init__(self, inputs):
        self._inputs = inputs

    def inputs(self):
        return self._inputs

    def envelope(self, name, version, phase, inputs):
        return {"program": name, "version": version, "status": "failed", "phase": phase,
                "inputs": inputs, "signals": {}, "errors": [], "evidence": []}

    def fail(self, status, error_class, detail, phase):
        raise AssertionError((status, error_class, detail, phase))

    def run(self, states, state):
        while state:
            state = states[state]()


def execute(source, inputs, owner):
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        runpy.run_path(str(ROOT / source), init_globals={"program": ProgramStub(inputs), **owner})
    return json.loads(output.getvalue())


class PlanProgramTests(unittest.TestCase):
    def test_page_continuation_uses_total_offset_and_page_count(self):
        # A stale owner hasMore=false must not terminate a page whose total proves more rows exist.
        handle = Handle(
            [{"id": "p1", "slug": "one", "status": "active", "updatedAt": "t1"},
             {"id": "p2", "slug": "two", "status": "draft", "updatedAt": "t2"}],
            {"total": 5, "hasMore": False},
        )
        manager = SimpleNamespace(plans=SimpleNamespace(list=lambda **kwargs: handle))
        result = execute("plan-inventory.py", {"limit": 2, "offset": 0}, {"plan_manager": manager})
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["count"], 2)
        self.assertEqual(result["signals"]["total"], 5)
        self.assertTrue(result["signals"]["has_more"])

    def test_second_page_is_reachable_and_stale_has_more_is_ignored(self):
        calls = []
        def list_page(**kwargs):
            calls.append(kwargs)
            return Handle(
                [{"id": "p3", "slug": "three", "status": "active", "updatedAt": "t3"},
                 {"id": "p4", "slug": "four", "status": "draft", "updatedAt": "t4"}],
                {"total": 5, "hasMore": False},
            )
        manager = SimpleNamespace(plans=SimpleNamespace(list=list_page))
        result = execute("plan-inventory.py", {"limit": 2, "offset": 2}, {"plan_manager": manager})
        self.assertEqual(calls, [{"include_archived": False, "limit": 2, "offset": 2}])
        self.assertEqual([row["id"] for row in result["signals"]["plans"]], ["p3", "p4"])
        self.assertEqual(result["signals"]["offset"], 2)
        self.assertTrue(result["signals"]["has_more"])

    def test_list_binding_manifest_exposes_pagination_fields(self):
        manifest = json.loads((REPO_ROOT / "scenarios/plan-manager/cli/manifest.json").read_text())
        command = next(command for group in manifest["groups"] if group["name"] == "plans"
                       for command in group["commands"] if command["name"] == "list")
        self.assertEqual(command["binding"]["method"], "ListPlans")
        self.assertTrue({"limit", "offset"}.issubset({flag["name"] for flag in command["flags"]}))

    def test_plan_read_returns_phase_index_without_authored_prose(self):
        record = {"id": "p1", "slug": "one", "title": "Safe title", "status": "active",
                  "updatedAt": "t1", "contentHash": "hash", "problemStatement": "private body",
                  "technicalApproach": "private approach", "phases": [{"id": "ph1", "title": "Build",
                  "status": "active", "steps": [{"title": "private step"}]}]}
        handle = Handle(meta={"plan": record})
        manager = SimpleNamespace(plans=SimpleNamespace(get=lambda **kwargs: handle))
        result = execute("plan-read.py", {"id": "p1"}, {"plan_manager": manager})
        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["signals"]["phase_count"], 1)
        self.assertEqual(result["signals"]["phases"][0]["step_count"], 1)
        self.assertNotIn("problemStatement", result["signals"]["plan"])
        self.assertNotIn("technicalApproach", result["signals"]["plan"])
        self.assertNotIn("steps", result["signals"]["phases"][0])


if __name__ == "__main__":
    unittest.main()
