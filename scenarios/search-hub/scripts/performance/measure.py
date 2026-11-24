#!/usr/bin/env python3
"""Run an equal-work Search Hub load batch and sample attributable Linux processes.

The harness never starts, stops, or reconfigures services. The operator chooses
the target and prepares cold, degraded, or model-free state through the normal
scenario lifecycle. Input is JSONL with one object per line, for example:

    {"case_id":"task-001","query":"find the workflow for ...","group":"workflow-health","limit":5}

The report contains case IDs and query hashes, never query text or result bodies.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


PHASES = ("idle", "cold", "warm", "concurrent", "degraded", "model-free")
DEFAULT_CLI = shutil.which("search-hub") or "search-hub"


@dataclass(frozen=True)
class ProcessGroup:
    name: str
    pattern: re.Pattern[str]


def process_groups(raw: list[str]) -> list[ProcessGroup]:
    groups = []
    names: set[str] = set()
    for item in raw:
        name, separator, expression = item.partition("=")
        if not separator or not name.strip() or not expression:
            raise ValueError(f"invalid --process-group {item!r}; expected NAME=REGEX")
        normalized_name = name.strip()
        if normalized_name in names:
            raise ValueError(f"duplicate process-group name {normalized_name!r}")
        names.add(normalized_name)
        groups.append(ProcessGroup(normalized_name, re.compile(expression)))
    return groups


def read_process(pid_dir: Path) -> tuple[int, int, str, int] | None:
    """Return CPU ticks, RSS, comm and start ticks; ignore exiting processes."""
    try:
        raw = (pid_dir / "stat").read_text()
        close = raw.rfind(")")
        fields = raw[close + 2 :].split()  # begins with proc stat field 3
        comm = (pid_dir / "comm").read_text().strip()
        cpu_ticks = int(fields[11]) + int(fields[12])  # utime + stime (fields 14, 15)
        rss_pages = int(fields[21])  # rss (field 24)
        start_ticks = int(fields[19])  # process start time (field 22)
        page_kib = os.sysconf("SC_PAGE_SIZE") // 1024
        return cpu_ticks, rss_pages * page_kib, comm, start_ticks
    except (OSError, ValueError, IndexError):
        return None


def read_cgroup_path(pid_dir: Path) -> str | None:
    try:
        for line in (pid_dir / "cgroup").read_text().splitlines():
            hierarchy, _, path = line.partition(":")
            if hierarchy == "0":
                return path.split(":", 1)[-1]
    except OSError:
        pass
    return None


def read_cgroup_stats(relative_path: str) -> tuple[int, int] | None:
    """Return (cpu usage usec, current memory bytes) for a cgroup v2 scope."""
    root = Path("/sys/fs/cgroup")
    scope = root / relative_path.lstrip("/")
    try:
        cpu_values = {}
        for line in (scope / "cpu.stat").read_text().splitlines():
            key, _, value = line.partition(" ")
            cpu_values[key] = int(value)
        return cpu_values["usage_usec"], int((scope / "memory.current").read_text().strip())
    except (OSError, ValueError, KeyError):
        return None


def read_pss(pid_dir: Path) -> dict[str, int] | None:
    """Read proportional set size and swap PSS without retaining process maps."""
    values = {}
    try:
        for line in (pid_dir / "smaps_rollup").read_text().splitlines():
            key, separator, value = line.partition(":")
            if separator and key in ("Pss", "Pss_Shmem", "SwapPss"):
                values[key] = int(value.strip().split()[0])
        return values if "Pss" in values else None
    except (OSError, ValueError, IndexError):
        return None


class ProcessSampler:
    """Sample process, exclusive cgroup and selected worker PSS counters."""

    def __init__(self, groups: list[ProcessGroup], cgroup_groups: list[ProcessGroup], pss_groups: list[ProcessGroup], interval_s: float) -> None:
        self.groups = groups
        self.cgroup_groups = cgroup_groups
        self.pss_groups = pss_groups
        self.interval_s = interval_s
        self.stop_event = threading.Event()
        self.thread: threading.Thread | None = None
        self._lock = threading.Lock()
        self._first_ticks: dict[str, dict[tuple[int, int], int]] = {g.name: {} for g in groups}
        self._last_ticks: dict[str, dict[tuple[int, int], int]] = {g.name: {} for g in groups}
        self._peak_rss_kib: dict[str, int] = {g.name: 0 for g in groups}
        self._peak_processes: dict[str, int] = {g.name: 0 for g in groups}
        self._cgroup_first: dict[str, dict[str, int]] = {g.name: {} for g in cgroup_groups}
        self._cgroup_last: dict[str, dict[str, int]] = {g.name: {} for g in cgroup_groups}
        self._cgroup_paths: dict[str, set[str]] = {g.name: set() for g in cgroup_groups}
        self._cgroup_peak_memory: dict[str, int] = {g.name: 0 for g in cgroup_groups}
        self._cgroup_sample_failures: dict[str, int] = {g.name: 0 for g in cgroup_groups}
        self._pss_first_ticks: dict[str, dict[tuple[int, int], int]] = {g.name: {} for g in pss_groups}
        self._pss_last_ticks: dict[str, dict[tuple[int, int], int]] = {g.name: {} for g in pss_groups}
        self._pss_peak: dict[str, dict[str, int]] = {g.name: {"Pss": 0, "Pss_Shmem": 0, "SwapPss": 0} for g in pss_groups}
        self._pss_peak_processes: dict[str, int] = {g.name: 0 for g in pss_groups}
        self._pss_sample_failures: dict[str, int] = {g.name: 0 for g in pss_groups}

    def start(self) -> None:
        self.thread = threading.Thread(target=self._sample_loop, name="search-hub-proc-sampler", daemon=True)
        self.thread.start()

    def stop(self) -> dict[str, dict[str, int]]:
        self.stop_event.set()
        if self.thread:
            self.thread.join(timeout=max(1.0, self.interval_s * 4))
        ticks_per_second = os.sysconf("SC_CLK_TCK")
        with self._lock:
            return {
                name: {
                    "cpu_ms": round(sum(max(0, self._last_ticks[name][identity] - first) for identity, first in values.items()) * 1000 / ticks_per_second),
                    "peak_rss_kib_sum": peak,
                    "peak_process_count": self._peak_processes[name],
                }
                for name, values in self._first_ticks.items()
                for peak in [self._peak_rss_kib[name]]
            }

    def cgroup_result(self) -> dict[str, dict[str, Any]]:
        with self._lock:
            results = {}
            for name, paths in self._cgroup_paths.items():
                unique_paths = self._root_cgroups(paths)
                results[name] = {
                    "available": bool(unique_paths) and self._cgroup_sample_failures[name] == 0,
                    # cgroup v2 cpu.stat usage is reported in microseconds.
                    "cpu_ms": round(sum(max(0, self._cgroup_last[name].get(path, first) - first) for path, first in self._cgroup_first[name].items() if path in unique_paths) / 1000),
                    "peak_memory_current_bytes_sum": self._cgroup_peak_memory[name],
                    "cgroup_v2_paths": [f"/sys/fs/cgroup{path}" for path in unique_paths],
                    "sample_failures": self._cgroup_sample_failures[name],
                }
            return results

    def pss_result(self) -> dict[str, dict[str, Any]]:
        ticks_per_second = os.sysconf("SC_CLK_TCK")
        with self._lock:
            return {
                name: {
                    "available": self._pss_peak_processes[name] > 0 and self._pss_sample_failures[name] == 0,
                    "cpu_available": bool(values),
                    "cpu_ms": round(sum(max(0, self._pss_last_ticks[name].get(identity, first) - first) for identity, first in values.items()) * 1000 / ticks_per_second),
                    "peak_pss_kib_sum": peak["Pss"],
                    "peak_pss_shmem_kib_sum": peak["Pss_Shmem"],
                    "peak_swap_pss_kib_sum": peak["SwapPss"],
                    "peak_process_count": self._pss_peak_processes[name],
                    "sample_failures": self._pss_sample_failures[name],
                }
                for name, values in self._pss_first_ticks.items()
                for peak in [self._pss_peak[name]]
            }

    @staticmethod
    def _root_cgroups(paths: set[str]) -> set[str]:
        ordered = sorted(paths, key=lambda item: (item.count("/"), item))
        roots: list[str] = []
        for path in ordered:
            if not any(path == root or path.startswith(root.rstrip("/") + "/") for root in roots):
                roots.append(path)
        return set(roots)

    def _sample_loop(self) -> None:
        proc_root = Path("/proc")
        while not self.stop_event.is_set():
            totals: dict[str, int] = {g.name: 0 for g in self.groups}
            counts: dict[str, int] = {g.name: 0 for g in self.groups}
            cgroup_candidates: dict[str, set[str]] = {g.name: set() for g in self.cgroup_groups}
            pss_totals: dict[str, dict[str, int]] = {g.name: {"Pss": 0, "Pss_Shmem": 0, "SwapPss": 0} for g in self.pss_groups}
            pss_counts: dict[str, int] = {g.name: 0 for g in self.pss_groups}
            try:
                entries = list(proc_root.iterdir())
            except OSError:
                entries = []
            for entry in entries:
                if not entry.name.isdigit():
                    continue
                proc = read_process(entry)
                if proc is None:
                    continue
                ticks, rss_kib, comm, start_ticks = proc
                for group in self.groups:
                    if group.pattern.search(comm):
                        totals[group.name] += rss_kib
                        counts[group.name] += 1
                        with self._lock:
                            identity = (int(entry.name), start_ticks)
                            self._first_ticks[group.name].setdefault(identity, ticks)
                            previous = self._last_ticks[group.name].get(identity, ticks)
                            if ticks > previous:
                                self._last_ticks[group.name][identity] = ticks
                            else:
                                self._last_ticks[group.name].setdefault(identity, ticks)
                matched_cgroup_groups = [group.name for group in self.cgroup_groups if group.pattern.search(comm)]
                matched_pss_groups = [group.name for group in self.pss_groups if group.pattern.search(comm)]
                if matched_cgroup_groups or matched_pss_groups:
                    pid = int(entry.name)
                    identity = (pid, start_ticks)
                    if matched_cgroup_groups:
                        cgroup_path = read_cgroup_path(entry)
                        if cgroup_path is not None:
                            for name in matched_cgroup_groups:
                                cgroup_candidates[name].add(cgroup_path)
                    if matched_pss_groups:
                        memory = read_pss(entry)
                        for name in matched_pss_groups:
                            with self._lock:
                                self._pss_first_ticks[name].setdefault(identity, ticks)
                                self._pss_last_ticks[name][identity] = ticks
                            if memory is not None:
                                pss_counts[name] += 1
                                for key, value in memory.items():
                                    pss_totals[name][key] += value
                            else:
                                self._pss_sample_failures[name] += 1
            cgroup_memory: dict[str, int] = {g.name: 0 for g in self.cgroup_groups}
            for group in self.cgroup_groups:
                roots = self._root_cgroups(cgroup_candidates[group.name])
                for path in roots:
                    stats = read_cgroup_stats(path)
                    if stats is None:
                        self._cgroup_sample_failures[group.name] += 1
                        continue
                    usage_usec, current_memory = stats
                    cgroup_memory[group.name] += current_memory
                    with self._lock:
                        self._cgroup_paths[group.name].add(path)
                        self._cgroup_first[group.name].setdefault(path, usage_usec)
                        self._cgroup_last[group.name][path] = usage_usec
            with self._lock:
                for group in self.groups:
                    self._peak_rss_kib[group.name] = max(self._peak_rss_kib[group.name], totals[group.name])
                    self._peak_processes[group.name] = max(self._peak_processes[group.name], counts[group.name])
                for group in self.cgroup_groups:
                    self._cgroup_peak_memory[group.name] = max(self._cgroup_peak_memory[group.name], cgroup_memory[group.name])
                for group in self.pss_groups:
                    for key, value in pss_totals[group.name].items():
                        self._pss_peak[group.name][key] = max(self._pss_peak[group.name][key], value)
                    self._pss_peak_processes[group.name] = max(self._pss_peak_processes[group.name], pss_counts[group.name])
            self.stop_event.wait(self.interval_s)


def load_cases(path: Path) -> tuple[list[dict[str, Any]], str]:
    raw = path.read_bytes()
    cases: list[dict[str, Any]] = []
    case_ids: set[str] = set()
    for line_number, line in enumerate(raw.decode("utf-8").splitlines(), 1):
        if not line.strip():
            continue
        try:
            item = json.loads(line)
        except json.JSONDecodeError as exc:
            raise ValueError(f"invalid JSONL at line {line_number}: {exc}") from exc
        if not isinstance(item, dict) or not str(item.get("query", "")).strip():
            raise ValueError(f"line {line_number} must be an object with a non-empty query")
        item["case_id"] = str(item.get("case_id", f"case-{len(cases) + 1}"))
        if item["case_id"] in case_ids:
            raise ValueError(f"duplicate case_id {item['case_id']!r} at line {line_number}")
        case_ids.add(item["case_id"])
        cases.append(item)
    if not cases:
        raise ValueError("workload has no cases")
    return cases, hashlib.sha256(raw).hexdigest()


def command_for(args: argparse.Namespace, case: dict[str, Any]) -> list[str]:
    command = [args.cli]
    if args.api_base:
        command += ["--api-base", args.api_base]
    if args.instance:
        command += ["--instance", args.instance]
    command += ["query", "query", str(case["query"]), "--json"]
    for key, flag in (("type", "--type"), ("group", "--group"), ("scope", "--scope"), ("limit", "--limit")):
        value = case.get(key)
        if value is not None and value != "":
            command.extend((flag, str(value)))
    if case.get("all"):
        command.append("--all")
    if case.get("explain"):
        command.append("--explain")
    return command


def summarize_response(stdout: bytes) -> dict[str, Any]:
    summary: dict[str, Any] = {"response_bytes": len(stdout)}
    try:
        payload = json.loads(stdout)
    except (UnicodeDecodeError, json.JSONDecodeError):
        summary["json_response"] = False
        return summary
    summary["json_response"] = True
    if isinstance(payload, dict):
        for key in ("latencyMs", "latency_ms", "degraded", "partial", "reranked", "rerankerLeg", "selectorLeg", "orderedBy"):
            if key in payload:
                summary[key] = payload[key]
        parts = payload.get("queryParts") or payload.get("query_parts") or []
        if isinstance(parts, list):
            statuses: dict[str, int] = {}
            searched_providers: set[str] = set()
            evidence_count = 0
            degraded_count = 0
            partial_count = 0
            for part in parts:
                if not isinstance(part, dict):
                    continue
                raw_status = str(part.get("status") or "unknown")
                status = raw_status.removeprefix("STATUS_").lower()
                statuses[status] = statuses.get(status, 0) + 1
                degraded_count += int(bool(part.get("degraded")))
                partial_count += int(bool(part.get("partial")))
                evidence = part.get("evidence") or []
                evidence_count += len(evidence) if isinstance(evidence, list) else 0
                provider_ids = part.get("searchedProviderIds") or part.get("searched_provider_ids") or []
                if isinstance(provider_ids, list):
                    searched_providers.update(str(value) for value in provider_ids if value)
            if parts:
                summary["response_part_summary"] = {
                    "count": sum(statuses.values()),
                    "statuses": statuses,
                    "degraded_count": degraded_count,
                    "partial_count": partial_count,
                    "evidence_reference_count": evidence_count,
                    "searched_provider_count": len(searched_providers),
                }
        degradations = payload.get("degradations") or []
        if isinstance(degradations, list) and degradations:
            counts: dict[str, int] = {}
            for degradation in degradations:
                if not isinstance(degradation, dict):
                    continue
                key = "/".join(str(degradation.get(field) or "unknown") for field in ("component", "reason"))
                counts[key] = counts.get(key, 0) + 1
            summary["response_degradation_counts"] = counts
        candidates = payload.get("correctionCandidates") or payload.get("correction_candidates") or []
        if isinstance(candidates, list):
            summary["correction_candidate_count"] = len(candidates)
        groups = payload.get("groups") or []
        ranked = payload.get("ranked") or []
        summary["provider_groups"] = len(groups) if isinstance(groups, list) else None
        summary["result_count"] = len(ranked) if isinstance(ranked, list) and ranked else sum(
            len(group.get("hits") or []) for group in groups if isinstance(group, dict)
        ) if isinstance(groups, list) else None
    return summary


def system_context() -> dict[str, Any]:
    context: dict[str, Any] = {"logical_cpu_count": os.cpu_count()}
    try:
        context["load_average"] = [float(value) for value in Path("/proc/loadavg").read_text().split()[:3]]
    except (OSError, ValueError):
        pass
    try:
        memory = {}
        for line in Path("/proc/meminfo").read_text().splitlines():
            key, _, value = line.partition(":")
            if key in ("MemTotal", "MemAvailable", "SwapTotal", "SwapFree"):
                memory[key] = int(value.strip().split()[0])
        context["memory_kib"] = memory
    except (OSError, ValueError, IndexError):
        pass
    for pressure in ("cpu", "memory"):
        try:
            context[f"{pressure}_pressure"] = Path(f"/proc/pressure/{pressure}").read_text().strip()
        except OSError:
            pass
    return context


def cache_snapshot(api_base: str, timeout_s: float) -> dict[str, Any]:
    """Read a rolling 15-minute cache/query counter snapshot."""
    started = time.perf_counter()
    if not api_base:
        return {"available": False, "error": "metrics_api_base_not_supplied"}
    request = urllib.request.Request(
        api_base.rstrip("/") + "/vrooli.search_hub.v1.metrics.MetricsService/Insights",
        data=b'{"window":"15m"}',
        headers={"Content-Type": "application/json", "Connect-Protocol-Version": "1"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=timeout_s) as response:
            payload = json.loads(response.read())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, OSError) as exc:
        return {"available": False, "error": type(exc).__name__, "snapshot_latency_ms": round((time.perf_counter() - started) * 1000, 3)}
    if not isinstance(payload, dict) or any(key not in payload for key in ("totalQueries", "resolverCacheHits", "resolverCacheMisses")):
        return {"available": False, "error": "invalid_metrics_shape", "snapshot_latency_ms": round((time.perf_counter() - started) * 1000, 3)}
    fields = ("totalQueries", "resolverCacheHits", "resolverCacheMisses", "resolverCacheHitRate")
    return {"available": True, "snapshot_latency_ms": round((time.perf_counter() - started) * 1000, 3), **{key: payload.get(key) for key in fields}}


def gateway_snapshot(cli: str, limit: int, timeout_s: float) -> dict[str, Any]:
    """Read metadata-only AI Gateway events attributed to Search Hub."""
    started = time.perf_counter()
    try:
        result = subprocess.run(
            [cli, "routing", "evidence-list", "--scenario", "search-hub", "--limit", str(limit), "--json"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=timeout_s,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        return {"available": False, "error": type(exc).__name__, "snapshot_latency_ms": round((time.perf_counter() - started) * 1000, 3)}
    if result.returncode != 0:
        return {"available": False, "error": "cli_exit", "stderr_bytes": len(result.stderr), "snapshot_latency_ms": round((time.perf_counter() - started) * 1000, 3)}
    try:
        payload = json.loads(result.stdout)
        events = payload["events"]
        if not isinstance(events, list):
            raise ValueError("events must be a list")
    except (KeyError, TypeError, ValueError, json.JSONDecodeError):
        return {"available": False, "error": "invalid_json_shape", "snapshot_latency_ms": round((time.perf_counter() - started) * 1000, 3)}
    rows = [event for event in events if event.get("scenario") == "search-hub"]
    return {"available": True, "events": rows, "truncated": len(events) >= limit, "snapshot_latency_ms": round((time.perf_counter() - started) * 1000, 3)}


def gateway_delta(before: dict[str, Any], after: dict[str, Any]) -> dict[str, Any]:
    if not before.get("available") or not after.get("available"):
        return {"available": False, "before": before, "after": after}
    prior = {event.get("event_id") for event in before["events"]}
    fresh = [event for event in after["events"] if event.get("event_id") not in prior]
    truncated = before["truncated"] or after["truncated"]
    usage_present = not truncated and all("input_tokens" in event and "output_tokens" in event for event in fresh)
    return {
        "available": True,
        "before_event_count": len(before["events"]),
        "after_event_count": len(after["events"]),
        "delta_event_count": len(fresh),
        "truncated": truncated,
        "token_usage_complete": usage_present,
        "input_tokens": sum(int(event.get("input_tokens", 0)) for event in fresh) if usage_present else None,
        "output_tokens": sum(int(event.get("output_tokens", 0)) for event in fresh) if usage_present else None,
        "events": [
            {key: event.get(key) for key in ("event_id", "role", "selected_model", "status", "created_at", "input_tokens", "output_tokens", "latency_ms")}
            for event in fresh
        ],
    }


def run_case(args: argparse.Namespace, case: dict[str, Any]) -> dict[str, Any]:
    started = time.perf_counter()
    try:
        if args.transport == "cli":
            result = subprocess.run(command_for(args, case), stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=args.timeout_s, check=False)
            stdout = result.stdout
            exit_code = result.returncode
            stderr_bytes = len(result.stderr)
            status_code = None
        else:
            query = urllib.request.Request(
                args.api_base.rstrip("/") + "/vrooli.search_hub.v1.routing.RoutingService/Query",
                data=json.dumps(connect_query_payload(case)).encode(),
                headers={"Content-Type": "application/json", "Connect-Protocol-Version": "1"},
                method="POST",
            )
            with urllib.request.urlopen(query, timeout=args.timeout_s) as response:
                stdout = response.read()
                status_code = response.status
            exit_code = 0
            stderr_bytes = 0
        elapsed_ms = round((time.perf_counter() - started) * 1000, 3)
        return {
            "case_id": case["case_id"],
            "query_sha256": hashlib.sha256(str(case["query"]).encode()).hexdigest(),
            "wall_ms": elapsed_ms,
            "exit_code": exit_code,
            "http_status": status_code,
            "stderr_bytes": stderr_bytes,
            **summarize_response(stdout),
        }
    except subprocess.TimeoutExpired:
        return {
            "case_id": case["case_id"],
            "query_sha256": hashlib.sha256(str(case["query"]).encode()).hexdigest(),
            "wall_ms": round((time.perf_counter() - started) * 1000, 3),
            "exit_code": None,
            "timed_out": True,
        }
    except urllib.error.HTTPError as exc:
        return {
            "case_id": case["case_id"],
            "query_sha256": hashlib.sha256(str(case["query"]).encode()).hexdigest(),
            "wall_ms": round((time.perf_counter() - started) * 1000, 3),
            "exit_code": None,
            "http_status": exc.code,
            "error_body_bytes": len(exc.read()),
        }
    except (urllib.error.URLError, TimeoutError):
        return {
            "case_id": case["case_id"],
            "query_sha256": hashlib.sha256(str(case["query"]).encode()).hexdigest(),
            "wall_ms": round((time.perf_counter() - started) * 1000, 3),
            "exit_code": None,
            "transport_error": "connect_failed_or_timed_out",
        }
    except OSError as exc:
        return {
            "case_id": case["case_id"],
            "query_sha256": hashlib.sha256(str(case["query"]).encode()).hexdigest(),
            "wall_ms": round((time.perf_counter() - started) * 1000, 3),
            "exit_code": None,
            "launch_error": type(exc).__name__,
        }


def connect_query_payload(case: dict[str, Any]) -> dict[str, Any]:
    payload: dict[str, Any] = {"query": str(case["query"])}
    if case.get("type"):
        payload["types"] = [value.strip() for value in str(case["type"]).split(",") if value.strip()]
    for key in ("group", "scope"):
        if case.get(key) not in (None, ""):
            payload[key] = str(case[key])
    if case.get("limit") is not None:
        payload["limit"] = int(case["limit"])
    if case.get("all"):
        payload["all"] = True
    if case.get("explain"):
        payload["explain"] = True
    return payload


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workload", type=Path, help="approved frozen JSONL task/query cohort; omitted for idle phase")
    parser.add_argument("--output", required=True, type=Path, help="measurement JSON output path")
    parser.add_argument("--phase", required=True, choices=PHASES, help="measurement phase; does not configure service state")
    parser.add_argument("--run-id", required=True, help="stable identifier pairing equivalent arms")
    parser.add_argument("--arm", required=True, help="build/configuration label, such as baseline or candidate")
    parser.add_argument("--cli", default=DEFAULT_CLI, help="search-hub CLI executable")
    parser.add_argument("--transport", choices=("cli", "connect"), default="cli", help="ordinary CLI or direct Connect API route")
    parser.add_argument("--api-base", default="", help="approved API endpoint; omit to use CLI routing")
    parser.add_argument("--metrics-api-base", default="", help="Search Hub API base for resolver-cache snapshots; commonly the same as --api-base")
    parser.add_argument("--gateway-cli", default=shutil.which("ai-gateway") or "ai-gateway", help="AI Gateway CLI for model token evidence")
    parser.add_argument("--gateway-evidence-limit", type=int, default=100, help="recent AI Gateway evidence rows to snapshot; API maximum is 100")
    parser.add_argument("--instance", default="", help="scenario instance name, such as shadow")
    parser.add_argument("--build-id", default="", help="externally verified serving build identity")
    parser.add_argument("--corpus-snapshot-id", default="", help="approved fixed corpus snapshot identity")
    parser.add_argument("--fault-schedule-id", default="", help="fault schedule receipt ID; use 'none' for normal path")
    parser.add_argument("--state-receipt-id", default="", help="owner-issued receipt ID for prepared phase state")
    parser.add_argument("--concurrency", type=int, default=1)
    parser.add_argument("--timeout-s", type=float, default=40)
    parser.add_argument("--evidence-timeout-s", type=float, default=10, help="timeout for optional cache and token snapshots")
    parser.add_argument("--sample-ms", type=float, default=1000)
    parser.add_argument("--idle-seconds", type=float, default=60, help="sampling duration when --phase idle is selected")
    parser.add_argument("--process-group", action="append", default=[], metavar="NAME=COMM_REGEX", help="repeat for Search Hub, providers, model workers; matches /proc comm")
    parser.add_argument("--cgroup-group", action="append", default=[], metavar="NAME=COMM_REGEX", help="sample matching processes' cgroup-v2 CPU/current-memory scopes")
    parser.add_argument("--pss-group", action="append", default=[], metavar="NAME=COMM_REGEX", help="sample matching workers' PSS/SwapPss and process CPU ticks")
    parser.add_argument("--note", action="append", default=[], help="record external fault/resource/telemetry evidence references")
    args = parser.parse_args()
    if args.concurrency < 1 or args.concurrency > 256:
        parser.error("--concurrency must be in [1,256]")
    if args.transport == "connect" and not args.api_base:
        parser.error("--transport connect requires --api-base")
    if args.phase == "concurrent" and args.concurrency < 2:
        parser.error("--phase concurrent requires --concurrency of at least 2")
    if args.phase != "idle" and not all((args.build_id, args.corpus_snapshot_id, args.fault_schedule_id, args.state_receipt_id)):
        parser.error("non-idle runs require --build-id, --corpus-snapshot-id, --fault-schedule-id, and --state-receipt-id")
    if args.timeout_s <= 0 or args.evidence_timeout_s <= 0 or args.sample_ms < 20 or args.idle_seconds <= 0:
        parser.error("--timeout-s, --evidence-timeout-s and --idle-seconds must be positive; --sample-ms must be at least 20")
    if args.gateway_evidence_limit < 1 or args.gateway_evidence_limit > 100:
        parser.error("--gateway-evidence-limit must be in [1,100]")
    if args.output.exists():
        parser.error(f"output already exists; choose a new evidence path: {args.output}")
    try:
        if args.workload:
            cases, workload_hash = load_cases(args.workload)
        elif args.phase == "idle":
            cases, workload_hash = [], hashlib.sha256(b"").hexdigest()
        else:
            parser.error("--workload is required for a non-idle phase")
        groups = process_groups(args.process_group)
        cgroup_groups = process_groups(args.cgroup_group)
        pss_groups = process_groups(args.pss_group)
    except (OSError, ValueError, re.error) as exc:
        parser.error(str(exc))

    evidence_timeout_s = min(args.timeout_s, args.evidence_timeout_s)
    cache_before = cache_snapshot(args.metrics_api_base or args.api_base, evidence_timeout_s)
    gateway_before = gateway_snapshot(args.gateway_cli, args.gateway_evidence_limit, evidence_timeout_s)
    harness_cpu_started = time.process_time()
    sampler = ProcessSampler(groups, cgroup_groups, pss_groups, args.sample_ms / 1000)
    sampler.start()
    started_at = time.time()
    start_context = system_context()
    if args.phase == "idle":
        time.sleep(args.idle_seconds)
        rows = []
    elif args.concurrency == 1:
        rows = [run_case(args, case) for case in cases]
    else:
        with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as pool:
            rows = list(pool.map(lambda case: run_case(args, case), cases))
    ended_at = time.time()
    end_context = system_context()
    process_stats = sampler.stop()
    cgroup_stats = sampler.cgroup_result()
    pss_stats = sampler.pss_result()
    cache_after = cache_snapshot(args.metrics_api_base or args.api_base, evidence_timeout_s)
    gateway_after = gateway_snapshot(args.gateway_cli, args.gateway_evidence_limit, evidence_timeout_s)
    cache_delta: dict[str, Any] = {"available": False, "before": cache_before, "after": cache_after}
    if cache_before.get("available") and cache_after.get("available"):
        try:
            query_delta = int(cache_after["totalQueries"]) - int(cache_before["totalQueries"])
            hit_delta = int(cache_after["resolverCacheHits"]) - int(cache_before["resolverCacheHits"])
            miss_delta = int(cache_after["resolverCacheMisses"]) - int(cache_before["resolverCacheMisses"])
            if min(query_delta, hit_delta, miss_delta) < 0:
                raise ValueError("rolling counters decreased")
            cache_delta = {
                "available": True,
                "window_query_delta": query_delta,
                "resolver_cache_hit_delta": hit_delta,
                "resolver_cache_miss_delta": miss_delta,
                "resolver_cache_hit_rate": hit_delta / (hit_delta + miss_delta) if hit_delta + miss_delta > 0 else None,
                "scope": "difference of rolling 15-minute Insights snapshots; unrelated Search Hub queries can contaminate it, and runs longer than 15 minutes are not comparable",
            }
        except (TypeError, ValueError) as exc:
            cache_delta = {
                "available": False,
                "before": cache_before,
                "after": cache_after,
                "error": "rolling_counters_decreased" if "decreased" in str(exc) else "invalid_counter",
            }
    gateway_usage = gateway_delta(gateway_before, gateway_after)

    successful = [row for row in rows if row.get("exit_code") == 0]
    latencies = sorted(float(row["wall_ms"]) for row in rows if "wall_ms" in row)
    api_latencies = sorted(
        float(row[key])
        for row in rows
        for key in ("latencyMs", "latency_ms")
        if key in row and isinstance(row[key], (int, float, str)) and str(row[key]).replace(".", "", 1).isdigit()
    )
    response_sizes = sorted(float(row["response_bytes"]) for row in rows if isinstance(row.get("response_bytes"), (int, float)))

    def percentile(values: list[float], fraction: float) -> float | None:
        if not values:
            return None
        index = min(len(values) - 1, max(0, int((len(values) - 1) * fraction + 0.5)))
        return values[index]

    def distribution(values: list[float]) -> dict[str, float | None]:
        return {
            "p50": percentile(values, 0.50),
            "p95": percentile(values, 0.95),
            "max": max(values) if values else None,
        }

    report = {
        "schema_version": 1,
        "run_id": args.run_id,
        "arm": args.arm,
        "phase": args.phase,
        "started_at_unix_ms": round(started_at * 1000),
        "ended_at_unix_ms": round(ended_at * 1000),
        "target": {"api_base": args.api_base or "cli-default", "instance": args.instance or "live"},
        "execution_context": {
            "build_id": args.build_id or None,
            "corpus_snapshot_id": args.corpus_snapshot_id or None,
            "fault_schedule_id": args.fault_schedule_id or None,
            "state_receipt_id": args.state_receipt_id or None,
        },
        "transport": args.transport,
        "workload": {"sha256": workload_hash, "cases": len(cases)},
        "concurrency": args.concurrency,
        "sample_interval_ms": args.sample_ms,
        "harness_process_cpu_ms": round((time.process_time() - harness_cpu_started) * 1000, 3),
        "process_group_patterns": [{"name": group.name, "comm_regex": group.pattern.pattern} for group in groups],
        "cgroup_group_patterns": [{"name": group.name, "comm_regex": group.pattern.pattern} for group in cgroup_groups],
        "pss_group_patterns": [{"name": group.name, "comm_regex": group.pattern.pattern} for group in pss_groups],
        "idle_seconds": args.idle_seconds if args.phase == "idle" else None,
        "system_context": {"start": start_context, "end": end_context},
        "counts": {"completed": len(rows), "successful_calls": len(successful), "failed_calls": len(rows) - len(successful), "timed_out": sum(bool(row.get("timed_out")) for row in rows)},
        "latency_ms": distribution(latencies),
        "api_response_latency_ms": distribution(api_latencies),
        "response_bytes": distribution(response_sizes),
        "process_groups": process_stats,
        "cgroup_groups": cgroup_stats,
        "pss_groups": pss_stats,
        "cache_tokens_and_grades": {
            "search_hub_cache": cache_delta,
            "ai_gateway_tokens": gateway_usage,
            "grading": "attach SH1 independent part grades and task acceptance before comparing cost per accepted result",
            "query_hashes": [row["query_sha256"] for row in rows],
        },
        "notes": args.note,
        "cases": rows,
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"output": str(args.output), "cases": len(rows), "successful": len(successful), "wall_p50_ms": percentile(latencies, 0.50), "wall_p95_ms": percentile(latencies, 0.95), "api_p95_ms": percentile(api_latencies, 0.95)}, separators=(",", ":")))
    return 0 if len(successful) == len(rows) else 2


if __name__ == "__main__":
    sys.exit(main())
