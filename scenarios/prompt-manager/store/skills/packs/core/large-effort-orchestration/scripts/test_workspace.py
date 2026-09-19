"""Qualification of portable identity; all mutations use disposable fixtures."""
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import workspace


class WorkspaceIdentityTest(unittest.TestCase):
    def test_new_identity_is_portable_and_reinitialization_preserves_it(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repo = root / "repo"
            (repo / ".vrooli").mkdir(parents=True)
            (repo / ".vrooli/repo-contract.json").write_text(json.dumps({"runtime_home": {
                "dir_name": "runtime", "entries": {"plan_artifacts": {"path": "artifacts"}}
            }}))
            with patch.object(Path, "home", return_value=root):
                folder = workspace.init(repo, "arbitrary-future-effort")
                first = workspace.read_json(folder / "effort.json")
                self.assertTrue(first["effort_ref"].startswith("effort:"))
                self.assertNotIn(str(root), first["effort_ref"])
                self.assertEqual(workspace.inspect(folder)["validation"], "structural_only")
                workspace.init(repo, "arbitrary-future-effort")
                self.assertEqual(first, workspace.read_json(folder / "effort.json"))
                other = workspace.init(repo, "another-effort")
                self.assertNotEqual(first["effort_ref"], workspace.read_json(other / "effort.json")["effort_ref"])
                # Legacy workspaces retain identity behavior instead of being silently migrated.
                first.pop("effort_ref")
                (folder / "effort.json").write_text(json.dumps(first))
                workspace.init(repo, "arbitrary-future-effort")
                self.assertNotIn("effort_ref", workspace.read_json(folder / "effort.json"))
                workspace.inspect(folder)


if __name__ == "__main__":
    unittest.main()
