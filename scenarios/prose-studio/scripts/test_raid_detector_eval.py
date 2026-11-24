"""Synthetic contract tests for raid_detector_eval; never opens public or personal corpora."""
import csv
import hashlib
import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("raid_detector_eval.py")
spec = importlib.util.spec_from_file_location("raid_detector_eval", MODULE_PATH)
evaluator = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(evaluator)


class RaidEvalTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="mp3-raid-eval-fixture-")
        self.root = Path(self.temp.name)
        self.data = self.root / "fixture.csv"
        self.scores = self.root / "scores.csv"
        self.manifest = self.root / "manifest.json"
        self.work = self.root / "scratch"
        self.work.mkdir()
        self.rows = [
            {"id":"c-h","source":"c1","adv":"","model":"human","domain":"blog","attack":"clean"},
            {"id":"c-a","source":"c2","adv":"","model":"model-a","domain":"blog","attack":"clean"},
            {"id":"e-h1","source":"h1","adv":"","model":"human","domain":"blog","attack":"clean"},
            {"id":"e-a1","source":"a1","adv":"","model":"model-a","domain":"blog","attack":"clean"},
            {"id":"e-h2","source":"h2","adv":"","model":"human","domain":"forum","attack":"rewrite"},
            {"id":"e-a2","source":"a2","adv":"","model":"model-a","domain":"forum","attack":"rewrite"},
            {"id":"e-h3","source":"h3","adv":"","model":"human","domain":"forum","attack":"clean"},
            {"id":"e-a3","source":"a3","adv":"","model":"model-a","domain":"blog","attack":"clean"},
        ]
        self.write_data()
        self.write_scores()
        self.write_manifest()

    def tearDown(self):
        self.temp.cleanup()

    def write_data(self):
        with self.data.open("w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=["id","source","adv","model","domain","attack"])
            writer.writeheader()
            writer.writerows(self.rows)

    def write_scores(self, edits=None):
        partitions = {"c-h":"calibration","c-a":"calibration"}
        scores = {"c-h":0.1,"c-a":0.9,"e-h1":0.1,"e-a1":0.9,"e-h2":0.8,"e-a2":0.4,"e-h3":0.3,"e-a3":0.7}
        if edits:
            partitions.update(edits.get("partitions", {}))
            scores.update(edits.get("scores", {}))
            extra = edits.get("extra", [])
        else:
            extra = []
        with self.scores.open("w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=["sample_id","score","partition"])
            writer.writeheader()
            for row in self.rows:
                writer.writerow({"sample_id":row["id"],"score":scores.get(row["id"],0.1),"partition":partitions.get(row["id"],"evaluation")})
            for row in extra:
                writer.writerow(row)

    def write_manifest(self, digest=None):
        manifest = {
            "schema_version": 1,
            "dataset_id": "synthetic/fixture",
            "revision": "a" * 40,
            "split": "train",
            "file_name": self.data.name,
            "file_sha256": digest or hashlib.sha256(self.data.read_bytes()).hexdigest(),
            "score_file_name": self.scores.name,
            "score_file_sha256": hashlib.sha256(self.scores.read_bytes()).hexdigest(),
            "detector":{"id":"synthetic-detector","revision":"fixture-v1"},
            "columns": {"sample_id":"id","source_id":"source","adv_source_id":"adv","model":"model","domain":"domain","attack":"attack"},
            "human_model":"human",
            "ai_models":["model-a"],
            "domains":["blog","forum"],
            "attack_classes":["clean","rewrite"],
            "clean_attack_value":"clean",
            "empty_attack_is_clean":False,
            "threshold":0.5,
            "threshold_provenance":"calibration-set equal-error-rate criterion; locked before evaluation",
            "bootstrap":{"replicates":100,"seed":42},
            "license":{"declared":"MIT","source_url":"https://example.org/dataset-card","per_source_rights":"unresolved"},
        }
        self.manifest.write_text(json.dumps(manifest), encoding="utf-8")

    def run_eval(self):
        return evaluator.evaluate(self.manifest, self.data, self.scores, self.work)

    def assert_rejected(self, phrase):
        with self.assertRaisesRegex(evaluator.EvaluationError, phrase):
            self.run_eval()

    def test_valid_report_has_grouped_metrics_and_uncertainty_without_examples(self):
        report = self.run_eval()
        self.assertEqual(report["evaluation"]["tp"], 2)
        self.assertEqual(report["evaluation"]["fp"], 1)
        self.assertEqual(report["evaluation"]["fn"], 1)
        self.assertEqual(report["evaluation"]["tn"], 2)
        self.assertAlmostEqual(report["evaluation"]["precision"], 2/3)
        self.assertAlmostEqual(report["evaluation"]["recall"], 2/3)
        self.assertIn("blog", report["by_domain"])
        self.assertIn("model-a", report["by_model"])
        self.assertIsNone(report["by_model"]["human"]["recall"])
        self.assertIsNotNone(report["by_model"]["human"]["source_group_bootstrap"]["intervals"]["false_positive_rate"])
        self.assertIn("rewrite", report["by_attack"])
        self.assertEqual(report["overall_source_group_bootstrap"]["replicates"], 100)
        encoded = json.dumps(report)
        for sample in (row["id"] for row in self.rows):
            self.assertNotIn(sample, encoded)

    def test_unknown_model_fails_closed(self):
        self.rows[0]["model"] = "unlisted-model"
        self.write_data(); self.write_manifest()
        self.assert_rejected("unknown model")

    def test_missing_label_fails_closed(self):
        self.rows[0]["model"] = ""
        self.write_data(); self.write_manifest()
        self.assert_rejected("missing sample/source/model/domain")

    def test_missing_attack_requires_explicit_schema_mapping(self):
        self.rows[0]["attack"] = ""
        self.write_data(); self.write_manifest()
        self.assert_rejected("missing attack label")
        doc = json.loads(self.manifest.read_text())
        doc["empty_attack_is_clean"] = True
        self.manifest.write_text(json.dumps(doc))
        report = self.run_eval()
        self.assertIn("clean", report["by_attack"])

    def test_unsupported_attack_fails_closed(self):
        self.rows[2]["attack"] = "unlisted-rewrite"
        self.write_data(); self.write_manifest()
        self.assert_rejected("unsupported attack class")

    def test_source_id_overlap_fails_closed(self):
        self.rows.append({"id":"e-h1-duplicate-source","source":"h1","adv":"","model":"human","domain":"blog","attack":"clean"})
        self.write_data()
        self.write_scores({"partitions":{"e-h1-duplicate-source":"calibration"}})
        self.write_manifest()
        self.assert_rejected("source_id overlaps")

    def test_adv_source_overlap_fails_closed(self):
        self.rows[2]["adv"] = "adv-shared"
        self.rows.append({"id":"c-h-adv","source":"another","adv":"adv-shared","model":"human","domain":"blog","attack":"clean"})
        self.write_data()
        self.write_scores({"partitions":{"c-h-adv":"calibration"}, "scores":{"c-h-adv":0.2}})
        self.write_manifest()
        self.assert_rejected("adv_source_id overlaps")

    def test_dataset_hash_mismatch_fails_closed(self):
        self.write_manifest(digest="0" * 64)
        self.assert_rejected("SHA-256 does not match")

    def test_score_hash_mismatch_fails_closed(self):
        doc = json.loads(self.manifest.read_text())
        doc["score_file_sha256"] = "0" * 64
        self.manifest.write_text(json.dumps(doc))
        self.assert_rejected("score-file SHA-256 does not match")

    def test_unknown_score_id_fails_closed(self):
        self.write_scores({"extra":[{"sample_id":"foreign","score":"0.2","partition":"evaluation"}]})
        self.write_manifest()
        self.assert_rejected("duplicate or unknown sample ID")

    def test_duplicate_sample_id_fails_closed(self):
        self.rows.append(dict(self.rows[0]))
        self.write_data()
        self.write_scores()
        self.write_manifest()
        self.assert_rejected("duplicate sample ID")

    def test_manifest_requires_immutable_revision(self):
        doc = json.loads(self.manifest.read_text())
        doc["revision"] = "main"
        self.manifest.write_text(json.dumps(doc))
        with self.assertRaisesRegex(evaluator.EvaluationError, "immutable"):
            evaluator.load_manifest(self.manifest)


if __name__ == "__main__":
    unittest.main()
