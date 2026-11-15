import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


SCRIPT = Path(__file__).with_name("clean_node_rehearsal.py")
spec = importlib.util.spec_from_file_location("clean_node_rehearsal", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class CleanNodeRehearsalTests(unittest.TestCase):
    def setup_case(self, root):
        escrow = root / "credential-recovery-receipt.json"
        escrow.write_text(json.dumps({"status": "verified", "source_generation": "gen-7"}))
        manifest = root / "manifest.json"
        manifest.write_text(json.dumps({
            "schema_version": 1,
            "rehearsal_id": "node-2-drill-1",
            "source_generation": "gen-7",
            "credential_recovery_receipt": {
                "path": str(escrow),
                "sha256": hashlib.sha256(escrow.read_bytes()).hexdigest(),
            },
            "destinations": [{
                "destination_id": "offhost-a", "name": "offsite", "backend": "filesystem",
                "location": str(root / "bundle"), "created_at": "2026-01-01T00:00:00Z",
                "cap_bytes": 0, "cap_policy": "alert-block",
            }],
            "targets": [{
                "owner": "example-scenario", "name": "durable-state", "kind": "filesystem",
                "locator": "data", "destination_id": "offhost-a",
                "snapshot_id": "snapshot-a", "restore_location": str(root / "restored"),
            }],
            "scenarios": ["example-scenario"],
        }))
        return manifest, escrow

    def test_restores_then_requires_healthy_runtime_and_writes_receipt(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest, _ = self.setup_case(root)
            calls = []

            def fake_run(argv, **_kwargs):
                calls.append(argv)
                if argv[:3] == ["data-backup-manager", "destinations", "import-existing"]:
                    stdout = json.dumps({"destination": {"id": "offhost-a"}})
                elif argv[:3] == ["data-backup-manager", "targets", "register"]:
                    stdout = json.dumps({"target": {"id": "new-target-id"}})
                elif argv[:3] == ["data-backup-manager", "restores", "verify"]:
                    stdout = json.dumps({"restore": {
                        "id": "verify-record", "target_id": "new-target-id", "destination_id": "offhost-a",
                        "snapshot_id": "snapshot-a", "status": "RESTORE_STATUS_VERIFIED", "checksum": "a" * 64,
                    }})
                elif argv[:3] == ["data-backup-manager", "restores", "restore"]:
                    stdout = json.dumps({"restore": {
                        "id": "restore-record", "target_id": "new-target-id", "destination_id": "offhost-a",
                        "snapshot_id": "snapshot-a", "status": "RESTORE_STATUS_RESTORED",
                    }})
                elif argv[:3] == ["vrooli", "scenario", "status"]:
                    stdout = json.dumps({"success": True, "scenario": {"status": "running", "health_status": "healthy"}})
                else:
                    stdout = "{}"
                return module.subprocess.CompletedProcess(argv, 0, stdout, "")

            with patch.object(module, "run", side_effect=fake_run):
                receipt = module.execute(manifest, root / "result.json")
            self.assertEqual(receipt["status"], "verified")
            self.assertEqual(receipt["destinations"][0]["status"], "attached")
            self.assertEqual(receipt["targets"][0]["status"], "restored")
            self.assertEqual(receipt["targets"][0]["verification"]["checksum_sha256"], "a" * 64)
            self.assertEqual(receipt["targets"][0]["restore"]["restore_id"], "restore-record")
            self.assertEqual(receipt["scenarios"][0]["health_status"], "healthy")
            self.assertEqual(calls[0][0:3], ["data-backup-manager", "destinations", "import-existing"])
            self.assertEqual(calls[0].count("--created-at"), 1)
            self.assertEqual(calls[0].count("--snapshot-ids"), 1)
            self.assertEqual(calls[1][0:3], ["data-backup-manager", "targets", "register"])
            self.assertEqual(calls[2][0:3], ["data-backup-manager", "restores", "verify"])
            self.assertEqual(calls[3][0:3], ["data-backup-manager", "restores", "restore"])
            self.assertIn("new-target-id", calls[2])
            self.assertIn("new-target-id", calls[3])
            self.assertEqual(calls[-1][0:4], ["vrooli", "scenario", "status", "example-scenario"])
            self.assertEqual((root / "result.json").stat().st_mode & 0o777, 0o600)

    def test_rejects_tampered_escrow_receipt_before_any_restore(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest, escrow = self.setup_case(root)
            escrow.write_text('{"status":"verified","source_generation":"other"}')
            with patch.object(module, "run") as fake_run:
                with self.assertRaisesRegex(ValueError, "digest does not match"):
                    module.execute(manifest, root / "result.json")
                fake_run.assert_not_called()

    def test_rejects_unhealthy_scenario(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest, _ = self.setup_case(root)

            def fake_run(argv, **_kwargs):
                if argv[:3] == ["data-backup-manager", "destinations", "import-existing"]:
                    stdout = json.dumps({"destination": {"id": "offhost-a"}})
                elif argv[:3] == ["data-backup-manager", "targets", "register"]:
                    stdout = json.dumps({"target": {"id": "new-target-id"}})
                elif argv[:3] == ["data-backup-manager", "restores", "verify"]:
                    stdout = json.dumps({"restore": {
                        "id": "verify-record", "target_id": "new-target-id", "destination_id": "offhost-a",
                        "snapshot_id": "snapshot-a", "status": "RESTORE_STATUS_VERIFIED", "checksum": "a" * 64,
                    }})
                elif argv[:3] == ["data-backup-manager", "restores", "restore"]:
                    stdout = json.dumps({"restore": {
                        "id": "restore-record", "target_id": "new-target-id", "destination_id": "offhost-a",
                        "snapshot_id": "snapshot-a", "status": "RESTORE_STATUS_RESTORED",
                    }})
                elif argv[:3] == ["vrooli", "scenario", "status"]:
                    stdout = json.dumps({"success": True, "scenario": {"status": "running", "health_status": "degraded"}})
                else:
                    stdout = "{}"
                return module.subprocess.CompletedProcess(argv, 0, stdout, "")

            with patch.object(module, "run", side_effect=fake_run):
                with self.assertRaisesRegex(ValueError, "not running and healthy"):
                    module.execute(manifest, root / "result.json")
            result = json.loads((root / "result.json").read_text())
            self.assertEqual(result["status"], "failed")

    def test_manifest_rejects_secret_shaped_extension_fields(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest, _ = self.setup_case(root)
            data = json.loads(manifest.read_text())
            data["repository_passphrase"] = "must-not-be-here"
            manifest.write_text(json.dumps(data))
            with self.assertRaisesRegex(ValueError, "unsupported fields"):
                module.load_manifest(manifest)

    def test_manifest_rejects_empty_scenario_health_gate(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest, _ = self.setup_case(root)
            data = json.loads(manifest.read_text())
            data["scenarios"] = []
            manifest.write_text(json.dumps(data))
            with self.assertRaisesRegex(ValueError, "non-empty list of scenario names"):
                module.load_manifest(manifest)

    def test_nonempty_restore_location_is_rejected_before_destination_attach(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest, _ = self.setup_case(root)
            restore_location = root / "restored"
            restore_location.mkdir()
            (restore_location / "keep.txt").write_text("existing data")
            with patch.object(module, "run") as fake_run:
                with self.assertRaisesRegex(ValueError, "not empty"):
                    module.execute(manifest, root / "result.json")
                fake_run.assert_not_called()

    def test_terminal_failed_verify_is_not_accepted_on_zero_cli_exit(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest, _ = self.setup_case(root)
            calls = []

            def fake_run(argv, **_kwargs):
                calls.append(argv)
                if argv[:3] == ["data-backup-manager", "destinations", "import-existing"]:
                    stdout = json.dumps({"destination": {"id": "offhost-a"}})
                elif argv[:3] == ["data-backup-manager", "targets", "register"]:
                    stdout = json.dumps({"target": {"id": "new-target-id"}})
                elif argv[:3] == ["data-backup-manager", "restores", "verify"]:
                    stdout = json.dumps({"restore": {
                        "id": "verify-record", "target_id": "new-target-id", "destination_id": "offhost-a",
                        "snapshot_id": "snapshot-a", "status": "RESTORE_STATUS_FAILED", "checksum": "",
                    }})
                else:
                    stdout = "{}"
                return module.subprocess.CompletedProcess(argv, 0, stdout, "")

            receipt_path = root / "failed-receipt.json"
            with patch.object(module, "run", side_effect=fake_run):
                with self.assertRaisesRegex(ValueError, "did not reach RESTORE_STATUS_VERIFIED"):
                    module.execute(manifest, receipt_path)
            self.assertEqual(len(calls), 3, "restore must not run after a terminally failed scratch verify")
            receipt = json.loads(receipt_path.read_text())
            self.assertEqual(receipt["status"], "failed")


if __name__ == "__main__":
    unittest.main()
