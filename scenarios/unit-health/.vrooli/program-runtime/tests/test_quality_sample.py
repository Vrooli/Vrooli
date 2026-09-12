import contextlib, io, json, unittest
from pathlib import Path

ROOT = Path(__file__).parents[1]

class SampleProgramTests(unittest.TestCase):
    class Program:
        def __init__(self, values):
            self.values = values

        def inputs(self):
            return self.values

    def invoke(self, **changes):
        values = {"scenario":"unit-health", "seed":"seed", "source_identity":"sha256:s", "sample_size":2,
                  "candidates":[{"workspace":"api","file":"a.go","testId":"TestA","framework":"go","testKind":"handler","staticStatus":"violation"},
                                {"workspace":"api","file":"b.go","testId":"TestB","framework":"go","testKind":"handler","staticStatus":"checked_clean"}], **changes}
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "test-quality-sample.py").read_text(), "sample", "exec"), {"program": self.Program(values), "inputs": values})
        return json.loads(stream.getvalue())

    def test_deterministic_bounded_advisory_cohort(self):
        a, b = self.invoke(), self.invoke()
        self.assertEqual(a["status"], "partial")
        self.assertEqual(a["signals"]["cohort"], b["signals"]["cohort"])
        self.assertTrue(a["signals"]["advisory"])
        self.assertEqual(a["signals"]["observations"][0]["ruleVersion"], "test-quality/v1")

    def test_invalid_and_empty_inventory_are_truthful(self):
        self.assertEqual(self.invoke(review_mode="blocking")["status"], "failed")
        self.assertEqual(self.invoke(candidates=[])["status"], "unavailable")
        self.assertEqual(self.invoke(candidates=[{"workspace":"api", "file":"a.go", "testId":"TestA", "framework":"go", "staticStatus":"checked_clean", "sourceDigest":"sha256:other"}])["status"], "unavailable")

    def test_invalid_ai_label_is_not_behavioral(self):
        result = self.invoke(classifications=[{"testIdentity":"api:a.go:TestA:go", "label":"made_up"}])
        row = result["signals"]["observations"][0]
        self.assertEqual(row["status"], "invalid")
        self.assertNotEqual(row["label"], "behavioral")

    def test_ai_path_uses_governed_body_helper_and_sends_capped_excerpt(self):
        class Handle:
            def head(self, _):
                return [{"signals":{"results":[{"index":0,"label":"behavioral"}]}}]
            def raw(self):
                return {"testIdentity":"api:a.go:TestA:go", "bodyExcerpt":"if value != want { t.Fatal(\"bad\") }", "bodyBytes":41, "redactions":0}
        calls = []
        class Gateway:
            def classify_batch(self, **kwargs):
                calls.append(kwargs)
                return Handle()
        class Lib: ai_gateway = Gateway()
        class BodyBinding:
            def test_body(self, **_kwargs):
                return Handle()
        class Validate: test_body = staticmethod(BodyBinding().test_body)
        class UnitHealth: validate = Validate()
        values = {"scenario":"unit-health", "seed":"seed", "source_identity":"sha256:s", "sample_size":1,
                  "review_ai":True, "candidates":[{"workspace":"api","file":"a.go","testId":"TestA","framework":"go","testKind":"unit","staticStatus":"checked_clean"}]}
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "test-quality-sample.py").read_text(), "sample", "exec"), {"program": self.Program(values), "inputs":values, "lib":Lib(), "unit_health":UnitHealth()})
        result = json.loads(stream.getvalue())
        self.assertEqual(result["status"], "ok")
        self.assertEqual(len(calls), 1)
        corpus = json.loads(calls[0]["corpus"][0])
        self.assertEqual(corpus["bodyExcerpt"], 'if value != want { t.Fatal("bad") }')
        self.assertLessEqual(len(corpus["bodyExcerpt"].encode()), 4096)

    def test_refused_body_is_excluded_and_reported_as_insufficient_context(self):
        class Handle:
            def raw(self):
                return {"refused":True, "refusalReason":"privacy_pattern"}
        class BodyBinding:
            def test_body(self, **_kwargs):
                return Handle()
        class Validate: test_body = staticmethod(BodyBinding().test_body)
        class UnitHealth: validate = Validate()
        values = {"scenario":"unit-health", "seed":"seed", "source_identity":"sha256:s", "sample_size":1,
                  "review_ai":True, "candidates":[{"workspace":"api","file":"secret_test.go","testId":"TestSecret","framework":"go","staticStatus":"checked_clean"}]}
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "test-quality-sample.py").read_text(), "sample", "exec"), {"program": self.Program(values), "inputs":values, "unit_health":UnitHealth()})
        result = json.loads(stream.getvalue())
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["review"]["refused_rows"], 1)
        self.assertEqual(result["signals"]["observations"][0]["label"], "insufficient_context")
        self.assertFalse(result["evidence"][-1].get("rows"))

    def test_live_inventory_folds_rule_results_into_one_candidate_per_test(self):
        payload = {"runId": "uh-live", "testQuality": {"results": [
            {"target": {"workspace": "api", "file": "a_test.go", "testId": "TestA"}, "status": "QUALITY_CHECK_STATUS_CHECKED_CLEAN", "supportProfile": "go-syntax-v1", "testKind": "unit"},
            {"target": {"workspace": "api", "file": "a_test.go", "testId": "TestA"}, "status": "QUALITY_CHECK_STATUS_UNKNOWN", "supportProfile": "go-syntax-v1", "testKind": "unit"},
            {"target": {"workspace": "api", "file": "b_test.go", "testId": "TestB"}, "status": "QUALITY_CHECK_STATUS_NOT_APPLICABLE", "supportProfile": "go-syntax-v1"},
            {"target": {"workspace": "ui", "file": "c.test.ts", "testId": "c"}, "status": "QUALITY_CHECK_STATUS_VIOLATION", "supportProfile": "vitest-syntax-1.6.9"},
        ]}}
        calls = []
        def scenario(**kwargs):
            calls.append(kwargs)
            return type("H", (), {"raw": staticmethod(lambda: payload)})()
        values = {"scenario": "unit-health", "seed": "seed", "source_identity": "sha256:s", "sample_size": 4, "workspace": ""}
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "test-quality-sample.py").read_text(), "sample", "exec"),
                 {"program": self.Program(values), "inputs": values, "unit_health": type("U", (), {"validate": type("V", (), {"scenario": staticmethod(scenario)})()})()})
        result = json.loads(stream.getvalue())
        self.assertEqual(calls[0]["rows"], "findings")
        selected = {row["testId"]: row for row in result["signals"]["cohort"]["selected"]}
        self.assertEqual(set(selected), {"TestA", "c"}, "not_applicable rows are not candidates")
        self.assertEqual(selected["TestA"]["staticStatus"], "unknown", "the worst rule status wins")
        self.assertEqual(selected["c"]["framework"], "vitest")
        self.assertEqual(result["signals"]["cohort"]["denominator"], 2)

if __name__ == "__main__": unittest.main()
