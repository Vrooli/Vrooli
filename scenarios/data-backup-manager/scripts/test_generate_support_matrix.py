import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import generate_support_matrix as matrix


def receipt(**overrides):
    row = {
        "id": "linux-build-one",
        "platform": "linux",
        "architecture": "amd64",
        "host": "runner",
        "observed_at": "2026-11-13T19:21:25Z",
        "capability": "api_cli_build",
        "outcome": "passed",
        "evidence_kind": "native-build",
        "command": "go build",
        "evidence": "temporary build output",
    }
    row.update(overrides)
    return row


class SupportMatrixTests(unittest.TestCase):
    def test_native_receipt_qualifies_only_its_platform_and_capability(self):
        rows = [receipt()]
        self.assertIn("qualified", matrix.render_cell(rows, "linux", "api_cli_build"))
        self.assertEqual("no native receipt", matrix.render_cell(rows, "macos", "api_cli_build"))
        self.assertEqual("no native receipt", matrix.render_cell(rows, "linux", "volume_identity"))

    def test_fixture_and_cross_build_do_not_qualify(self):
        rows = [receipt(evidence_kind="cross-build"), receipt(id="linux-fixture-one", evidence_kind="fixture")]
        self.assertEqual("no native receipt", matrix.render_cell(rows, "linux", "api_cli_build"))

    def test_latest_native_failure_replaces_prior_pass(self):
        rows = [receipt(), receipt(id="linux-build-two", observed_at="2026-11-13T20:00:00Z", outcome="failed")]
        self.assertIn("failed", matrix.render_cell(rows, "linux", "api_cli_build"))

    def test_receipt_document_rejects_timezone_free_timestamps(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "receipts.json"
            path.write_text(json.dumps({"schema_version": 1, "receipts": [receipt(observed_at="2026-11-13T19:21:25")]}), encoding="utf-8")
            with patch.object(matrix, "RECEIPTS", path), self.assertRaisesRegex(ValueError, "timezone"):
                matrix.load_receipts()

    def test_volume_owner_ledger_is_imported_and_validated(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source = root / "receipts.json"
            source.write_text(json.dumps({"schema_version": 1, "receipts": []}), encoding="utf-8")
            ledger = root / "receipts.jsonl"
            ledger.write_text(json.dumps({
                "id": "volume-linux-check-01", "platform": "linux", "architecture": "amd64",
                "host": "host-abc123", "observed_at": "2026-11-13T20:01:02Z",
                "action": "check", "capability": "destination_check", "outcome": "passed",
                "evidence_kind": "native-runtime", "command": "filesystem check <volume>",
                "evidence": "identity_sha256=abc; result_sha256=def",
            }), encoding="utf-8")
            with patch.object(matrix, "RECEIPTS", source), patch.object(matrix, "VOLUME_RECEIPTS", ledger):
                rows = matrix.load_receipts()
            self.assertIn("qualified", matrix.render_cell(rows, "linux", "destination_check"))

    def test_volume_owner_ledger_rejects_invalid_native_rows(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source = root / "receipts.json"
            source.write_text(json.dumps({"schema_version": 1, "receipts": []}), encoding="utf-8")
            ledger = root / "receipts.jsonl"
            ledger.write_text(json.dumps({"id": "broken", "action": "check"}), encoding="utf-8")
            with patch.object(matrix, "RECEIPTS", source), patch.object(matrix, "VOLUME_RECEIPTS", ledger), self.assertRaisesRegex(ValueError, "required"):
                matrix.load_receipts()

    def test_volume_owner_dry_run_stays_unqualified(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source = root / "receipts.json"
            source.write_text(json.dumps({"schema_version": 1, "receipts": []}), encoding="utf-8")
            ledger = root / "receipts.jsonl"
            ledger.write_text(json.dumps({
                "id": "volume-linux-dry-01", "platform": "linux", "architecture": "amd64",
                "host": "host-abc123", "observed_at": "2026-11-13T20:01:02Z",
                "action": "mount_read_write", "capability": "volume_mount", "outcome": "passed",
                "evidence_kind": "fixture", "command": "mount <volume>", "evidence": "dry run",
            }), encoding="utf-8")
            with patch.object(matrix, "RECEIPTS", source), patch.object(matrix, "VOLUME_RECEIPTS", ledger):
                rows = matrix.load_receipts()
            self.assertEqual("no native receipt", matrix.render_cell(rows, "linux", "volume_mount"))

    def test_dbm_runtime_ledger_qualifies_only_actual_backup_and_restore_receipts(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source = root / "receipts.json"
            source.write_text(json.dumps({"schema_version": 1, "receipts": []}), encoding="utf-8")
            volume = root / "volume.jsonl"
            dbm = root / "dbm.jsonl"
            volume.write_text("", encoding="utf-8")
            dbm.write_text("\n".join(json.dumps(receipt(
                id=f"dbm-linux-{capability.replace('_', '-')}", capability=capability, evidence_kind="native-runtime",
                command="DBM native execution", evidence="proof_sha256=" + "a" * 64,
            )) for capability in ("destination_check", "backup", "scratch_restore")), encoding="utf-8")
            with patch.object(matrix, "RECEIPTS", source), patch.object(matrix, "VOLUME_RECEIPTS", volume), patch.object(matrix, "DBM_RECEIPTS", dbm):
                rows = matrix.load_receipts()
            self.assertIn("qualified", matrix.render_cell(rows, "linux", "destination_check"))
            self.assertIn("qualified", matrix.render_cell(rows, "linux", "backup"))
            self.assertIn("qualified", matrix.render_cell(rows, "linux", "scratch_restore"))

    def test_fixture_fault_injection_receipts_never_qualify_native_matrix_cells(self):
        """A simulator may report every capability without making a platform claim."""
        rows = []
        for platform in matrix.PLATFORMS:
            for capability, _ in matrix.CAPABILITIES:
                rows.append(receipt(
                    id=f"{platform}-{capability.replace('_', '-')}-fixture",
                    platform=platform,
                    capability=capability,
                    outcome="passed",
                    evidence_kind="fixture",
                    command="DB6 simulated owner fault injection",
                    evidence="scenario_sha256=" + "b" * 64,
                ))
        for platform in matrix.PLATFORMS:
            for capability, _ in matrix.CAPABILITIES:
                self.assertEqual(
                    "no native receipt",
                    matrix.render_cell(rows, platform, capability),
                    f"fixture receipt qualified {platform}/{capability}",
                )


if __name__ == "__main__":
    unittest.main()
