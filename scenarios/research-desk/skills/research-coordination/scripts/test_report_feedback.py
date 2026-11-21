import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))

import packet
import report_feedback


class FeedbackTest(unittest.TestCase):
    def test_packet_preflight_checks_infrastructure_and_syncs_registered_answers(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            scripts = root / "scripts"
            kit = root / "scenarios/notification-hub/skills/notification-hub/kit"
            topic = root / "topics/t"
            for path in (scripts / "research_report.py", scripts / "report_feedback.py", scripts / "evidence_record.py",
                         scripts.parent / "templates/topics/common/reporting/report.json", kit / "schemas/evidence.schema.json",
                         kit / "schemas/research-report.schema.json", kit / "templates/research-report.html",
                         topic / "reports/c.json", topic / "work/README.md"):
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("{}" if path.name == "c.json" else "")
            (topic / "reports/c.json").write_text('{"ask_id":"report-1"}')
            health = subprocess.CompletedProcess([], 0, '{"status":"healthy","readiness":true}', "")
            answer = subprocess.CompletedProcess([], 0, json.dumps({"ask": {"id": "report-1",
                         "answered_at": "today", "note": "prior direction"}}), "")
            with patch.object(packet, "REPO", root), patch.object(packet, "SCRIPTS", scripts), \
                    patch.object(packet.subprocess, "run", side_effect=[health, answer]):
                self.assertEqual(packet.report_preflight(root, ["t"]), [])
            self.assertIn("prior direction", next((topic / "feedback").glob("*.json")).read_text())
            (kit / "templates/research-report.html").unlink()
            with patch.object(packet, "REPO", root), patch.object(packet, "SCRIPTS", scripts), \
                    patch.object(packet.subprocess, "run", side_effect=[health, subprocess.CalledProcessError(1, "get")]):
                errors = packet.report_preflight(root, ["t"])
            self.assertTrue(any("infrastructure missing" in error for error in errors))
            self.assertTrue(any("inaccessible" in error for error in errors))

    def test_sync_preserves_answer_and_flags_application_once(self):
        with tempfile.TemporaryDirectory() as d:
            topic = Path(d)
            (topic / "reports").mkdir()
            (topic / "work").mkdir()
            (topic / "work/README.md").write_text("# Plan\n")
            report_data = topic / "previous-report.json"
            report_data.write_text('{"template":"research-report","meta":{"title":"Old"}}')
            (topic / "reports/c1.json").write_text(json.dumps({"ask_id": "ask-1", "subject_key": "research/topic", "report_data_path": str(report_data)}))
            answer = {"ask": {"id": "ask-1", "answered_at": "2026-11-16T00:00:00Z",
                              "answer_changes": {"notes": {"measure.latency": {"label": "Latency", "text": "Check this", "attachments": []}}},
                              "note": "do more", "answer_images": [{"name": "plan.png", "sha256": "abc"}]}}
            failure = subprocess.CalledProcessError(1, "get")
            ok = subprocess.CompletedProcess([], 0, json.dumps(answer), "")
            with patch.object(report_feedback.subprocess, "run", side_effect=[failure, ok, failure, ok]) as run:
                self.assertEqual(report_feedback.sync(topic), [])
                self.assertEqual(report_feedback.sync(topic), [])
            self.assertIn("--subject-key", run.call_args_list[1].args[0])
            self.assertTrue((topic / "reports/c1.json").is_file())
            migrated = list((topic / "reporting/history").glob("*/receipt.json"))
            self.assertEqual(len(migrated), 1)
            self.assertEqual(json.loads(migrated[0].read_text())["legacy_source_path"], str(topic / "reports/c1.json"))
            self.assertEqual(json.loads(migrated[0].with_name("report.json").read_text())["meta"]["title"], "Old")
            latest = json.loads((topic / "reporting/latest.json").read_text())
            self.assertEqual(latest["path"], str(migrated[0].parent.relative_to(topic)))
            self.assertEqual(latest["created_at"], migrated[0].parent.name[:18])
            saved = next(json.loads(path.read_text()) for path in (topic / "feedback").glob("*.json") if "hub_response" in path.read_text())
            self.assertEqual(saved["hub_response"], answer)
            items = json.loads(next((topic / "feedback").glob("hub-*-items.json")).read_text())["items"]
            self.assertEqual(items[0]["status"], "planned")
            self.assertEqual(items[0]["reason"], "Review and route this item in the next commission.")
            report_feedback.reconcile(topic, items[0]["id"], "applied", "Updated the latency benchmark workstream brief.")
            reconciled = json.loads(next((topic / "feedback").glob("hub-*-items.json")).read_text())["items"][0]
            self.assertEqual((reconciled["status"], reconciled["reason"]), ("applied", "Updated the latency benchmark workstream brief."))
            self.assertEqual((topic / "work/README.md").read_text().count("pending supervisor review"), 1)
            snapshot = packet.feedback_snapshot(topic.parent.parent, [])  # no topics still yields a clear contract
            self.assertIn("not an application receipt", snapshot)

    def test_inaccessible_receipt_is_an_error_and_is_not_no_feedback(self):
        with tempfile.TemporaryDirectory() as d:
            topic = Path(d)
            (topic / "reports").mkdir()
            (topic / "reports/c.json").write_text('{"ask_id":"old"}')
            with patch.object(report_feedback.subprocess, "run", side_effect=subprocess.CalledProcessError(1, "get")):
                errors = report_feedback.sync(topic)
            self.assertIn("inaccessible", errors[0])
            self.assertFalse((topic / "feedback").exists())

    def test_invalid_report_history_is_not_treated_as_empty(self):
        with tempfile.TemporaryDirectory() as d:
            topic = Path(d)
            (topic / "reports").write_text("not a directory")
            self.assertIn("not a directory", report_feedback.sync(topic)[0])

    def test_snapshot_keeps_local_operator_sources(self):
        with tempfile.TemporaryDirectory() as d:
            home = Path(d)
            topic = home / "topics/t"
            for sub in ("feedback", "history", "work"):
                (topic / sub).mkdir(parents=True, exist_ok=True)
            (topic / "feedback/f.md").write_text("keep this concern")
            (topic / "history/OPERATOR-ANSWERS-old.md").write_text("old answer")
            (topic / "work/README.md").write_text("Wanted: new plan")
            (topic / "INBOX.md").write_text("### 2026-01-01 · from operator · answer\nnew answer\n")
            result = packet.feedback_snapshot(home, ["t"])
            for value in ("keep this concern", "old answer", "Wanted: new plan", "new answer"):
                self.assertIn(value, result)


if __name__ == "__main__":
    unittest.main()
