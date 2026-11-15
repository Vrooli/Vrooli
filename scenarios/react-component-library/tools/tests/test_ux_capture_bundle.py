import hashlib
import json
import tempfile
import unittest
import uuid
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from ux_capture_bundle import copy_response


class BundleArtifactTests(unittest.TestCase):
    def response(self, root: Path, content: bytes) -> dict:
        bundle_id = str(uuid.uuid4())
        bundle_dir = root / "ux" / bundle_id
        (bundle_dir / "pixels").mkdir(parents=True)
        (bundle_dir / "pixels" / "desktop.png").write_bytes(content)
        return {
            "bundle_dir": str(bundle_dir),
            "manifest": {
                "schema_version": "ux-capture.v1",
                "id": bundle_id,
                "privacy": {"redaction_status": "complete"},
                "artifacts": [
                    {
                        "kind": "pixels",
                        "path": "pixels/desktop.png",
                        "size_bytes": len(content),
                        "sha256": hashlib.sha256(content).hexdigest(),
                    }
                ],
            },
        }

    def test_copies_only_a_manifest_verified_artifact(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            target = root / "result.png"
            content = b"png fixture"
            copy_response(self.response(root, content), "pixels", target)
            self.assertEqual(target.read_bytes(), content)

    def test_rejects_artifact_digest_mismatch(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            response = self.response(root, b"png fixture")
            (Path(response["bundle_dir"]) / "pixels" / "desktop.png").write_bytes(b"PNG fixture")
            with self.assertRaisesRegex(ValueError, "digest"):
                copy_response(response, "pixels", root / "result.png")


if __name__ == "__main__":
    unittest.main()
