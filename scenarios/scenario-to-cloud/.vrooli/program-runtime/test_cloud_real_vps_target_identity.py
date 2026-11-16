import ast
from pathlib import Path
import unittest

RUNNER = Path(__file__).with_name("cloud-real-vps-qualification.py")


def load_validator():
    tree = ast.parse(RUNNER.read_text())
    fn = next(node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "validate_bridge_target")
    namespace = {}
    exec(compile(ast.Module(body=[fn], type_ignores=[]), str(RUNNER), "exec"), namespace)
    return namespace["validate_bridge_target"]


def load_receipt_validator():
    tree = ast.parse(RUNNER.read_text())
    fn = next(node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "validate_target_receipt")
    namespace = {}
    exec(compile(ast.Module(body=[fn], type_ignores=[]), str(RUNNER), "exec"), namespace)
    return namespace["validate_target_receipt"]


def load_qemu_receipt_validator():
    runner = RUNNER.with_name("cloud-qemu-qualification.py")
    tree = ast.parse(runner.read_text())
    fn = next(node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "validate_target_receipt")
    namespace = {}
    exec(compile(ast.Module(body=[fn], type_ignores=[]), str(runner), "exec"), namespace)
    return namespace["validate_target_receipt"]


class BridgeTargetIdentityTests(unittest.TestCase):
    def test_rejects_missing_or_mismatched_identity(self):
        validate = load_validator()
        complete = {"transport": "bridge", "machineId": "disposable-vps:alpha", "nodeId": "node-alpha", "enrollmentGeneration": 7}
        for field, replacement in (("transport", "ssh"), ("nodeId", ""), ("enrollmentGeneration", 0)):
            with self.subTest(field=field):
                target = dict(complete)
                target[field] = replacement
                self.assertIsNotNone(validate(target, "disposable-vps:alpha", 7))
        self.assertIsNotNone(validate(complete, "disposable-vps:alpha", 8))
        self.assertIsNotNone(validate(complete, "disposable-vps:other", 7))

    def test_accepts_complete_staging_vps_identity(self):
        validate = load_validator()
        target = {"transport": "bridge", "machineId": "disposable-vps:alpha", "nodeId": "node-alpha", "enrollmentGeneration": 7}
        self.assertIsNone(validate(target, "disposable-vps:alpha", 7))


class TargetReceiptBindingTests(unittest.TestCase):
    def test_requires_exact_operation_plan_step_and_bridge_identity(self):
        validate = load_receipt_validator()
        row = {
            "deployment_id": "dep-1", "operation_id": "op-1", "plan_digest": "sha256:" + "a" * 64,
            "step": "release.activate",
            "target": {"machine_id": "disposable-vps:alpha", "node_id": "node-alpha", "enrollment_generation": 7, "transport": "bridge"},
            "receipt": {"found": True, "outcome": "succeeded", "fence": 12},
        }
        args = ("dep-1", "op-1", "sha256:" + "a" * 64, "release.activate", "disposable-vps:alpha", "node-alpha", 7)
        self.assertIsNone(validate(row, *args))
        for field, replacement in (("operation_id", "op-2"), ("step", "release.stage"), ("plan_digest", "sha256:" + "b" * 64)):
            with self.subTest(field=field):
                changed = dict(row)
                changed[field] = replacement
                self.assertIsNotNone(validate(changed, *args))
        missing = dict(row)
        missing["receipt"] = {"found": False}
        self.assertIsNotNone(validate(missing, *args))

    def test_qemu_requires_fresh_receipt_with_exact_guest_identity(self):
        validate = load_qemu_receipt_validator()
        target = {"machine_id": "disposable-qemu:alpha", "node_id": "node-alpha", "enrollment_generation": 4, "transport": "bridge"}
        row = {
            "deployment_id": "dep-1", "operation_id": "op-1", "plan_digest": "sha256:" + "c" * 64,
            "step": "release.activate", "target": dict(target),
            "receipt": {"found": True, "outcome": "succeeded", "fence": 8},
        }
        args = ("dep-1", "op-1", "sha256:" + "c" * 64, "release.activate", target)
        self.assertIsNone(validate(row, *args))
        changed = dict(row)
        changed["target"] = dict(target, enrollment_generation=5)
        self.assertIsNotNone(validate(changed, *args))


if __name__ == "__main__":
    unittest.main()
