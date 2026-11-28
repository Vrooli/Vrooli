"""Guard the compact, stripped CPython input used by Kale's managed bundle."""
import json
import pathlib
import unittest


RESOURCE = pathlib.Path(__file__).resolve().parents[1] / "resource.json"


class BundleContractTests(unittest.TestCase):
    def test_python_runtime_uses_pinned_stripped_distribution(self):
        manifest = json.loads(RESOURCE.read_text(encoding="utf-8"))
        service = manifest["managed_service"]
        target = next(
            item for item in service["acquisition"]["targets"]
            if item.get("when") == {"os": "linux", "arch": "amd64"}
        )
        runtime = next(step for step in target["compose"] if step["role"] == "runtime")

        self.assertIn("-install_only_stripped.tar.gz", runtime["url"])
        self.assertEqual(
            runtime["sha256"],
            "5acfa3e9ba26b51ae161c83aff278da915b590d22373a424b2ba55b8afe91fcc",
        )
        self.assertEqual(
            service["artifact"]["sha256_by_platform"]["linux-amd64"],
            target["artifact_sha256"],
        )


if __name__ == "__main__":
    unittest.main()
