import sys
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

KERNEL = Path(__file__).resolve().parents[3] / "../program-runtime/kernel"
sys.path.insert(0, str(KERNEL.resolve()))
from host.standalone import execute  # noqa: E402

PROGRAM = Path(__file__).resolve().parents[1] / "qualification.py"


class Handle:
    def __init__(self, rows):
        self.rows = rows

    def head(self, count):
        return self.rows[:count]


def invoke(receipt, run_id="run-local-1", error=None, cells=None):
    calls = []

    def read_local(**kwargs):
        calls.append(kwargs)
        if error:
            raise error
        return Handle([receipt] if receipt is not None else [])

    namespace = NS(runs=NS(local_receipt=read_local))
    _, envelope = execute(str(PROGRAM), {
        "request_key": "fixture-request", "candidate_digest": "sha256:" + "0" * 64,
        "target_ids": ["fixture-target"], "authorized_target_ids": ["fixture-target"],
        "authorization_ref": "fixture-grant", "plan_only": True,
        "local_run_ids": [run_id], "evidence_cells": cells or [],
    }, vrooli_bridge=namespace)
    return envelope, calls


class QualificationReceiptConsumerTest(unittest.TestCase):
    def receipt(self, **changes):
        result = {
            "runId": "run-local-1", "lane": "local-contract", "targetClass": "local-fixture",
            "provenance": "local-fixture", "laneEligible": False, "targetId": "fixture-node",
            "scenario": "vrooli-bridge", "verb": "scenario.status", "status": "passed",
            "exitCode": 0, "evidenceRefs": ["bridge-run://run-local-1"],
        }
        result.update(changes)
        return result

    def test_reloads_durable_receipt_and_keeps_local_pass_out_of_remote_lane(self):
        envelope, calls = invoke(self.receipt(), cells=[{"cell_id": "qemu/smoke", "status": "passed"}])
        self.assertEqual(calls, [{"id": "run-local-1"}])
        self.assertEqual(envelope["status"], "ok")
        local = envelope["signals"]["local_receipts"][0]
        self.assertEqual(local["status"], "local_pass_observed")
        self.assertEqual(local["provenance"], "local-fixture")
        self.assertFalse(local["lane_eligible"])
        self.assertFalse(local["qualifies_remote_lane"])
        self.assertEqual(envelope["signals"]["cells"][0]["source"], "caller_observation")
        self.assertFalse(envelope["signals"]["cells"][0]["lane_eligible"])

    def test_missing_and_mismatched_or_promoted_receipts_fail_closed(self):
        cases = [
            (None, "missing"),
            (self.receipt(runId="different-run"), "wrong run id"),
            (self.receipt(laneEligible=True), "promoted lane"),
            (self.receipt(targetClass="qemu"), "wrong target class"),
            (self.receipt(status="running"), "nonterminal status"),
        ]
        for receipt, name in cases:
            with self.subTest(name=name):
                envelope, _ = invoke(receipt)
                self.assertNotEqual(envelope["status"], "ok")
                self.assertNotIn("local_receipts", envelope.get("signals", {}))

    def test_bridge_read_unavailable_is_not_qualification_success(self):
        envelope, calls = invoke(None, error=RuntimeError("scenario_unreachable"))
        self.assertEqual(calls, [{"id": "run-local-1"}])
        self.assertEqual(envelope["status"], "unavailable")


if __name__ == "__main__":
    unittest.main()
