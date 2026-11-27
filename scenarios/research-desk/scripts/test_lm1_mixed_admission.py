import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import lm1_mixed_admission as lm1


class MixedAdmissionHarnessTests(unittest.TestCase):
    def test_fixed_caller_denominator_and_shapes(self):
        commands = lm1.caller_commands()
        self.assertEqual(len(commands), 20)
        self.assertEqual(sum(name.startswith("audio-") for name, _ in commands), 10)
        self.assertEqual(sum(name.startswith("search-") for name, _ in commands), 9)
        self.assertEqual(sum(name.startswith("gateway-") for name, _ in commands), 1)
        gateway = dict(commands)["gateway-generate-01"]
        self.assertIn("local-only", gateway)
        self.assertIn("--privacy-class", gateway)
        self.assertIn("generate", gateway)

    def test_image_submission_requires_normal_gpu_and_disk_admission(self):
        command = lm1.image_command(Path("/tmp/fixture.png"), "a" * 32)
        self.assertIn("--no-reclaim", command)
        self.assertIn("--require-gpu", command)
        self.assertIn("--fallback-policy", command)
        self.assertIn("local_only", command)
        self.assertIn("--wait", command)
        self.assertEqual(command.count("--operation-id"), 1)

    def test_success_requires_owner_specific_real_response(self):
        def row(name, value, code=0):
            return {"name": name, "stdout": json.dumps(value), "returncode": code}

        self.assertTrue(lm1.success(row("audio-01", {"text": "hello", "providerTier": "PROVIDER_TIER_LOCAL"})))
        self.assertFalse(lm1.success(row("audio-01", {"text": "hello", "providerTier": "PROVIDER_TIER_REMOTE"})))
        self.assertTrue(lm1.success(row("search-01", {"groups": [{"hits": [{"id": "doc"}]}]})))
        self.assertTrue(lm1.success(row("gateway-generate-01", {"valid": True, "evidence": {"status": "success"}})))
        self.assertFalse(lm1.success(row("gateway-generate-01", {"valid": False, "evidence": {"status": "success"}})))

    def test_framework_identity_digests_bind_exact_declared_inputs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "scripts").mkdir()
            (root / "scripts/producer.py").write_bytes(b"producer source")
            (root / "scripts/contract.json").write_bytes(b'{"schema":"test/v1"}')
            definition = {
                "command": ["python3", "scripts/producer.py", "{output}", "{api_url}"],
                "directory": ".",
                "sources": ["scripts/producer.py", "scripts/contract.json"],
                "contract": "scripts/contract.json",
                "endpoints": {"api_url": "API_PORT"},
                "samples": 1,
                "warmups": 0,
                "p95_max_ms": 600000,
                "timeout_seconds": 540,
                "max_age_seconds": 86400,
                "min_cpu_cores": 4,
                "min_memory_bytes": 17179869184,
            }
            output = root / "receipts" / ("a" * 32) / "producer"
            output.mkdir(parents=True)
            (output.parent / "invocation.json").write_text(json.dumps({
                "definition": definition,
                "command": ["python3", "scripts/producer.py", str(output), "http://127.0.0.1:1"],
                "started_at": "2026-11-25T00:00:00Z",
            }))
            with patch.object(lm1, "SCENARIO", root):
                invocation, producer, contract, config = lm1.workload_identity(output)
            self.assertEqual(invocation["definition"], definition)
            self.assertEqual(len(producer), 64)
            self.assertEqual(contract, lm1.sha((root / "scripts/contract.json").read_bytes()))
            self.assertEqual(len(config), 64)
            (root / "scripts/producer.py").write_bytes(b"changed")
            with patch.object(lm1, "SCENARIO", root):
                self.assertNotEqual(lm1.workload_identity(output)[1], producer)


if __name__ == "__main__":
    unittest.main()
