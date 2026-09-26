"""Regressions for tracked, untracked, and selected source inventory metrics."""

import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import refactor_inventory


class InventoryTest(unittest.TestCase):
    def test_include_untracked_reports_status_splits_and_selected_totals(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            scenario = root / "scenarios/browser-automation-studio"
            files = {
                "api/automation/tracked.go": "package automation\n\n// tracked\n",
                "api/automation/tracked_test.go": "package automation\n// test\n",
                "api/automation/untracked.go": "package automation\n\n\n\n// new\n",
                "api/automation/untracked_test.go": "package automation\n\n\n// new test\n",
            }
            for relative, contents in files.items():
                target = scenario / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text(contents)

            tracked_paths = [
                "scenarios/browser-automation-studio/api/automation/tracked.go",
                "scenarios/browser-automation-studio/api/automation/tracked_test.go",
            ]
            subprocess.run(["git", "-C", str(root), "init", "--quiet"], check=True)
            subprocess.run(["git", "-C", str(root), "config", "user.name", "BAS inventory test"], check=True)
            subprocess.run(["git", "-C", str(root), "config", "user.email", "bas-inventory@example.invalid"], check=True)
            subprocess.run(["git", "-C", str(root), "add", "--", *tracked_paths], check=True)
            subprocess.run(["git", "-C", str(root), "commit", "--quiet", "-m", "inventory fixture"], check=True)

            with (
                patch.object(refactor_inventory, "ROOT", root),
                patch.object(refactor_inventory, "SCENARIO", scenario),
            ):
                tracked_only = refactor_inventory.inventory()
                inclusive = refactor_inventory.inventory(include_untracked=True)

        tracked = tracked_only["surfaces"]["api"]
        all_selected = inclusive["surfaces"]["api"]
        self.assertEqual((2, 5), (tracked["tracked_source_files"], tracked["tracked_source_lines"]))
        self.assertEqual((0, 0), (tracked["untracked_source_files"], tracked["untracked_source_lines"]))
        self.assertEqual((2, 5), (tracked["selected_source_files"], tracked["selected_source_lines"]))
        self.assertEqual((1, 3), (tracked["runtime_files"], tracked["runtime_lines"]))

        self.assertEqual((2, 5), (all_selected["tracked_source_files"], all_selected["tracked_source_lines"]))
        self.assertEqual((2, 9), (all_selected["untracked_source_files"], all_selected["untracked_source_lines"]))
        self.assertEqual((4, 14), (all_selected["selected_source_files"], all_selected["selected_source_lines"]))
        self.assertEqual((2, 8), (all_selected["runtime_files"], all_selected["runtime_lines"]))
        self.assertEqual((1, 3), (all_selected["tracked_runtime_files"], all_selected["tracked_runtime_lines"]))
        self.assertEqual((1, 5), (all_selected["untracked_runtime_files"], all_selected["untracked_runtime_lines"]))
        self.assertNotEqual(tracked_only["source_digest_sha256"], inclusive["source_digest_sha256"])


if __name__ == "__main__":
    unittest.main()
