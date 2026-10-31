"""Cross-owner declaration parity; no models, network, or GPU are used."""
import hashlib
import json
from pathlib import Path
import unittest


class RegistryContractTests(unittest.TestCase):
    def test_music_declarations_match_owned_catalog_and_artifact(self):
        root = Path(__file__).resolve().parents[3]
        resource = root / "resources/songgen"
        raw = (resource / "server/profiles.json").read_bytes()
        catalog = json.loads(raw)["profiles"]
        manifest = json.loads((resource / "resource.json").read_text())
        self.assertEqual(manifest["managed_service"]["environment"]["SONGGEN_COMPOSED_SHA256"],
                         manifest["managed_service"]["artifact"]["sha256_by_platform"]["linux-amd64"],
                         "served receipt identity must match the lifecycle's selected artifact")
        registry = json.loads((root / "scenarios/music-tools/api/internal/models/registry.seed.json").read_text())
        declared = {m["runtime"]["profile"]: m for m in registry if m.get("runtime", {}).get("resource") == "songgen"}
        self.assertEqual(set(declared), set(catalog))
        for key, profile in catalog.items():
            with self.subTest(profile=key):
                model = declared[key]
                runtime = model["runtime"]
                self.assertEqual(runtime["catalog_sha256"], hashlib.sha256(raw).hexdigest())
                self.assertEqual(runtime["artifact_sha256"], manifest["managed_service"]["artifact"]["sha256_by_platform"]["linux-amd64"])
                self.assertEqual(model["id"], profile["model_id"])
                self.assertEqual(model["revision"], profile["revision"])
                self.assertEqual(model["license"], profile["license"])
                self.assertEqual(runtime["weight_files"], len(profile["files"]))
                self.assertEqual(model["bytes"], sum(f["bytes"] for f in profile["files"]))
                for field in ("max_duration", "default_steps", "min_steps", "max_steps", "default_guidance", "min_guidance", "max_guidance"):
                    self.assertEqual(runtime[field], profile[field], field)

    def test_storage_surface_targets_real_runtime_directories_and_protects_active_jobs(self):
        root = Path(__file__).resolve().parents[3]
        manifest = json.loads((root / "resources/songgen/resource.json").read_text())
        entries = manifest["storage"]["entries"]
        self.assertEqual((entries["models"]["class"], entries["models"]["subpath"]), ("data", "models"))
        self.assertFalse(entries["models"]["regenerable"])
        self.assertEqual((entries["outputs"]["class"], entries["outputs"]["subpath"]), ("data", "outputs"))
        self.assertNotIn("max_bytes", entries["outputs"]["budget"])
        self.assertEqual(entries["outputs"]["budget"]["max_age"], "1d")
        profiles = json.loads((root / "resources/songgen/server/profiles.json").read_text())["profiles"]
        self.assertTrue(all(p["max_execution_seconds"] < 86400 for p in profiles.values()))

    def test_model_groups_are_explicit_and_never_eagerly_reacquired(self):
        root = Path(__file__).resolve().parents[3]
        resource = root / "resources/songgen"
        manifest = json.loads((resource / "resource.json").read_text())
        profiles = json.loads((resource / "server/profiles.json").read_text())["profiles"]
        files = {f["path"]: (key, f["bytes"]) for key, profile in profiles.items() for f in profile["files"]}
        self.assertEqual(manifest["managed_service"]["minimum_data_groups"], 1)
        for artifact in manifest["managed_service"]["data_artifacts"]:
            self.assertEqual(artifact["acquisition_mode"], "on-demand")
            self.assertEqual((artifact["group"], artifact["size_bytes"]), files[artifact["path"]])
