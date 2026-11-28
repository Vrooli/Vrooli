import hashlib
import importlib.util
import json
import math
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).with_name("problem_owner_metrics.py")
SPEC = importlib.util.spec_from_file_location("gct_problem_owner_metrics", SCRIPT)
metrics = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(metrics)


class GCTProblemOwnerMetricsTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.repo = Path(self.temp.name)
        self.rel = Path(metrics.PROBLEMS_PATH)
        self.source = self.repo / self.rel
        self.source.parent.mkdir(parents=True)
        self.body = b"private backlog text\n"
        self.source.write_bytes(self.body)
        self.now = 1_795_274_210.0
        mtime = self.now - 9.1 * 86400
        os.utime(self.source, (mtime, mtime))
        self.digest = hashlib.sha256(self.body).hexdigest()
        self.index = {
            "index_schema_version": 3,
            "_active_problem_rows": [{"path": metrics.PROBLEMS_PATH,
                                      "bytes": len(self.body), "age_days": 9.1}],
            "records": [{"id": f"problems-md:{metrics.PROBLEMS_PATH}",
                         "source": "problems-md", "owner": metrics.OWNER,
                         "state": "open", "fingerprint": self.digest,
                         "evidence": {"path": metrics.PROBLEMS_PATH,
                                      "bytes": len(self.body)}}],
        }

    def tearDown(self):
        self.temp.cleanup()

    def test_matches_canonical_owner_row_and_does_not_emit_source_text(self):
        canonical = metrics._load_ratchet().owner_measurements(self.index)
        result = metrics.measure(self.index, self.repo, now=self.now)
        self.assertEqual(result["measurement"], canonical[metrics.OWNER])
        self.assertEqual(result["measurement"]["max_age_days"], 9.1)
        self.assertEqual(result["source"]["sha256"], self.digest)
        self.assertEqual(result["source"]["age_precision"], "indexed source age in fractional days")
        self.assertEqual(result["source"]["age_semantics"], "PROBLEMS.md modification age; issue dates inside the document are not parsed")
        self.assertEqual(result["closure_claims"], [])
        self.assertNotIn(self.body.decode().strip(), json.dumps(result))

    def test_rejects_stale_bytes_or_fingerprint_instead_of_recording_a_false_window(self):
        self.source.write_bytes(self.body + b"concurrent edit")
        with self.assertRaisesRegex(ValueError, "byte count"):
            metrics.measure(self.index, self.repo)
        self.source.write_bytes(self.body)
        self.index["records"][0]["fingerprint"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "fingerprint"):
            metrics.measure(self.index, self.repo)

    def test_rejects_understated_fractional_age_from_canonical_aggregator(self):
        with patch.object(metrics, "_load_ratchet") as load:
            load.return_value.owner_measurements.return_value = {
                metrics.OWNER: {"eligible_count": 1, "bytes": len(self.body),
                                "max_age_days": 9.0, "age_distribution": {"8-30": 1}}}
            with self.assertRaisesRegex(ValueError, "indexed source age 9.1"):
                metrics.measure(self.index, self.repo)

    def test_rejects_closed_or_missing_pointer(self):
        self.index["records"][0]["state"] = "closed"
        with self.assertRaisesRegex(ValueError, "absent, misowned, or not open"):
            metrics.measure(self.index, self.repo)

    def test_rejects_stale_cached_source_age(self):
        self.index["_active_problem_rows"][0]["age_days"] = 8.0
        with self.assertRaisesRegex(ValueError, "live PROBLEMS file age is 9.1 days"):
            metrics.measure(self.index, self.repo, now=self.now)


if __name__ == "__main__":
    unittest.main()
