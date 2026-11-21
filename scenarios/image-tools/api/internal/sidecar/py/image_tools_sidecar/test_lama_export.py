"""Safety checks for the closed LaMa export source preparation."""
import tempfile
import hashlib
import unittest
import zipfile
from pathlib import Path

from image_tools_sidecar.lama_export import _load_generator_weights, _prepare_upstream_utils, _safe_extract


class SafeExtractTest(unittest.TestCase):
    def test_extracts_regular_files_and_records_names(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / "source.zip"
            with zipfile.ZipFile(archive, "w") as bundle:
                bundle.writestr("repo/configs/model.yaml", "generator: {}\n")
            output = root / "out"
            self.assertEqual(_safe_extract(archive, output, 100), ["repo/configs/model.yaml"])
            self.assertEqual((output / "repo/configs/model.yaml").read_text(), "generator: {}\n")

    def test_rejects_parent_traversal_absolute_and_size_overflow(self):
        for name, limit, message in (("../escape", 100, "unsafe archive path"),
                                     ("/absolute", 100, "unsafe archive path"),
                                     ("file.bin", 2, "unpacked-size bound")):
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                archive = root / "source.zip"
                with zipfile.ZipFile(archive, "w") as bundle:
                    bundle.writestr(name, "payload")
                with self.assertRaisesRegex(ValueError, message):
                    _safe_extract(archive, root / "out", limit)

    def test_rejects_symlink_members(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / "source.zip"
            info = zipfile.ZipInfo("repo/link")
            info.create_system = 3
            info.external_attr = (0o120777 << 16)
            with zipfile.ZipFile(archive, "w") as bundle:
                bundle.writestr(info, "../../escape")
            with self.assertRaisesRegex(ValueError, "links/special files"):
                _safe_extract(archive, root / "out", 100)

    def test_training_only_import_is_replaced_by_hash_fenced_seed_helper(self):
        original = b"import torch\nfrom pytorch_lightning import seed_everything\n"
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "utils.py"
            path.write_bytes(original)
            original_hash = hashlib.sha256(original).hexdigest()
            prepared_hash = _prepare_upstream_utils(path, original_hash)
            prepared = path.read_text()
            self.assertNotIn("pytorch_lightning", prepared)
            self.assertIn("def seed_everything(seed):", prepared)
            self.assertIn("torch.manual_seed(seed)", prepared)
            self.assertEqual(prepared_hash, hashlib.sha256(path.read_bytes()).hexdigest())
            with self.assertRaisesRegex(ValueError, "source digest changed"):
                _prepare_upstream_utils(path, original_hash)

    def test_safetensors_requires_exact_model_keys_and_tensor_shapes(self):
        import torch
        from safetensors.torch import save_file

        class Model:
            def __init__(self):
                self.expected = {"weight": torch.zeros((2, 3)), "bias": torch.zeros((2,))}
                self.loaded = None
                self.strict = None

            def state_dict(self):
                return self.expected

            def load_state_dict(self, state, strict):
                self.loaded = state
                self.strict = strict

        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "weights.safetensors"
            model = Model()
            save_file({f"model.generator.{key}": value for key, value in model.expected.items()}, str(path))
            _load_generator_weights(model, path)
            self.assertTrue(model.strict)
            self.assertEqual(set(model.loaded), set(model.expected))

            save_file({"model.generator.weight": torch.zeros((3, 2)), "model.generator.bias": torch.zeros((2,))}, str(path))
            with self.assertRaisesRegex(ValueError, "tensor shapes differ"):
                _load_generator_weights(model, path)
            save_file({"model.generator.weight": torch.zeros((2, 3)), "model.generator.bias": torch.zeros((2,)), "model.generator.extra": torch.zeros((1,))}, str(path))
            with self.assertRaisesRegex(ValueError, "keys differ"):
                _load_generator_weights(model, path)
            save_file({"weight": torch.zeros((2, 3)), "bias": torch.zeros((2,))}, str(path))
            with self.assertRaisesRegex(ValueError, "pinned .* wrapper"):
                _load_generator_weights(model, path)


if __name__ == "__main__":
    unittest.main()
