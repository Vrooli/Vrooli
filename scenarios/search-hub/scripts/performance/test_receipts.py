"""Offline contract tests for SH4 measurement and attribution receipt tools.

Uses only Python's standard library and synthetic local JSON artifacts. It does
not contact Search Hub, AI Gateway, or any serving process.
"""

from __future__ import annotations

import contextlib
import base64
import hashlib
import importlib.util
import io
import json
import sys
import tempfile
import unittest
import math
import random
from pathlib import Path
from unittest import mock


HERE = Path(__file__).resolve().parent


def load_tool(module_name: str, filename: str):
    spec = importlib.util.spec_from_file_location(module_name, HERE / filename)
    module = importlib.util.module_from_spec(spec)
    sys.modules[module_name] = module
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


measure = load_tool("sh4_measure", "measure.py")
scope_receipts = load_tool("scope_receipts", "scope_receipts.py")
run_telemetry = load_tool("run_telemetry", "run_telemetry.py")
stage_telemetry = load_tool("stage_telemetry", "stage_telemetry.py")
compare = load_tool("sh4_compare", "compare.py")
manifest = load_tool("sh4_manifest", "manifest.py")


def probable_prime(rng: random.Random, bits: int) -> int:
    small_primes = (3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41)
    while True:
        candidate = rng.getrandbits(bits) | (1 << (bits - 1)) | 1
        if any(candidate % prime == 0 for prime in small_primes):
            continue
        d, shifts = candidate - 1, 0
        while d % 2 == 0:
            shifts += 1
            d //= 2
        if all(pow(base, d, candidate) in (1, candidate - 1) or any(
                pow(base, d * (2 ** step), candidate) == candidate - 1 for step in range(shifts))
               for base in small_primes):
            return candidate


def test_rsa_keypair() -> tuple[int, int, int]:
    rng = random.Random(0x534834)
    exponent = 65537
    p = probable_prime(rng, 1024)
    q = probable_prime(rng, 1024)
    while p == q or math.gcd(exponent, (p - 1) * (q - 1)) != 1:
        q = probable_prime(rng, 1024)
    modulus = p * q
    return modulus, exponent, pow(exponent, -1, (p - 1) * (q - 1))


def test_sign_payload(payload: dict, signer, modulus: int, private_exponent: int) -> dict:
    message = signer.signing_bytes(payload)
    info = scope_receipts.SHA256_DER_PREFIX + hashlib.sha256(message).digest()
    size = (modulus.bit_length() + 7) // 8
    encoded = b"\x00\x01" + b"\xff" * (size - len(info) - 3) + b"\x00" + info
    signature = pow(int.from_bytes(encoded, "big"), private_exponent, modulus).to_bytes(size, "big")
    return {**payload, "signature_base64": base64.b64encode(signature).decode()}


def write_json(path: Path, value: dict) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, sort_keys=True, default=str) + "\n")
    return path


def sha(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()


class ReceiptContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory(prefix="sh4-receipt-test-")
        self.root = Path(self.temp.name)
        self.next_ms = 1_800_000_000_000

    def tearDown(self) -> None:
        self.temp.cleanup()

    def call_compare(self, base: Path, candidate: Path, base_grade: Path, candidate_grade: Path, output: Path,
                     stage_traces: tuple[Path, Path] | None = None, *, resource_group: str = "api",
                     component_groups: dict[str, str] | None = None) -> None:
        argv = ["compare.py", "--baseline", str(base), "--candidate", str(candidate),
                "--baseline-grades", str(base_grade), "--candidate-grades", str(candidate_grade),
                "--resource-source", "cgroup_groups", "--resource-group", resource_group, "--output", str(output)]
        if stage_traces:
            argv.extend(("--baseline-stage-trace", str(stage_traces[0]), "--candidate-stage-trace", str(stage_traces[1])))
        for role, group in (component_groups or {}).items():
            argv.extend(("--component-group", f"{role}={group}"))
        with mock.patch.object(sys, "argv", argv), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            compare.main()

    def make_grade(self, path: Path, build_id: str, accepted_parts: int = 1, total_parts: int = 2,
                   cohort: str = "cohort-a", query_hash: str | None = None) -> Path:
        return write_json(path, {
            "schema_version": 1, "issuer": "SH1", "receipt_id": path.stem,
            "evidence_uri": f"sh1-eval://{path.stem}", "evidence_sha256": sha(path.stem + "-raw-eval"),
            "cohort_id": cohort,
            "task_set_sha256": sha("frozen-task-set"),
            "acceptance_policy_id": "sh1-part-task-acceptance-v1",
            "run_tag": f"sh1-{build_id}", "evaluation_run_ids": [f"eval-{build_id}"],
            "tier": "federated", "build_id": build_id,
            "cases": {"case-1": {"query_sha256": query_hash or sha("case query"),
                                   "parts": {"accepted": accepted_parts, "total": total_parts,
                                             "verdicts": [True] * accepted_parts + [False] * (total_parts - accepted_parts)},
                                   "tasks": {"accepted": 1, "total": 1, "verdicts": [True]}}},
        })

    def make_pair(self, phase: str, repeat: int, *, candidate_query_hash: str | None = None,
                  candidate_parts_total: int = 2, candidate_cpu: int = 80,
                  candidate_start_offset: int = 4000, resource_group: str = "api",
                  component_groups: dict[str, str] | None = None) -> dict:
        pair_dir = self.root / "pairs" / phase / str(repeat)
        run_id = f"{phase}-{repeat}"
        started = self.next_ms
        self.next_ms += 20_000
        fault = "fault-degraded" if phase == "degraded" else "none"
        group_pattern = [{"name": "api", "comm_regex": "^search-hub-api$"},
                         {"name": "serving_total", "comm_regex": "^search-hub-(api|provider)$"}]

        def report(arm: str, start: int, query_hash: str) -> Path:
            context = {"build_id": f"build-{arm}", "corpus_snapshot_id": "corpus-v1",
                       "fault_schedule_id": fault, "state_receipt_id": f"state-{arm}",
                       "admission_receipt_id": "admission-v1"}
            body = {
                "run_id": run_id, "arm": arm, "phase": phase, "transport": "connect",
                "workload": {"sha256": sha("same frozen workload"), "cases": 1},
                "concurrency": 4 if phase == "concurrent" else 1, "sample_interval_ms": 1000,
                "execution_context": context,
                "run_started_at_unix_ms": start,
                "run_ended_at_unix_ms": start + 1000,
                "started_at_unix_ms": start + 100,
                "ended_at_unix_ms": start + 900,
                "cases": [{"case_id": "case-1", "query_sha256": query_hash}],
                "cgroup_group_patterns": group_pattern,
                "cgroup_groups": {"api": {"available": True, "cpu_ms": 100 if arm == "baseline" else candidate_cpu,
                                              "peak_memory_current_bytes_sum": 10000 if arm == "baseline" else 8500},
                                  "serving_total": {"available": True, "cpu_ms": 150 if arm == "baseline" else 120,
                                                    "peak_memory_current_bytes_sum": 20000 if arm == "baseline" else 17000}},
                "cache_tokens_and_grades": {"search_hub_cache": {"available": True, "resolver_cache_hit_delta": 1},
                                             "ai_gateway_tokens": {"available": True, "input_tokens": 12}},
                "telemetry_scope": {"cache": "rolling-window; not run-attributed", "tokens": "scenario-window; not request-attributed"},
                "latency_ms": {"p50": 25 if arm == "baseline" else 20, "p95": 40 if arm == "baseline" else 32, "max": 60, "n": 1},
                "api_response_latency_ms": {"p50": 18 if arm == "baseline" else 15, "p95": 30 if arm == "baseline" else 25, "max": 44, "n": 1},
                "idle_seconds": None,
            }
            return write_json(pair_dir / f"{arm}.json", body)

        query_hash = sha("case query")
        base_report = report("baseline", started, query_hash)
        candidate_report = report("candidate", started + candidate_start_offset,
                                  candidate_query_hash or query_hash)
        base_grade = self.make_grade(pair_dir / "baseline-grade.json", "build-baseline")
        cand_grade = self.make_grade(pair_dir / "candidate-grade.json", "build-candidate", total_parts=candidate_parts_total)
        pair_receipt = pair_dir / "comparison.json"
        self.call_compare(base_report, candidate_report, base_grade, cand_grade, pair_receipt,
                          resource_group=resource_group, component_groups=component_groups)
        return {"receipt": pair_receipt, "inputs": {
            "baseline_report": base_report, "candidate_report": candidate_report,
            "baseline_grade_receipt": base_grade, "candidate_grade_receipt": cand_grade,
        }}

    def make_idle(self, arm: str, repeat: int, *, cpu_ms: int = 100) -> Path:
        idle_dir = self.root / "idle"
        start = self.next_ms
        self.next_ms += 20_000
        context = {"build_id": f"build-{arm}", "corpus_snapshot_id": "corpus-v1",
                   "fault_schedule_id": "none", "state_receipt_id": f"state-{arm}",
                    "admission_receipt_id": "admission-v1"}
        return write_json(idle_dir / f"{arm}-{repeat}.json", {
            "run_id": f"idle-{repeat}", "arm": arm, "phase": "idle", "workload": {"sha256": sha(""), "cases": 0},
            "execution_context": context, "idle_seconds": 10, "sample_interval_ms": 1000,
            "run_started_at_unix_ms": start, "run_ended_at_unix_ms": start + 12000,
            "cgroup_group_patterns": [{"name": "api", "comm_regex": "^search-hub-api$"},
                                      {"name": "serving_total", "comm_regex": "^search-hub-(api|provider)$"}],
            "cgroup_groups": {"api": {"available": True, "cpu_ms": cpu_ms,
                                          "peak_memory_current_bytes_sum": 10000},
                              "serving_total": {"available": True, "cpu_ms": cpu_ms,
                                                "peak_memory_current_bytes_sum": 10000}},
        })

    def make_manifest_plan(self, *, overlap: bool = False, candidate_idle_cpu_ms: int = 100,
                           include_authority: bool = True, component_groups: dict[str, str] | None = None) -> Path:
        primary_group = "serving_total" if component_groups else "api"
        phases = {}
        for phase in ("cold", "warm", "concurrent", "degraded"):
            phases[phase] = []
            for repeat in range(1, 4):
                pair = self.make_pair(phase, repeat,
                    candidate_start_offset=500 if overlap and phase == "warm" and repeat == 2 else 4000,
                    resource_group=primary_group, component_groups=component_groups)
                phases[phase].append({"receipt": pair["receipt"], "inputs": {key: str(value) for key, value in pair["inputs"].items()}})
        idle_reports = {
            "baseline": [self.make_idle("baseline", n) for n in range(1, 4)],
            "candidate": [self.make_idle("candidate", n, cpu_ms=candidate_idle_cpu_ms) for n in range(1, 4)],
        }
        evidence = []
        if include_authority:
            for receipt_id in ("admission-v1", "corpus-v1", "state-baseline", "state-candidate", "fault-degraded"):
                path = write_json(self.root / "authority" / f"{receipt_id}.json", {"receipt_id": receipt_id})
                evidence.append({"receipt_id": receipt_id, "path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()})
        plan = {
            "manifest_id": "offline-test-manifest", "admission_receipt_id": "admission-v1",
            "corpus_snapshot_id": "corpus-v1", "resource_source": "cgroup_groups", "resource_group": primary_group,
            "minimum_repeats": 3,
            "concurrency_by_phase": {"cold": 1, "warm": 1, "concurrent": 4, "degraded": 1},
            "phase_receipts": phases,
            "idle_reports": {arm: [str(path) for path in paths] for arm, paths in idle_reports.items()},
            "evidence_receipts": evidence,
        }
        if component_groups:
            plan["component_groups"] = component_groups
        return write_json(self.root / "plan.json", plan)

    def call_manifest(self, plan: Path, output: Path) -> None:
        with mock.patch.object(sys, "argv", ["manifest.py", "--plan", str(plan), "--output", str(output)]), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            manifest.main()

    def verify_manifest_bundle(self, plan: Path, bundle: Path) -> None:
        with mock.patch.object(sys, "argv", ["manifest.py", "--plan", str(plan), "--verify-bundle", str(bundle)]), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            manifest.main()

    def call_preflight(self, plan: Path, output: Path) -> None:
        with mock.patch.object(sys, "argv", ["manifest.py", "--plan", str(plan), "--preflight-report", str(output)]), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            manifest.main()

    def call_audit_manifest(self, plan: Path, bundle: Path, summary: Path) -> None:
        argv = ["manifest.py", "--plan", str(plan), "--output", str(bundle), "--audit-report", str(summary)]
        with mock.patch.object(sys, "argv", argv), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            manifest.main()

    def test_measure_rejects_non_idle_run_without_receipt_context_before_io(self) -> None:
        argv = ["measure.py", "--workload", str(self.root / "does-not-exist.jsonl"),
                "--output", str(self.root / "out.json"), "--phase", "warm", "--run-id", "r1", "--arm", "baseline"]
        with mock.patch.object(sys, "argv", argv), contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as error:
            measure.main()
        self.assertEqual(error.exception.code, 2)
        self.assertFalse((self.root / "out.json").exists())

    def test_signed_provider_scope_receipt_is_authenticated_and_group_bound(self) -> None:
        modulus, exponent, private_exponent = test_rsa_keypair()
        group_definition = {"name": "serving", "comm_regex": "^(search-hub|provider|model)$"}
        receipt = {
            "schema_version": 1, "receipt_id": "scope-receipt-v1", "issuer": "serving-owner",
            "key_id": "serving-owner-2026", "signature_algorithm": "rsa-pkcs1v15-sha256",
            "resource_source": "cgroup_groups", "resource_group": "serving",
            "resource_group_definition_sha256": scope_receipts.group_definition_sha256(group_definition),
            "coverage_status": "complete",
            "components": [
                {"component_id": "search-hub", "owner_id": "search-hub-owner", "kind": "search_hub"},
                {"component_id": "provider-a", "owner_id": "provider-owner", "kind": "provider"},
                {"component_id": "model-a", "owner_id": "model-owner", "kind": "model"},
            ],
            "unmeasured_costs": [],
        }
        encoded = scope_receipts.canonical_bytes(receipt)
        digest_info = scope_receipts.SHA256_DER_PREFIX + hashlib.sha256(encoded).digest()
        size = (modulus.bit_length() + 7) // 8
        encoded_message = b"\x00\x01" + b"\xff" * (size - len(digest_info) - 3) + b"\x00" + digest_info
        signature = pow(int.from_bytes(encoded_message, "big"), private_exponent, modulus).to_bytes(size, "big")
        receipt["signature_base64"] = base64.b64encode(signature).decode()
        modulus_hex = format(modulus, "x")
        fingerprint = scope_receipts.public_key_fingerprint("serving-owner", "serving-owner-2026", modulus_hex, exponent)
        admission = {"schema_version": 1, "receipt_id": "key-admission-v1",
                     "purpose": scope_receipts.SCOPE_PURPOSE, "issuer": "serving-owner",
                     "key_id": "serving-owner-2026", "key_fingerprint_sha256": fingerprint,
                     "approval_status": "approved", "owner_acknowledgement_ref": "owner-ack-1",
                     "manager_review_ref": "manager-review-1", "reviewer_ids": ["reviewer-a", "reviewer-b"]}
        policy = {"schema_version": 1, "keys": [{"issuer": "serving-owner", "key_id": "serving-owner-2026",
                                                   "modulus_hex": modulus_hex, "public_exponent": exponent,
                                                   "enabled": True, "purpose": scope_receipts.SCOPE_PURPOSE,
                                                   "key_fingerprint_sha256": fingerprint,
                                                   "admission_receipt_id": "key-admission-v1",
                                                   "admission_receipt_sha256": hashlib.sha256(scope_receipts.canonical_bytes(admission)).hexdigest(),
                                                   "reviewer_ids": ["reviewer-a", "reviewer-b"]}]}
        verified = scope_receipts.verify_scope_receipt(receipt, policy, resource_source="cgroup_groups",
            resource_group="serving", group_definition=group_definition)
        self.assertEqual(verified["signature_status"], "verified")
        self.assertEqual(verified["component_kinds"], ["model", "provider", "search_hub"])
        self.assertEqual(verified["key_admission_receipt_id"], "key-admission-v1")
        admission_verified = scope_receipts.verify_key_admission_artifact(admission,
            hashlib.sha256(scope_receipts.canonical_bytes(admission)).hexdigest(), verified)
        self.assertEqual(admission_verified["approval_status"], "approved")
        with self.assertRaisesRegex(ValueError, "hash differs"):
            scope_receipts.verify_key_admission_artifact(admission, sha("wrong bytes"), verified)
        with self.assertRaisesRegex(ValueError, "not uniquely pinned"):
            scope_receipts.verify_scope_receipt(receipt, {"schema_version": 1, "keys": []},
                resource_source="cgroup_groups", resource_group="serving", group_definition=group_definition)
        disabled_policy = {"schema_version": 1, "keys": [dict(policy["keys"][0], enabled=False)]}
        with self.assertRaisesRegex(ValueError, "disabled"):
            scope_receipts.verify_scope_receipt(receipt, disabled_policy,
                resource_source="cgroup_groups", resource_group="serving", group_definition=group_definition)
        bad_fingerprint_policy = {"schema_version": 1, "keys": [dict(policy["keys"][0], key_fingerprint_sha256=sha("wrong fingerprint"))]}
        with self.assertRaisesRegex(ValueError, "fingerprint"):
            scope_receipts.verify_scope_receipt(receipt, bad_fingerprint_policy,
                resource_source="cgroup_groups", resource_group="serving", group_definition=group_definition)
        tampered = dict(receipt, resource_group="other")
        with self.assertRaisesRegex(ValueError, "resource source/group"):
            scope_receipts.verify_scope_receipt(tampered, policy, resource_source="cgroup_groups",
                resource_group="serving", group_definition=group_definition)
        bad_signature = dict(receipt, signature_base64=base64.b64encode(bytes([signature[0] ^ 1]) + signature[1:]).decode())
        with self.assertRaisesRegex(ValueError, "signature verification failed"):
            scope_receipts.verify_scope_receipt(bad_signature, policy, resource_source="cgroup_groups",
                resource_group="serving", group_definition=group_definition)

    def test_run_telemetry_producer_and_signed_consumer_require_exact_case_coverage(self) -> None:
        modulus, exponent, private_exponent = test_rsa_keypair()
        pairs = [{"case_id": "case-a", "query_sha256": sha("query-a")},
                 {"case_id": "case-b", "query_sha256": sha("query-b")}]
        gateway = {"schema_version": 1, "run_id": "run-telemetry-1", "complete": True, "truncated": False,
                   "case_coverage": [{**row, "complete": True} for row in pairs],
                   "events": [{"event_id": "event-a-1", **pairs[0], "input_tokens": 11, "output_tokens": 4},
                              {"event_id": "event-a-2", **pairs[0], "input_tokens": 7, "output_tokens": 2}]}
        cache = {"schema_version": 1, "run_id": "run-telemetry-1", "complete": True,
                 "case_coverage": [{**pairs[0], "complete": True, "hits": 2, "misses": 1},
                                   {**pairs[1], "complete": True, "hits": 0, "misses": 3}]}
        sources = [{"kind": "gateway_export", "receipt_id": "gateway-export-1", "evidence_uri": "owner://gateway/1", "evidence_sha256": sha("gateway-export")},
                   {"kind": "cache_export", "receipt_id": "cache-export-1", "evidence_uri": "owner://cache/1", "evidence_sha256": sha("cache-export")}]
        unsigned = run_telemetry.produce_unsigned_receipt(receipt_id="run-receipt-1", issuer="telemetry-owner",
            key_id="telemetry-owner-2026", run_id="run-telemetry-1", arm="candidate", phase="warm",
            query_pairs=pairs, gateway_export=gateway, cache_export=cache, source_evidence=sources)
        self.assertEqual(unsigned["gateway"]["input_tokens"], 18)
        self.assertEqual(unsigned["gateway"]["output_tokens"], 6)
        self.assertEqual(unsigned["cache"], {"complete": True, "coverage_case_count": 2, "hits": 2, "misses": 4, "lookup_count": 6})
        message = run_telemetry.signing_bytes(unsigned)
        info = scope_receipts.SHA256_DER_PREFIX + hashlib.sha256(message).digest()
        size = (modulus.bit_length() + 7) // 8
        encoded = b"\x00\x01" + b"\xff" * (size - len(info) - 3) + b"\x00" + info
        signature = pow(int.from_bytes(encoded, "big"), private_exponent, modulus).to_bytes(size, "big")
        signed = {**unsigned, "signature_base64": base64.b64encode(signature).decode()}
        modulus_hex = format(modulus, "x")
        fingerprint = scope_receipts.public_key_fingerprint("telemetry-owner", "telemetry-owner-2026", modulus_hex, exponent)
        admission = {"schema_version": 1, "receipt_id": "telemetry-key-admission-1", "purpose": run_telemetry.TELEMETRY_PURPOSE,
                     "issuer": "telemetry-owner", "key_id": "telemetry-owner-2026", "key_fingerprint_sha256": fingerprint,
                     "approval_status": "approved", "owner_acknowledgement_ref": "owner-ack", "manager_review_ref": "manager-review",
                     "reviewer_ids": ["reviewer-a", "reviewer-b"]}
        policy = {"schema_version": 1, "keys": [{"issuer": "telemetry-owner", "key_id": "telemetry-owner-2026",
            "modulus_hex": modulus_hex, "public_exponent": exponent, "enabled": True,
            "purpose": run_telemetry.TELEMETRY_PURPOSE, "key_fingerprint_sha256": fingerprint,
            "admission_receipt_id": "telemetry-key-admission-1",
            "admission_receipt_sha256": hashlib.sha256(scope_receipts.canonical_bytes(admission)).hexdigest(),
            "reviewer_ids": ["reviewer-a", "reviewer-b"]}]}
        verified = run_telemetry.verify_receipt(signed, policy, run_id="run-telemetry-1", arm="candidate", phase="warm",
            case_query_set_sha256=run_telemetry.query_set_digest(pairs), case_count=2)
        self.assertTrue(verified["verified"])
        self.assertEqual(verified["key_admission_receipt_id"], "telemetry-key-admission-1")
        self.assertEqual(verified["source_evidence"], sources)
        with self.assertRaisesRegex(ValueError, "not enabled"):
            run_telemetry.verify_receipt(signed, {"schema_version": 1, "keys": []}, run_id="run-telemetry-1", arm="candidate",
                phase="warm", case_query_set_sha256=run_telemetry.query_set_digest(pairs), case_count=2)
        tampered = dict(signed, case_count=3)
        with self.assertRaisesRegex(ValueError, "case_count differs"):
            run_telemetry.verify_receipt(tampered, policy, run_id="run-telemetry-1", arm="candidate", phase="warm",
                case_query_set_sha256=run_telemetry.query_set_digest(pairs), case_count=2)
        incomplete = dict(gateway, case_coverage=gateway["case_coverage"][:1])
        with self.assertRaisesRegex(ValueError, "exactly match"):
            run_telemetry.produce_unsigned_receipt(receipt_id="bad", issuer="telemetry-owner", key_id="telemetry-owner-2026",
                run_id="run-telemetry-1", arm="candidate", phase="warm", query_pairs=pairs,
                gateway_export=incomplete, cache_export=cache, source_evidence=sources)

    def test_stage_telemetry_contract_requires_exact_cases_stages_and_owner_signature(self) -> None:
        self.assertEqual(json.loads((HERE / "stage-trace-export-v1.schema.json").read_text())["properties"]["case_coverage"]["type"], "array")
        self.assertEqual(json.loads((HERE / "stage-trace-receipt-v1.schema.json").read_text())["properties"]["signature_algorithm"]["const"], "rsa-pkcs1v15-sha256")
        self.assertEqual(json.loads((HERE / "stage-telemetry-trust-v1.json").read_text())["keys"], [])
        self.assertEqual(json.loads((HERE / "stage-telemetry-trust-v1.schema.json").read_text())["properties"]["keys"]["items"]["properties"]["purpose"]["const"], stage_telemetry.STAGE_PURPOSE)
        modulus, exponent, private_exponent = test_rsa_keypair()
        pairs = [{"case_id": "case-a", "query_sha256": sha("query-a")},
                 {"case_id": "case-b", "query_sha256": sha("query-b")}]
        cases = []
        for index, pair in enumerate(pairs):
            stages = {}
            for stage_id in stage_telemetry.STAGE_IDS:
                executed = not (stage_id == "rerank" and index == 1)
                stages[stage_id] = {"executed": executed, "duration_ms": (index + 1) * 5 if executed else None,
                    "cpu_ms": (index + 1) * 3 if executed else None,
                    "peak_memory_bytes": (index + 1) * 1000 if executed else None,
                    "resource_method_id": "owner-cgroup-v1", "resource_scope_receipt_id": "scope-1",
                    "resource_scope_uri": "owner://stage-resource-scope/1", "resource_scope_sha256": sha("stage-resource-scope")}
            cases.append({**pair, "stages": stages})
        export = {"schema_version": 1, "run_id": "stage-run-1", "arm": "candidate", "phase": "warm",
                  "complete": True, "truncated": False, "case_coverage": cases}
        source = {"receipt_id": "trace-export-1", "evidence_uri": "owner://trace/1", "evidence_sha256": sha("trace-export")}
        unsigned = stage_telemetry.produce_unsigned_receipt(receipt_id="stage-receipt-1", issuer="trace-owner",
            key_id="trace-owner-2026", run_id="stage-run-1", arm="candidate", phase="warm",
            query_pairs=pairs, export=export, source_evidence=source)
        self.assertEqual(unsigned["stages"]["rerank"], {"executed_case_count": 1, "not_applicable_case_count": 1,
                                                           "complete_case_count": 2, "durations_ms": [5],
                                                           "resource_status": "owner_attributed_diagnostic",
                                                           "resource_method_id": "owner-cgroup-v1",
                                                           "resource_scope_receipt_id": "scope-1",
                                                           "resource_scope_uri": "owner://stage-resource-scope/1",
                                                           "resource_scope_sha256": sha("stage-resource-scope"),
                                                           "cpu_ms_total": 3, "peak_memory_bytes": 1000})
        message = stage_telemetry.signing_bytes(unsigned)
        info = scope_receipts.SHA256_DER_PREFIX + hashlib.sha256(message).digest()
        size = (modulus.bit_length() + 7) // 8
        encoded = b"\x00\x01" + b"\xff" * (size - len(info) - 3) + b"\x00" + info
        signature = pow(int.from_bytes(encoded, "big"), private_exponent, modulus).to_bytes(size, "big")
        signed = {**unsigned, "signature_base64": base64.b64encode(signature).decode()}
        modulus_hex = format(modulus, "x")
        fingerprint = scope_receipts.public_key_fingerprint("trace-owner", "trace-owner-2026", modulus_hex, exponent)
        policy = {"schema_version": 1, "keys": [{"issuer": "trace-owner", "key_id": "trace-owner-2026",
            "modulus_hex": modulus_hex, "public_exponent": exponent, "enabled": True,
            "purpose": stage_telemetry.STAGE_PURPOSE, "key_fingerprint_sha256": fingerprint,
            "admission_receipt_id": "trace-admission", "admission_receipt_sha256": sha("admission"),
            "reviewer_ids": ["reviewer-a", "reviewer-b"]}]}
        verified = stage_telemetry.verify_receipt(signed, policy, run_id="stage-run-1", arm="candidate", phase="warm", query_pairs=pairs)
        self.assertTrue(verified["verified"])
        self.assertEqual(verified["stages"], unsigned["stages"])
        baseline_cases = []
        for case in cases:
            baseline_stages = {stage_id: {**stage,
                "duration_ms": stage["duration_ms"] * 2 if stage["duration_ms"] is not None else None,
                "cpu_ms": stage["cpu_ms"] * 2 if stage["cpu_ms"] is not None else None,
                "peak_memory_bytes": stage["peak_memory_bytes"] * 2 if stage["peak_memory_bytes"] is not None else None}
                for stage_id, stage in case["stages"].items()}
            baseline_cases.append({**case, "stages": baseline_stages})
        baseline_export = {**export, "arm": "baseline", "case_coverage": baseline_cases}
        baseline_unsigned = stage_telemetry.produce_unsigned_receipt(receipt_id="stage-base-receipt-1", issuer="trace-owner",
            key_id="trace-owner-2026", run_id="stage-run-1", arm="baseline", phase="warm",
            query_pairs=pairs, export=baseline_export, source_evidence=source)
        baseline_signed = test_sign_payload(baseline_unsigned, stage_telemetry, modulus, private_exponent)
        baseline_verified = stage_telemetry.verify_receipt(baseline_signed, policy, run_id="stage-run-1", arm="baseline",
            phase="warm", query_pairs=pairs)
        stage_pair = stage_telemetry.compare_verified_receipts(baseline_verified, verified, quality_gate=True,
            baseline_accepted_parts=2, candidate_accepted_parts=2, baseline_accepted_tasks=1, candidate_accepted_tasks=1)
        self.assertEqual(stage_pair["stages"]["classifier"]["status"], "descriptive_pair_stage_latency")
        self.assertEqual(stage_pair["stages"]["classifier"]["candidate_to_baseline_total_duration_ratio"], 0.5)
        self.assertEqual(stage_pair["stages"]["classifier"]["resource_attribution"]["cpu_ms_ratio"], 0.5)
        self.assertEqual(stage_pair["stages"]["classifier"]["resource_attribution"]["peak_memory_ratio"], 0.5)
        self.assertEqual(stage_pair["stages"]["classifier"]["resource_attribution"]["cpu_ms_per_accepted_task"]["candidate_to_baseline_ratio"], 0.5)
        self.assertEqual(stage_pair["resource_attribution_status"], "descriptive_owner_attributed_stage_resources_available")
        self.assertEqual(stage_pair["stages"]["rerank"]["paired_case_count"], 1)
        scope_mismatch = json.loads(json.dumps(verified))
        scope_mismatch["stages"]["classifier"]["resource_scope_sha256"] = sha("different-resource-scope")
        mismatch_pair = stage_telemetry.compare_verified_receipts(baseline_verified, scope_mismatch, quality_gate=True)
        self.assertEqual(mismatch_pair["stages"]["classifier"]["resource_attribution"]["status"],
                         "unavailable_stage_cpu_memory_not_measured")
        self.assertEqual(mismatch_pair["stages"]["resolver"]["resource_attribution"]["cpu_ms_ratio"], 0.5)
        with self.assertRaisesRegex(ValueError, "not enabled"):
            stage_telemetry.verify_receipt(signed, {"schema_version": 1, "keys": []}, run_id="stage-run-1",
                arm="candidate", phase="warm", query_pairs=pairs)
        tampered = dict(signed, stages={**signed["stages"], "resolver": {"duration_ms": 1}})
        with self.assertRaisesRegex(ValueError, "signature verification failed"):
            stage_telemetry.verify_receipt(tampered, policy, run_id="stage-run-1", arm="candidate", phase="warm", query_pairs=pairs)
        truncated = dict(export, truncated=True)
        with self.assertRaisesRegex(ValueError, "not truncated"):
            stage_telemetry.produce_unsigned_receipt(receipt_id="bad", issuer="trace-owner", key_id="trace-owner-2026",
                run_id="stage-run-1", arm="candidate", phase="warm", query_pairs=pairs, export=truncated, source_evidence=source)
        missing_stage = dict(export, case_coverage=[{**cases[0], "stages": {"classifier": cases[0]["stages"]["classifier"]}}])
        with self.assertRaisesRegex(ValueError, "explicit coverage"):
            stage_telemetry.produce_unsigned_receipt(receipt_id="bad", issuer="trace-owner", key_id="trace-owner-2026",
                run_id="stage-run-1", arm="candidate", phase="warm", query_pairs=pairs, export=missing_stage, source_evidence=source)
        partial_resource = json.loads(json.dumps(export))
        partial_resource["case_coverage"][0]["stages"]["classifier"]["cpu_ms"] = 10
        partial_resource["case_coverage"][0]["stages"]["classifier"].pop("peak_memory_bytes")
        with self.assertRaisesRegex(ValueError, "field allowlist"):
            stage_telemetry.produce_unsigned_receipt(receipt_id="bad", issuer="trace-owner", key_id="trace-owner-2026",
                run_id="stage-run-1", arm="candidate", phase="warm", query_pairs=pairs,
                export=partial_resource, source_evidence=source)

    def test_gateway_token_window_can_be_complete_but_not_run_attributable(self) -> None:
        before = {"available": True, "truncated": False, "events": [{"event_id": "old"}]}
        after = {"available": True, "truncated": False, "events": [
            {"event_id": "old"}, {"event_id": "new", "input_tokens": 7, "output_tokens": 3},
        ]}
        delta = measure.gateway_delta(before, after)
        self.assertTrue(delta["token_usage_complete"])
        self.assertEqual(delta["input_tokens"], 7)
        self.assertEqual(delta["attribution_status"], "scenario_window_not_request_or_run_attributed")

    def test_compare_stage_receipt_integration_fails_closed_with_empty_owner_allowlist(self) -> None:
        pair = self.make_pair("warm", 7)
        base_report = json.loads(pair["inputs"]["baseline_report"].read_text())
        query_pairs = [{"case_id": row["case_id"], "query_sha256": row["query_sha256"]} for row in base_report["cases"]]
        modulus, _, private_exponent = test_rsa_keypair()
        trace_paths = []
        for arm in ("baseline", "candidate"):
            coverage = [{**row, "stages": {stage_id: {"executed": True, "duration_ms": 5}
                                                   for stage_id in stage_telemetry.STAGE_IDS}}
                        for row in query_pairs]
            export = {"schema_version": 1, "run_id": "warm-7", "arm": arm, "phase": "warm",
                      "complete": True, "truncated": False, "case_coverage": coverage}
            unsigned = stage_telemetry.produce_unsigned_receipt(receipt_id=f"stage-{arm}-7", issuer="test-owner",
                key_id="test-stage-key", run_id="warm-7", arm=arm, phase="warm", query_pairs=query_pairs,
                export=export, source_evidence={"receipt_id": f"trace-{arm}", "evidence_uri": f"owner://{arm}",
                                                "evidence_sha256": sha(arm)})
            signed = test_sign_payload(unsigned, stage_telemetry, modulus, private_exponent)
            trace_paths.append(write_json(self.root / f"{arm}-stage.json", signed))
        output = self.root / "stage-integration-output.json"
        with self.assertRaises(SystemExit):
            self.call_compare(pair["inputs"]["baseline_report"], pair["inputs"]["candidate_report"],
                pair["inputs"]["baseline_grade_receipt"], pair["inputs"]["candidate_grade_receipt"], output,
                (trace_paths[0], trace_paths[1]))
        self.assertFalse(output.exists())
        modulus_hex = format(modulus, "x")
        fingerprint = scope_receipts.public_key_fingerprint("test-owner", "test-stage-key", modulus_hex, 65537)
        stage_policy = {"schema_version": 1, "keys": [{"issuer": "test-owner", "key_id": "test-stage-key",
            "modulus_hex": modulus_hex, "public_exponent": 65537, "enabled": True,
            "purpose": stage_telemetry.STAGE_PURPOSE, "key_fingerprint_sha256": fingerprint,
            "admission_receipt_id": "test-stage-admission", "admission_receipt_sha256": sha("stage-admission"),
            "reviewer_ids": ["test-reviewer-a", "test-reviewer-b"]}]}
        original_read_json = compare.read_json
        def patched_read_json(path):
            return stage_policy if Path(path).name == "stage-telemetry-trust-v1.json" else original_read_json(path)
        accepted_output = self.root / "stage-integration-accepted.json"
        with mock.patch.object(compare, "read_json", side_effect=patched_read_json):
            self.call_compare(pair["inputs"]["baseline_report"], pair["inputs"]["candidate_report"],
                pair["inputs"]["baseline_grade_receipt"], pair["inputs"]["candidate_grade_receipt"], accepted_output,
                (trace_paths[0], trace_paths[1]))
        accepted = json.loads(accepted_output.read_text())
        self.assertEqual(accepted["stage_latency"]["stages"]["resolver"]["status"], "descriptive_pair_stage_latency")
        self.assertEqual(accepted["stage_latency"]["measurement_dimension"], "latency_only")
        self.assertEqual(accepted["stage_latency"]["resource_attribution_status"], "unavailable_stage_cpu_memory_not_measured")
        self.assertIn("baseline_stage_trace", accepted["receipt"]["inputs"])
        self.assertEqual(set(accepted["comparison"]["correlated_stage_telemetry"]), {"baseline", "candidate"})

    def test_manifest_can_compare_future_run_correlated_token_and_cache_evidence(self) -> None:
        rows = []
        for repeat in range(1, 4):
            run_id = f"run-{repeat}"
            common = {"available": True, "signature_verified": True, "attribution_status": "run_correlated", "run_id": run_id,
                      "case_query_set_sha256": "frozen-query-set"}
            rows.append({"receipt_id": f"pair-{repeat}", "run_id": run_id,
                         "grade_case_query_set_sha256": "frozen-query-set",
                         "accepted_parts": {"baseline": 2, "candidate": 2},
                         "baseline_acceptance_nonregression": True,
                         "separate_evidence": {
                             "baseline_tokens": {**common, "input_tokens": 100, "output_tokens": 50},
                             "candidate_tokens": {**common, "input_tokens": 80, "output_tokens": 40},
                             "baseline_cache": {**common, "resolver_cache_hit_delta": 2, "resolver_cache_miss_delta": 2},
                             "candidate_cache": {**common, "resolver_cache_hit_delta": 3, "resolver_cache_miss_delta": 1},
                         },
                         "stage_latency": {"stages": {stage_id: {
                             "status": "descriptive_pair_stage_latency",
                             "candidate_to_baseline_total_duration_ratio": 0.8,
                             "resource_attribution": {"status": "descriptive_owner_attributed_stage_resource",
                                 "method_id": "owner-cgroup-v1", "scope_receipt_id": "scope-1",
                                 "scope_uri": "owner://stage-resource-scope/1", "scope_sha256": sha("stage-resource-scope"),
                                 "cpu_ms": {"baseline": 100.0, "candidate": 70.0 + repeat},
                                     "cpu_ms_per_accepted_part": {"baseline": 50.0, "candidate": 35.0 + repeat / 2},
                                     "cpu_ms_per_accepted_task": {"baseline": 100.0, "candidate": 70.0 + repeat},
                                 "peak_memory_bytes": {"baseline": 1000.0, "candidate": 800.0}},
                         } for stage_id in stage_telemetry.STAGE_IDS}}})
        summary = manifest.run_attributed_telemetry(rows)
        self.assertAlmostEqual(summary["metrics"]["input_tokens_per_accepted_part"]["candidate_to_baseline_ratio"]["mean"], 0.8)
        self.assertAlmostEqual(summary["metrics"]["output_tokens_per_accepted_part"]["candidate_to_baseline_ratio"]["mean"], 0.8)
        self.assertAlmostEqual(summary["metrics"]["cache_hit_rate_delta"]["candidate_minus_baseline_rate"]["mean"], 0.25)
        budget = manifest.phase_budget_rollup({"warm": rows}, provider_scope_hash_present=False)
        self.assertEqual(budget["phases"]["warm"]["tokens_and_cache"]["status"], "run_attributed_summary_available")
        stage_resource = budget["phases"]["warm"]["stage_latency"]
        self.assertEqual(stage_resource["resource_attribution_status"], "descriptive_stage_resource_diagnostics_available")
        self.assertAlmostEqual(stage_resource["metrics"]["resolver"]["resource_attribution"]["metrics"]["cpu_ms"]["candidate_to_baseline_ratio"]["mean"], 0.72)
        scope_drift_rows = json.loads(json.dumps(rows))
        scope_drift_rows[1]["stage_latency"]["stages"]["resolver"]["resource_attribution"]["scope_sha256"] = sha("different-scope")
        scope_drift = manifest.phase_budget_rollup({"warm": scope_drift_rows}, provider_scope_hash_present=False)
        self.assertEqual(scope_drift["phases"]["warm"]["stage_latency"]["metrics"]["resolver"]["resource_attribution"]["status"], "unavailable_stage_resource_method_or_scope_drift")
        rows[1]["stage_latency"]["stages"]["resolver"].pop("resource_attribution")
        incomplete = manifest.phase_budget_rollup({"warm": rows}, provider_scope_hash_present=False)
        self.assertEqual(incomplete["phases"]["warm"]["stage_latency"]["metrics"]["resolver"]["resource_attribution"]["metrics"]["cpu_ms"]["status"], "unavailable")

    def test_student_t_interval_critical_value_remains_finite_sample_correct_above_table(self) -> None:
        self.assertAlmostEqual(manifest.student_t_975(31), 2.0395, delta=0.0002)
        self.assertAlmostEqual(manifest.student_t_975(60), 2.0003, delta=0.0002)
        self.assertAlmostEqual(manifest.student_t_975(1000), 1.9623, delta=0.0002)
        self.assertGreater(manifest.student_t_975(31), 1.96)
        with self.assertRaisesRegex(ValueError, "at least two degrees"):
            manifest.student_t_975(1)

    def test_task_and_part_cpu_floors_are_independently_gated(self) -> None:
        rows = [{"receipt_id": f"task-floor-{index}", "baseline_acceptance_nonregression": True,
                 "budget_metrics": {"cpu_part": (100.0, 70.0), "cpu_task": (100.0, 90.0),
                                    "peak_memory": (1000.0, 800.0), "client_p95": (10.0, 9.0),
                                    "api_p95": (8.0, 7.0)}} for index in range(3)]
        metrics = manifest.phase_budget_rollup({"warm": rows}, provider_scope_hash_present=True)["phases"]["warm"]["metrics"]
        self.assertTrue(metrics["cpu_per_accepted_part"]["conditional_threshold_met"])
        self.assertFalse(metrics["cpu_per_accepted_task"]["conditional_threshold_met"])
        self.assertEqual(metrics["cpu_per_accepted_part"]["required_max_ratio"], 0.80)
        self.assertEqual(metrics["cpu_per_accepted_task"]["required_max_ratio"], 0.80)

    def test_client_p95_floor_requires_task_quality_gate_and_paired_uncertainty(self) -> None:
        rows = [{"receipt_id": f"latency-floor-{index}", "baseline_acceptance_nonregression": True,
                 "budget_metrics": {"cpu_part": (100.0, 90.0), "cpu_task": (100.0, 90.0),
                                    "peak_memory": (1000.0, 900.0), "client_p95": (100.0, 60.0),
                                    "api_p95": (80.0, 55.0)}} for index in range(3)]
        metrics = manifest.phase_budget_rollup({"warm": rows}, provider_scope_hash_present=False)["phases"]["warm"]["metrics"]
        client = metrics["client_latency_p95"]
        self.assertTrue(client["conditional_threshold_met"])
        self.assertFalse(client["provider_model_scope_required_for_metric"])
        self.assertEqual(client["confidence_floor_status"], "conditional_interval_within_threshold_pending_showcase_review")
        rows[1]["baseline_acceptance_nonregression"] = False
        blocked = manifest.phase_budget_rollup({"warm": rows}, provider_scope_hash_present=False)["phases"]["warm"]["metrics"]["client_latency_p95"]
        self.assertFalse(blocked["conditional_threshold_met"])
        self.assertEqual(blocked["confidence_floor_status"], "ineligible_quality_regression")

    def test_compare_accepts_equivalent_pair_and_binds_receipt_hashes(self) -> None:
        pair = self.make_pair("warm", 1)
        result = json.loads(pair["receipt"].read_text())
        self.assertEqual(result["comparison"]["phase"], "warm")
        self.assertEqual(result["comparison"]["case_count"], 1)
        self.assertIsNone(result["provider_model_attribution"]["scope_receipt_id"])
        self.assertFalse(result["telemetry_scope"]["tokens_comparable"])
        self.assertFalse(result["telemetry_scope"]["cache_comparable"])
        self.assertEqual(result["independent_grades"]["acceptance_nonregression_gate"], True)
        self.assertEqual(result["stage_latency"]["status"], "unavailable_stage_receipts_not_supplied")
        self.assertEqual(result["stage_latency"]["resource_attribution_status"], "unavailable_stage_cpu_memory_not_measured")
        self.assertTrue(all(value["candidate_to_baseline_total_duration_ratio"] is None
                            for value in result["stage_latency"]["stages"].values()))
        self.assertTrue(result["resource_attribution"]["cpu_floor_point_estimate_pass"])
        self.assertTrue(result["resource_attribution"]["cpu_task_floor_point_estimate_pass"])
        self.assertEqual(result["receipt"]["inputs"]["baseline_report"]["sha256"], hashlib.sha256(pair["inputs"]["baseline_report"].read_bytes()).hexdigest())
        encoded = json.dumps(result)
        self.assertNotIn("case-1", encoded)
        self.assertNotIn("case query", encoded)
        self.assertNotIn("verdicts", encoded)

    def test_compare_reports_component_resource_groups_separately_from_primary_total(self) -> None:
        pair = self.make_pair("warm", 8)
        output = self.root / "component-comparison.json"
        self.call_compare(pair["inputs"]["baseline_report"], pair["inputs"]["candidate_report"],
            pair["inputs"]["baseline_grade_receipt"], pair["inputs"]["candidate_grade_receipt"], output,
            resource_group="serving_total", component_groups={"search_hub_api": "api"})
        result = json.loads(output.read_text())
        component = result["resource_attribution"]["component_groups"]["search_hub_api"]
        self.assertEqual(component["group"], "api")
        self.assertEqual(component["cpu_per_accepted_task_ratio"], 0.8)
        self.assertEqual(component["peak_memory_ratio"], 0.85)
        self.assertEqual(component["quality_gate"], True)
        self.assertIn("never summed", component["interpretation"])
        self.assertEqual(result["resource_attribution"]["cpu_ms"], {"baseline": 150.0, "candidate": 120.0})
        self.assertIn("do not sum", result["resource_attribution"]["component_group_policy"])

    def test_manifest_rolls_up_component_resource_groups_without_summing(self) -> None:
        plan = self.make_manifest_plan(component_groups={"search_hub_api": "api"})
        output = self.root / "component-manifest.json"
        self.call_manifest(plan, output)
        self.verify_manifest_bundle(plan, output)
        bundle = json.loads(output.read_text())
        warm = bundle["phase_budget_rollup"]["phases"]["warm"]["component_resource_attribution"]
        self.assertEqual(warm["status"], "descriptive_component_groups_available")
        api = warm["roles"]["search_hub_api"]
        self.assertEqual(api["status"], "descriptive_component_group_rollup")
        self.assertEqual(api["group"], "api")
        self.assertAlmostEqual(api["metrics"]["cpu_task"]["candidate_to_baseline_ratio"]["mean"], 0.8)
        self.assertEqual(bundle["manifest"]["common_contract"]["component_groups"][0]["role"], "search_hub_api")
        self.assertFalse(bundle["showcase_gate_matrix"]["performance_claim_allowed"])

    def test_grade_schema_is_strict_and_parseable(self) -> None:
        schema = json.loads((HERE / "sh1-grade-export-v1.schema.json").read_text())
        self.assertEqual(schema["$schema"], "https://json-schema.org/draft/2020-12/schema")
        self.assertFalse(schema["additionalProperties"])
        self.assertIn("evaluation_run_ids", schema["required"])
        self.assertIn("evidence_sha256", schema["required"])

    def test_provider_model_scope_schema_is_strict_and_pins_complete_coverage_shape(self) -> None:
        schema = json.loads((HERE / "provider-model-scope-receipt-v1.schema.json").read_text())
        self.assertFalse(schema["additionalProperties"])
        self.assertIn("signature_base64", schema["required"])
        self.assertEqual(schema["properties"]["coverage_status"]["const"], "complete")
        policy = json.loads((HERE / "provider-model-scope-trust-v1.json").read_text())
        self.assertEqual(policy["keys"], [])
        trust_schema = json.loads((HERE / "provider-model-scope-trust-v1.schema.json").read_text())
        self.assertFalse(trust_schema["additionalProperties"])
        self.assertIn("admission_receipt_sha256", trust_schema["properties"]["keys"]["items"]["required"])
        admission_schema = json.loads((HERE / "provider-model-scope-key-admission-v1.schema.json").read_text())
        self.assertFalse(admission_schema["additionalProperties"])
        self.assertIn("manager_review_ref", admission_schema["required"])
        run_schema = json.loads((HERE / "run-telemetry-receipt-v1.schema.json").read_text())
        self.assertFalse(run_schema["additionalProperties"])
        self.assertIn("source_evidence", run_schema["required"])
        telemetry_policy = json.loads((HERE / "run-telemetry-trust-v1.json").read_text())
        self.assertEqual(telemetry_policy["keys"], [])
        self.assertEqual(json.loads((HERE / "run-telemetry-key-admission-v1.schema.json").read_text())["properties"]["purpose"]["const"], run_telemetry.TELEMETRY_PURPOSE)
        duplicate = self.root / "duplicate-scope.json"
        duplicate.write_text('{"receipt_id":"first","receipt_id":"second"}')
        with self.assertRaisesRegex(ValueError, "duplicate JSON field"):
            scope_receipts.read_scope_receipt(duplicate)

    def test_compare_rejects_query_hash_mismatch(self) -> None:
        # The comparator must reject before writing an output receipt.
        with self.assertRaises(SystemExit):
            self.make_pair("warm", 2, candidate_query_hash=sha("different query"))
        self.assertFalse((self.root / "pairs" / "warm" / "2" / "comparison.json").exists())

    def test_compare_rejects_grade_denominator_mismatch(self) -> None:
        phase_dir = self.root / "pairs" / "denominator"
        start = self.next_ms
        self.next_ms += 20000
        # Reuse the pair builder, which invokes compare; alter a grade and invoke it directly.
        pair = self.make_pair("warm", 3)
        candidate_report = pair["inputs"]["candidate_report"]
        base_grade = pair["inputs"]["baseline_grade_receipt"]
        candidate_grade = pair["inputs"]["candidate_grade_receipt"]
        grade = json.loads(candidate_grade.read_text())
        grade["cases"]["case-1"]["parts"]["total"] = 3
        grade["cases"]["case-1"]["parts"]["verdicts"] = [True, False, False]
        write_json(candidate_grade, grade)
        output = phase_dir / "should-not-exist.json"
        with self.assertRaises(SystemExit):
            self.call_compare(pair["inputs"]["baseline_report"], candidate_report, base_grade, candidate_grade, output)
        self.assertFalse(output.exists())

    def test_compare_rejects_grade_receipt_with_wrong_build_or_query(self) -> None:
        pair = self.make_pair("warm", 4)
        original = json.loads(pair["inputs"]["candidate_grade_receipt"].read_text())
        for field, value in (("build_id", "unmeasured-build"),):
            bad = dict(original)
            bad[field] = value
            grade_path = write_json(self.root / f"bad-{field}.json", bad)
            with self.assertRaises(SystemExit):
                self.call_compare(pair["inputs"]["baseline_report"], pair["inputs"]["candidate_report"],
                                  pair["inputs"]["baseline_grade_receipt"], grade_path,
                                  self.root / f"{field}-output.json")
        bad_query = dict(original)
        bad_query["cases"] = {"case-1": dict(original["cases"]["case-1"], query_sha256=sha("wrong query"))}
        bad_query_path = write_json(self.root / "bad-query-grade.json", bad_query)
        with self.assertRaises(SystemExit):
            self.call_compare(pair["inputs"]["baseline_report"], pair["inputs"]["candidate_report"],
                              pair["inputs"]["baseline_grade_receipt"], bad_query_path,
                              self.root / "bad-query-output.json")

    def test_compare_rejects_extra_private_grade_fields(self) -> None:
        pair = self.make_pair("warm", 6)
        grade = json.loads(pair["inputs"]["candidate_grade_receipt"].read_text())
        grade["expected_ids"] = ["private-expected-identity"]
        candidate_grade = write_json(self.root / "extra-private-grade-field.json", grade)
        with self.assertRaises(SystemExit):
            self.call_compare(pair["inputs"]["baseline_report"], pair["inputs"]["candidate_report"],
                              pair["inputs"]["baseline_grade_receipt"], candidate_grade,
                              self.root / "private-field-output.json")

    def test_compare_detects_part_regression_hidden_by_equal_aggregate_rate(self) -> None:
        pair = self.make_pair("warm", 5)
        grade = json.loads(pair["inputs"]["candidate_grade_receipt"].read_text())
        grade["cases"]["case-1"]["parts"]["verdicts"] = [False, True]
        candidate_grade = write_json(self.root / "swapped-part-verdicts.json", grade)
        output = self.root / "part-regression-comparison.json"
        self.call_compare(pair["inputs"]["baseline_report"], pair["inputs"]["candidate_report"],
                          pair["inputs"]["baseline_grade_receipt"], candidate_grade, output)
        result = json.loads(output.read_text())
        self.assertFalse(result["independent_grades"]["acceptance_nonregression_gate"])
        self.assertEqual(result["independent_grades"]["regressed_part_count"], 1)
        self.assertEqual(result["independent_grades"]["baseline"]["accepted_parts"],
                         result["independent_grades"]["candidate"]["accepted_parts"])

    def test_manifest_accepts_complete_bundle_and_idle_in_uncertainty(self) -> None:
        plan = self.make_manifest_plan()
        output = self.root / "manifest.json"
        self.call_manifest(plan, output)
        self.verify_manifest_bundle(plan, output)
        saved_bundle = output.read_bytes()
        result = json.loads(saved_bundle)
        self.assertEqual(result["provenance"]["plan"], {"name": plan.name, "sha256": hashlib.sha256(plan.read_bytes()).hexdigest()})
        first_pair = result["phase_receipts"]["cold"][0]
        self.assertEqual(first_pair["source_artifacts"]["pair_receipt"],
                         {"name": first_pair["path"], "sha256": first_pair["sha256"]})
        for role in ("baseline_report", "candidate_report", "baseline_grade_receipt", "candidate_grade_receipt"):
            ref = first_pair["source_artifacts"]["inputs"][role]
            self.assertTrue(ref["name"])
            self.assertEqual(len(ref["sha256"]), 64)
        output.write_bytes(saved_bundle + b" ")
        with self.assertRaises(SystemExit):
            self.verify_manifest_bundle(plan, output)
        output.write_bytes(saved_bundle)
        plan_data = json.loads(plan.read_text())
        report_path = Path(plan_data["phase_receipts"]["cold"][0]["inputs"]["baseline_report"])
        saved_report = report_path.read_bytes()
        report_path.write_bytes(saved_report + b" ")
        with self.assertRaises(SystemExit):
            self.verify_manifest_bundle(plan, output)
        report_path.write_bytes(saved_report)
        result = json.loads(output.read_text())
        self.assertEqual({key: len(value) for key, value in result["phase_receipts"].items()},
                         {"cold": 3, "warm": 3, "concurrent": 3, "degraded": 3})
        self.assertTrue(result["idle_check"]["idle_cpu_within_baseline_uncertainty"])
        self.assertTrue(result["idle_check"]["idle_memory_within_baseline_uncertainty"])
        rollup = result["phase_budget_rollup"]["phases"]
        self.assertEqual(set(rollup), {"cold", "warm", "concurrent", "degraded"})
        cold = rollup["cold"]
        self.assertEqual(cold["metrics"]["cpu_per_accepted_part"]["candidate_to_baseline_ratio"]["sample_count"], 3)
        self.assertFalse(cold["metrics"]["cpu_per_accepted_part"]["conditional_threshold_met"])
        self.assertEqual(cold["metrics"]["cpu_per_accepted_part"]["confidence_floor_status"], "ineligible_missing_provider_model_scope_receipt")
        self.assertFalse(cold["metrics"]["cpu_per_accepted_part"]["provider_model_scope_receipt_hash_present"])
        self.assertFalse(cold["metrics"]["peak_memory"]["conditional_threshold_met"])
        self.assertAlmostEqual(cold["metrics"]["client_latency_p95"]["candidate_to_baseline_ratio"]["mean"], 0.8)
        self.assertFalse(cold["metrics"]["client_latency_p95"]["conditional_threshold_met"])
        self.assertEqual(cold["metrics"]["client_latency_p95"]["required_max_ratio"], 0.70)
        self.assertEqual(cold["tokens_and_cache"]["status"], "run_attributed_summary_unavailable; ambient snapshots retained separately and never converted to serving CPU")
        self.assertEqual(cold["stage_latency"]["status"], "stage_latency_incomplete_or_unavailable")
        self.assertEqual(cold["stage_latency"]["metrics"]["resolver"]["status"], "unavailable_missing_or_invalid_stage_receipt")
        self.assertEqual(len(cold["tokens_and_cache"]["repeats"]), 3)
        self.assertEqual(cold["tokens_and_cache"]["repeats"][0]["scope"]["baseline"]["cache"], "rolling-window; not run-attributed")
        self.assertEqual(cold["run_attributed_telemetry"]["metrics"]["input_tokens_per_accepted_part"]["status"], "unavailable")
        self.assertIn("cpu_ms_per_second", result["idle_check"]["baseline_reports"][0])
        self.assertIn("memory_bytes", result["idle_check"]["baseline_reports"][0])
        self.assertIn("unverified", result["acceptance"])
        gate_matrix = result["showcase_gate_matrix"]
        self.assertEqual(gate_matrix["status"], "incomplete_or_blocked")
        self.assertFalse(gate_matrix["performance_claim_allowed"])
        self.assertFalse(gate_matrix["provider_model_scope_signature_and_admission_verified"])
        self.assertIn("phase:cold:client_latency_p95:candidate_threshold_not_met", gate_matrix["blockers"])
        self.assertIn("phase:cold:run_correlated_token_cache_metrics_unavailable", gate_matrix["reporting_gaps"])

    def test_stage_receipts_flow_through_compare_manifest_and_replay(self) -> None:
        plan = self.make_manifest_plan()
        plan_data = json.loads(plan.read_text())
        entry = plan_data["phase_receipts"]["warm"][0]
        original_pair = json.loads(Path(entry["receipt"]).read_text())
        input_paths = {role: Path(path) for role, path in entry["inputs"].items()}
        reports = {arm: json.loads(input_paths[arm + "_report"].read_text()) for arm in ("baseline", "candidate")}
        query_pairs = [{"case_id": row["case_id"], "query_sha256": row["query_sha256"]} for row in reports["baseline"]["cases"]]
        modulus, exponent, private_exponent = test_rsa_keypair()
        modulus_hex = format(modulus, "x")
        fingerprint = scope_receipts.public_key_fingerprint("test-stage-owner", "test-stage-key", modulus_hex, exponent)
        admission = {"schema_version": 1, "receipt_id": "test-stage-admission", "purpose": stage_telemetry.STAGE_PURPOSE,
            "issuer": "test-stage-owner", "key_id": "test-stage-key", "key_fingerprint_sha256": fingerprint,
            "approval_status": "approved", "owner_acknowledgement_ref": "synthetic-owner-ack",
            "manager_review_ref": "synthetic-manager-review", "reviewer_ids": ["reviewer-a", "reviewer-b"]}
        admission_path = write_json(self.root / "stage-admission.json", admission)
        admission_hash = hashlib.sha256(admission_path.read_bytes()).hexdigest()
        stage_policy = {"schema_version": 1, "keys": [{"issuer": "test-stage-owner", "key_id": "test-stage-key",
            "modulus_hex": modulus_hex, "public_exponent": exponent, "enabled": True,
            "purpose": stage_telemetry.STAGE_PURPOSE, "key_fingerprint_sha256": fingerprint,
            "admission_receipt_id": admission["receipt_id"], "admission_receipt_sha256": admission_hash,
            "reviewer_ids": admission["reviewer_ids"]}]}
        trace_paths = {}
        evidence = [{"receipt_id": admission["receipt_id"], "path": str(admission_path), "sha256": admission_hash}]
        scope_path = write_json(self.root / "stage-resource-scope.json", {
            "method_id": "owner-cgroup-v1", "fixture_only": True,
            "scope": "synthetic isolated stage worker accounting; not a serving measurement"})
        scope_hash = hashlib.sha256(scope_path.read_bytes()).hexdigest()
        scope_id = "stage-resource-scope-1"
        evidence.append({"receipt_id": scope_id, "path": str(scope_path), "sha256": scope_hash})
        for arm in ("baseline", "candidate"):
            durations = 20 if arm == "baseline" else 10
            coverage = [{**row, "stages": {stage_id: {"executed": True, "duration_ms": durations,
                "cpu_ms": 12 if arm == "baseline" else 6, "peak_memory_bytes": 4096 if arm == "baseline" else 3072,
                "resource_method_id": "owner-cgroup-v1", "resource_scope_receipt_id": scope_id,
                "resource_scope_uri": "owner://stage-resource-scope/1", "resource_scope_sha256": scope_hash}
                                                for stage_id in stage_telemetry.STAGE_IDS}}
                        for row in query_pairs]
            export = {"schema_version": 1, "run_id": "warm-1", "arm": arm, "phase": "warm",
                      "complete": True, "truncated": False, "case_coverage": coverage}
            source_path = write_json(self.root / f"{arm}-stage-source.json", {"synthetic_trace_export": arm})
            source = {"receipt_id": f"trace-source-{arm}", "evidence_uri": f"owner://synthetic/{arm}",
                      "evidence_sha256": hashlib.sha256(source_path.read_bytes()).hexdigest()}
            unsigned = stage_telemetry.produce_unsigned_receipt(receipt_id=f"stage-{arm}-warm-1", issuer="test-stage-owner",
                key_id="test-stage-key", run_id="warm-1", arm=arm, phase="warm", query_pairs=query_pairs,
                export=export, source_evidence=source)
            signed = test_sign_payload(unsigned, stage_telemetry, modulus, private_exponent)
            trace_path = write_json(self.root / f"{arm}-stage-receipt.json", signed)
            trace_paths[arm] = trace_path
            evidence.append({"receipt_id": source["receipt_id"], "path": str(source_path), "sha256": source["evidence_sha256"]})
        comparison_path = self.root / "warm-1-stage-comparison.json"
        original_compare_read = compare.read_json
        def compare_reader(path):
            return stage_policy if Path(path).name == "stage-telemetry-trust-v1.json" else original_compare_read(path)
        with mock.patch.object(compare, "read_json", side_effect=compare_reader):
            self.call_compare(input_paths["baseline_report"], input_paths["candidate_report"],
                input_paths["baseline_grade_receipt"], input_paths["candidate_grade_receipt"], comparison_path,
                (trace_paths["baseline"], trace_paths["candidate"]))
        comparison = json.loads(comparison_path.read_text())
        self.assertEqual(comparison["stage_latency"]["stages"]["resolver"]["candidate_to_baseline_total_duration_ratio"], 0.5)
        self.assertEqual(comparison["stage_latency"]["stages"]["resolver"]["resource_attribution"]["cpu_ms_ratio"], 0.5)
        entry["receipt"] = str(comparison_path)
        entry["inputs"]["baseline_stage_trace"] = str(trace_paths["baseline"])
        entry["inputs"]["candidate_stage_trace"] = str(trace_paths["candidate"])
        plan_data["evidence_receipts"].extend(evidence)
        write_json(plan, plan_data)
        bundle_path = self.root / "stage-integrated-manifest.json"
        original_manifest_read = manifest.read_json
        def manifest_reader(path):
            return stage_policy if Path(path).name == "stage-telemetry-trust-v1.json" else original_manifest_read(path)
        with mock.patch.object(manifest, "read_json", side_effect=manifest_reader):
            self.call_manifest(plan, bundle_path)
            self.verify_manifest_bundle(plan, bundle_path)
        bundle = json.loads(bundle_path.read_text())
        warm_rows = bundle["phase_receipts"]["warm"]
        integrated = next(row for row in warm_rows if row["run_id"] == "warm-1")
        self.assertEqual(integrated["source_artifacts"]["inputs"]["baseline_stage_trace"]["name"], trace_paths["baseline"].name)
        self.assertEqual(integrated["stage_latency"]["stages"]["provider"]["candidate_to_baseline_total_duration_ratio"], 0.5)
        self.assertEqual(integrated["stage_latency"]["stages"]["provider"]["resource_attribution"]["peak_memory_ratio"], 0.75)
        self.assertEqual(bundle["evidence_receipts"][scope_id]["sha256"], scope_hash)
        self.assertEqual(comparison["comparison"]["correlated_stage_telemetry"]["baseline"]["resource_scopes"]["provider"]["receipt_id"], scope_id)
        audit = manifest.operator_audit_summary(bundle, hashlib.sha256(bundle_path.read_bytes()).hexdigest())
        audit_stage = audit["phases"]["warm"]["stage_latency"]
        self.assertEqual(audit_stage["resource_attribution_status"], "unavailable_stage_cpu_memory_not_measured")
        self.assertEqual(audit_stage["resource_metrics"]["provider"]["observed_scope_evidence"][0]["receipt_id"], scope_id)
        self.assertEqual(audit_stage["resource_metrics"]["provider"]["metrics"]["cpu_ms"]["status"], "unavailable")
        self.assertFalse(audit["performance_claim_allowed"])
        unindexed_data = json.loads(plan.read_text())
        unindexed_data["evidence_receipts"] = [item for item in unindexed_data["evidence_receipts"] if item["receipt_id"] != scope_id]
        unindexed_plan = write_json(self.root / "stage-unindexed-scope-plan.json", unindexed_data)
        with mock.patch.object(manifest, "read_json", side_effect=manifest_reader), self.assertRaises(SystemExit):
            self.call_manifest(unindexed_plan, self.root / "unindexed-scope-bundle.json")
        self.assertEqual(bundle["phase_budget_rollup"]["phases"]["warm"]["stage_latency"]["status"],
                         "stage_latency_incomplete_or_unavailable")
        self.assertFalse(bundle["showcase_gate_matrix"]["performance_claim_allowed"])

    def test_manifest_keeps_missing_latency_unavailable_and_quality_gate_blocks_floor(self) -> None:
        plan = self.make_manifest_plan()
        data = json.loads(plan.read_text())
        receipt_path = Path(data["phase_receipts"]["warm"][0]["receipt"])
        pair = json.loads(receipt_path.read_text())
        pair["independent_grades"]["acceptance_nonregression_gate"] = False
        pair["comparison"].pop("latency_ms", None)
        write_json(receipt_path, pair)
        output = self.root / "manifest.json"
        self.call_manifest(plan, output)
        warm = json.loads(output.read_text())["phase_budget_rollup"]["phases"]["warm"]
        self.assertFalse(warm["metrics"]["cpu_per_accepted_part"]["confidence_floor_eligible"])
        self.assertFalse(warm["metrics"]["cpu_per_accepted_part"]["conditional_threshold_met"])
        self.assertEqual(warm["metrics"]["client_latency_p95"]["status"], "unavailable")
        self.assertIn("missing or invalid", warm["metrics"]["client_latency_p95"]["reason"])

    def test_manifest_rejects_stage_ratio_without_signed_source_receipts(self) -> None:
        plan = self.make_manifest_plan()
        body = json.loads(plan.read_text())
        pair_path = Path(body["phase_receipts"]["warm"][0]["receipt"])
        pair = json.loads(pair_path.read_text())
        pair["stage_latency"]["status"] = "descriptive_pair_stage_latency"
        pair["stage_latency"]["stages"]["resolver"] = {"status": "descriptive_pair_stage_latency",
                                                           "candidate_to_baseline_total_duration_ratio": 0.5}
        write_json(pair_path, pair)
        output = self.root / "manifest.json"
        with self.assertRaises(SystemExit):
            self.call_manifest(plan, output)
        self.assertFalse(output.exists())

    def test_manifest_records_idle_uncertainty_failure_without_claiming_performance(self) -> None:
        plan = self.make_manifest_plan(candidate_idle_cpu_ms=300)
        output = self.root / "manifest.json"
        self.call_manifest(plan, output)
        result = json.loads(output.read_text())
        self.assertFalse(result["idle_check"]["idle_cpu_within_baseline_uncertainty"])
        self.assertIn("unverified", result["acceptance"])

    def test_idle_equivalence_gate_uses_paired_interval_not_candidate_mean_only(self) -> None:
        plan = self.make_manifest_plan()
        plan_data = json.loads(plan.read_text())
        for arm, values in (("baseline", [80, 100, 120]), ("candidate", [50, 100, 150])):
            for path_value, cpu_ms in zip(plan_data["idle_reports"][arm], values):
                path = Path(path_value)
                report = json.loads(path.read_text())
                report["cgroup_groups"]["api"]["cpu_ms"] = cpu_ms
                write_json(path, report)
        output = self.root / "manifest.json"
        self.call_manifest(plan, output)
        idle = json.loads(output.read_text())["idle_check"]
        # Candidate mean equals baseline mean and would have passed the former
        # mean-in-baseline-CI check, but its paired uncertainty is too wide.
        self.assertEqual(idle["candidate_cpu_ms_per_second_mean"], 10.0)
        self.assertGreater(idle["candidate_minus_baseline_cpu_ms_per_second_ci95"]["ci95_high"],
                           idle["baseline_cpu_uncertainty_margin"])
        self.assertFalse(idle["idle_cpu_within_baseline_uncertainty"])
        self.assertFalse(json.loads(output.read_text())["showcase_gate_matrix"]["performance_claim_allowed"])

    def test_idle_equivalence_gate_accepts_tight_paired_delta_inside_baseline_uncertainty(self) -> None:
        plan = self.make_manifest_plan()
        plan_data = json.loads(plan.read_text())
        for arm, values in (("baseline", [80, 100, 120]), ("candidate", [85, 100, 115])):
            for path_value, cpu_ms in zip(plan_data["idle_reports"][arm], values):
                path = Path(path_value)
                report = json.loads(path.read_text())
                report["cgroup_groups"]["api"]["cpu_ms"] = cpu_ms
                write_json(path, report)
        output = self.root / "manifest.json"
        self.call_manifest(plan, output)
        idle = json.loads(output.read_text())["idle_check"]
        self.assertTrue(idle["idle_cpu_within_baseline_uncertainty"])
        self.assertLessEqual(abs(idle["candidate_minus_baseline_cpu_ms_per_second_ci95"]["ci95_high"]),
                             idle["baseline_cpu_uncertainty_margin"])

    def test_manifest_rejects_overlapping_evidence_windows(self) -> None:
        plan = self.make_manifest_plan(overlap=True)
        with self.assertRaises(SystemExit):
            self.call_manifest(plan, self.root / "manifest.json")
        self.assertFalse((self.root / "manifest.json").exists())

    def test_manifest_rejects_provider_scope_receipt_drift(self) -> None:
        plan = self.make_manifest_plan()
        data = json.loads(plan.read_text())
        second = Path(data["phase_receipts"]["cold"][1]["receipt"])
        pair = json.loads(second.read_text())
        pair["comparison"]["serving_scope_receipt_id"] = "different-owner-scope"
        pair["provider_model_attribution"]["scope_receipt_id"] = "different-owner-scope"
        write_json(second, pair)
        with self.assertRaises(SystemExit):
            self.call_manifest(plan, self.root / "manifest.json")
        self.assertFalse((self.root / "manifest.json").exists())

    def test_manifest_fails_closed_on_malformed_scope_receipt_artifact(self) -> None:
        plan = self.make_manifest_plan()
        data = json.loads(plan.read_text())
        for entries in data["phase_receipts"].values():
            for entry in entries:
                path = Path(entry["receipt"])
                pair = json.loads(path.read_text())
                pair["comparison"]["serving_scope_receipt_id"] = "untrusted-scope"
                pair["provider_model_attribution"]["scope_receipt_id"] = "untrusted-scope"
                write_json(path, pair)
        for paths in data["idle_reports"].values():
            for raw_path in paths:
                path = Path(raw_path)
                report = json.loads(path.read_text())
                report["execution_context"]["serving_scope_receipt_id"] = "untrusted-scope"
                write_json(path, report)
        scope = write_json(self.root / "authority" / "untrusted-scope.json", {"receipt_id": "untrusted-scope"})
        data["evidence_receipts"].append({"receipt_id": "untrusted-scope", "path": str(scope),
                                          "sha256": hashlib.sha256(scope.read_bytes()).hexdigest()})
        plan = write_json(self.root / "plan.json", data)
        with self.assertRaises(SystemExit):
            self.call_manifest(plan, self.root / "manifest.json")
        self.assertFalse((self.root / "manifest.json").exists())

    def test_manifest_keeps_cost_floor_ineligible_without_provider_scope_receipt(self) -> None:
        plan = self.make_manifest_plan()
        data = json.loads(plan.read_text())
        for entries in data["phase_receipts"].values():
            for entry in entries:
                path = Path(entry["receipt"])
                pair = json.loads(path.read_text())
                pair["comparison"]["serving_scope_receipt_id"] = None
                pair["provider_model_attribution"]["scope_receipt_id"] = None
                pair["provider_model_attribution"]["status"] = "missing_owner_scope_receipt"
                write_json(path, pair)
        for paths in data["idle_reports"].values():
            for raw_path in paths:
                path = Path(raw_path)
                report = json.loads(path.read_text())
                report["execution_context"]["serving_scope_receipt_id"] = None
                write_json(path, report)
        plan = write_json(self.root / "plan.json", data)
        output = self.root / "manifest.json"
        self.call_manifest(plan, output)
        result = json.loads(output.read_text())
        metric = result["phase_budget_rollup"]["phases"]["cold"]["metrics"]["cpu_per_accepted_part"]
        self.assertFalse(metric["provider_model_scope_receipt_hash_present"])
        self.assertFalse(metric["confidence_floor_eligible"])
        self.assertFalse(metric["conditional_threshold_met"])

    def test_manifest_rejects_missing_authority_receipt_artifact(self) -> None:
        plan = self.make_manifest_plan(include_authority=False)
        with self.assertRaises(SystemExit):
            self.call_manifest(plan, self.root / "manifest.json")
        self.assertFalse((self.root / "manifest.json").exists())

    def test_manifest_plan_preflight_rejects_unknown_and_unpaired_fields(self) -> None:
        plan = self.make_manifest_plan()
        body = json.loads(plan.read_text())
        output = self.root / "manifest.json"
        body["unreviewed_override"] = True
        write_json(plan, body)
        with self.assertRaises(SystemExit):
            self.call_manifest(plan, output)
        self.assertFalse(output.exists())
        body.pop("unreviewed_override")
        body["phase_receipts"]["warm"][0]["inputs"]["baseline_report_sha256"] = sha("ignored")
        write_json(plan, body)
        with self.assertRaises(SystemExit):
            self.call_manifest(plan, output)
        self.assertFalse(output.exists())
        body["phase_receipts"]["warm"][0]["inputs"].pop("baseline_report_sha256")
        body["concurrency_by_phase"].pop("warm")
        write_json(plan, body)
        with self.assertRaises(SystemExit):
            self.call_manifest(plan, output)
        self.assertFalse(output.exists())

    def test_preflight_report_aggregates_missing_artifacts_and_pending_trust(self) -> None:
        plan = self.make_manifest_plan()
        body = json.loads(plan.read_text())
        body["phase_receipts"]["cold"][0]["inputs"]["baseline_report"] = "missing-baseline-report.json"
        stage_entry = body["phase_receipts"]["warm"][0]
        stage_pair_path = Path(stage_entry["receipt"])
        stage_pair = json.loads(stage_pair_path.read_text())
        for arm in ("baseline", "candidate"):
            artifact = write_json(self.root / f"{arm}-preflight-stage.json", {"synthetic": True})
            role = arm + "_stage_trace"
            stage_entry["inputs"][role] = str(artifact)
            stage_pair["receipt"]["inputs"][role] = {"name": artifact.name,
                                                        "sha256": hashlib.sha256(artifact.read_bytes()).hexdigest()}
        write_json(stage_pair_path, stage_pair)
        write_json(plan, body)
        report_path = self.root / "preflight.json"
        self.call_preflight(plan, report_path)
        report = json.loads(report_path.read_text())
        self.assertEqual(report["plan_shape"], "valid")
        self.assertEqual(report["status"], "blocked_missing_or_mismatched_artifacts")
        self.assertTrue(any(item["role"] == "phase:cold:repeat:1:baseline_report" and item["status"] == "missing"
                            for item in report["artifacts"]))
        self.assertEqual(report["trust_gates"]["telemetry"], "empty_allowlist_owner_admission_pending")
        self.assertEqual(report["trust_gates"]["provider_model_scope"], "empty_allowlist_owner_admission_pending")
        self.assertEqual(report["trust_gates"]["stage_telemetry"], "empty_allowlist_owner_admission_pending")
        self.assertTrue(report["trust_gates"]["stage_receipts_present"])
        self.assertTrue(report["trust_gates"]["stage_receipts_required_together"])
        self.assertTrue(any(item["role"] == "phase:warm:repeat:1:baseline_stage_trace" and item["status"] == "present"
                            for item in report["artifacts"]))
        self.assertFalse(report["trust_gates"]["serving_scope_reference_present"])
        self.assertIn("inventory only", report["claims"])
        self.assertFalse((self.root / "manifest.json").exists())

    def test_integrated_audit_path_builds_replays_and_summarizes_without_claim(self) -> None:
        plan = self.make_manifest_plan()
        bundle_path = self.root / "showcase-bundle.json"
        summary_path = self.root / "showcase-summary.json"
        self.call_audit_manifest(plan, bundle_path, summary_path)
        bundle = json.loads(bundle_path.read_text())
        summary = json.loads(summary_path.read_text())
        self.assertEqual(summary["status"], "bundle_replayed_requires_approved_review")
        self.assertTrue(summary["mechanical_replay_verified"])
        self.assertFalse(summary["performance_claim_allowed"])
        self.assertEqual(summary["bundle"]["sha256"], hashlib.sha256(bundle_path.read_bytes()).hexdigest())
        self.assertEqual(summary["bundle"]["plan_sha256"], bundle["provenance"]["plan"]["sha256"])
        self.assertFalse(summary["provider_model_scope"]["receipt_artifact_hash_present"])
        self.assertEqual(summary["phases"]["cold"]["tokens_and_cache"]["status"], "run_attributed_summary_unavailable; ambient snapshots retained separately and never converted to serving CPU")
        self.assertEqual(summary["phases"]["cold"]["stage_latency"]["status"], "stage_latency_incomplete_or_unavailable")
        self.assertEqual(summary["phases"]["cold"]["stage_latency"]["metrics"]["provider"], "unavailable_missing_or_invalid_stage_receipt")
        self.assertEqual(summary["showcase_gate_matrix"]["phases"]["cold"]["stage_latency_attribution"], "stage_latency_incomplete_or_unavailable")
        self.assertFalse(summary["phases"]["cold"]["cpu_per_accepted_part"]["conditional_threshold_met"])
        self.assertIn("paired candidate-minus-baseline", summary["idle"]["method"])
        self.assertEqual(summary["idle"]["cpu_candidate_minus_baseline_ci95"]["sample_count"], 3)
        self.assertGreaterEqual(summary["idle"]["cpu_baseline_uncertainty_margin"], 0)
        self.assertFalse(summary["showcase_gate_matrix"]["performance_claim_allowed"])
        self.assertEqual(summary["showcase_gate_matrix"]["status"], "incomplete_or_blocked")
        self.assertIn("approved matched-load showcase review", summary["required_review"][-1])


if __name__ == "__main__":
    unittest.main(verbosity=2)
