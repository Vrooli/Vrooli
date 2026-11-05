import ast
import unittest
from pathlib import Path


SOURCE = Path(__file__).parents[1] / "research.py"


class FakeHandle:
    def __init__(self, meta, rows):
        self._meta = meta
        self._rows = rows

    def meta(self):
        return self._meta

    def count(self):
        return len(self._rows)

    def head(self, count):
        return self._rows[:count]


def run_classification(meta, rows):
    module = ast.parse(SOURCE.read_text())
    function = next(
        node for node in module.body
        if isinstance(node, ast.FunctionDef) and node.name == "step_classify"
    )
    envelope = {
        "status": "failed",
        "signals": {
            "ranked": [],
            "ranked_count": 0,
            "corpora_searched": [],
            "corpora_count": 0,
            "providers_hit": {},
            "degraded": None,
            "partial": None,
            "escalate_to_web": False,
        },
        "errors": [],
        "evidence": [],
    }
    namespace = {
        "envelope": envelope,
        "EFFORT": {"fast": (False, 5, 5)},
        "effort": "fast",
        "handles": {"q": FakeHandle(meta, rows)},
        "types": [],
        "hit_row": lambda row: row,
    }
    exec(compile(ast.Module(body=[function], type_ignores=[]), str(SOURCE), "exec"), namespace)
    namespace["step_classify"]()
    return envelope


class SearchHubResearchDegradedResultTest(unittest.TestCase):
    def test_empty_result_from_degraded_provider_is_unavailable_and_never_escalates_as_zero(self):
        report = run_classification(
            {
                "corporaSearched": ["source-ledger.scopes"],
                "degraded": True,
                "partial": False,
                "rerankerLeg": "none",
                "groups": [
                    {
                        "providerId": "source-ledger.scopes",
                        "degraded": True,
                        "note": "provider circuit open after 6 consecutive failures",
                        "hits": [],
                    }
                ],
            },
            [],
        )

        self.assertEqual(report["status"], "unavailable")
        self.assertEqual(report["signals"]["verdict"], "unavailable")
        self.assertTrue(report["signals"]["degraded"])
        self.assertFalse(report["signals"]["escalate_to_web"])
        self.assertEqual(report["errors"][0]["class"], "degraded_leg")
        self.assertIn("source-ledger.scopes", report["errors"][0]["detail"])
        self.assertIn("circuit open", report["errors"][0]["detail"])

    def test_healthy_empty_result_remains_a_zero_result(self):
        report = run_classification(
            {
                "corporaSearched": ["program-runtime.library"],
                "degraded": False,
                "partial": False,
                "rerankerLeg": "none",
                "groups": [],
            },
            [],
        )

        self.assertEqual(report["status"], "ok")
        self.assertEqual(report["signals"]["verdict"], "zero_result")
        self.assertTrue(report["signals"]["escalate_to_web"])

    def test_partial_empty_result_is_unavailable_without_provider_note(self):
        report = run_classification(
            {
                "corporaSearched": ["provider-a"],
                "degraded": False,
                "partial": True,
                "rerankerLeg": "cross-encoder",
                "groups": [],
            },
            [],
        )

        self.assertEqual(report["status"], "unavailable")
        self.assertEqual(report["signals"]["verdict"], "unavailable")
        self.assertFalse(report["signals"]["escalate_to_web"])
        self.assertEqual(report["errors"][0]["class"], "degraded_leg")
        self.assertIn("partial", report["errors"][0]["detail"])


if __name__ == "__main__":
    unittest.main()
