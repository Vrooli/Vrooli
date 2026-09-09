import contextlib, io, json, unittest
from pathlib import Path

ROOT = Path(__file__).parents[1]

class SampleProgramTests(unittest.TestCase):
    def invoke(self, **changes):
        values = {"scenario":"unit-health", "seed":"seed", "source_identity":"sha256:s", "sample_size":2,
                  "candidates":[{"workspace":"api","file":"a.go","testId":"TestA","framework":"go","testKind":"handler","staticStatus":"violation"},
                                {"workspace":"api","file":"b.go","testId":"TestB","framework":"go","testKind":"handler","staticStatus":"checked_clean"}], **changes}
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "test-quality-sample.py").read_text(), "sample", "exec"), {"inputs": values})
        return json.loads(stream.getvalue())

    def test_deterministic_bounded_advisory_cohort(self):
        a, b = self.invoke(), self.invoke()
        self.assertEqual(a["status"], "partial")
        self.assertEqual(a["signals"]["cohort"], b["signals"]["cohort"])
        self.assertTrue(a["signals"]["advisory"])

    def test_invalid_and_empty_inventory_are_truthful(self):
        self.assertEqual(self.invoke(review_mode="blocking")["status"], "failed")
        self.assertEqual(self.invoke(candidates=[])["status"], "unavailable")
        self.assertEqual(self.invoke(candidates=[{"workspace":"api", "file":"a.go", "testId":"TestA", "framework":"go", "staticStatus":"checked_clean", "sourceDigest":"sha256:other"}])["status"], "unavailable")

    def test_invalid_ai_label_is_not_behavioral(self):
        result = self.invoke(classifications=[{"testIdentity":"api:a.go:TestA:go", "label":"made_up"}])
        row = result["signals"]["observations"][0]
        self.assertEqual(row["status"], "invalid")
        self.assertNotEqual(row["label"], "behavioral")

    def test_ai_path_uses_governed_helper_and_keeps_metadata_only(self):
        class Handle:
            def head(self, _):
                return [{"signals":{"results":[{"index":0,"label":"behavioral"}]}}]
        calls = []
        class Gateway:
            def classify_batch(self, **kwargs):
                calls.append(kwargs)
                return Handle()
        class Lib: ai_gateway = Gateway()
        values = {"scenario":"unit-health", "seed":"seed", "source_identity":"sha256:s", "sample_size":1,
                  "review_ai":True, "candidates":[{"workspace":"api","file":"a.go","testId":"TestA","framework":"go","testKind":"unit","staticStatus":"checked_clean"}]}
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "test-quality-sample.py").read_text(), "sample", "exec"), {"inputs":values, "lib":Lib()})
        result = json.loads(stream.getvalue())
        self.assertEqual(result["status"], "ok")
        self.assertEqual(len(calls), 1)
        self.assertNotIn("source", json.dumps(calls[0]).lower())

    def test_sensitive_candidate_is_refused_before_inference(self):
        result = self.invoke(candidates=[{"workspace":"api", "file":".env", "testId":"TestSecret", "framework":"go", "staticStatus":"checked_clean"}], review_ai=True)
        self.assertEqual(result["status"], "refused")
        self.assertEqual(result["errors"][0]["class"], "privacy_refused")

if __name__ == "__main__": unittest.main()
