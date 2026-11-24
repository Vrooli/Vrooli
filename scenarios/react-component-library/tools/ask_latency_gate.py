#!/usr/bin/env python3
"""Run the RCL explicit-Ask latency and citation-contract gate."""

import argparse
import json
import math
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

SCENARIO_ROOT = Path(__file__).resolve().parents[1]
SEARCH_CONFIG = SCENARIO_ROOT / ".vrooli" / "search.json"
SEARCH_PATH = "/vrooli.react_component_library.v1.search.SearchService/Search"
ASK_QUERIES = (
    "landing page",
    "plant care app",
    "dashboard",
    "settings panel",
    "checkout flow",
    "profile page",
    "analytics overview",
    "mobile navigation",
    "search results",
    "empty state",
    "loading indicator",
    "error message",
    "calendar view",
    "task manager",
    "finance app",
    "recipe organizer",
    "travel planner",
    "fitness tracker",
    "music player",
    "user onboarding",
)


def percentile(values, percent):
    if not values:
        raise ValueError("percentile requires at least one value")
    if not 0 <= percent <= 100:
        raise ValueError("percent must be between 0 and 100")
    ordered = sorted(values)
    rank = (len(ordered) - 1) * percent / 100
    low = math.floor(rank)
    high = math.ceil(rank)
    return ordered[low] + (ordered[high] - ordered[low]) * (rank - low)


def configured_slo(path=SEARCH_CONFIG):
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    for provider in data.get("providers", []):
        if provider.get("provider_id") == "react-component-library.workbench":
            performance = provider.get("performance", {})
            target = performance.get("ask_p95_ms")
            samples = performance.get("ask_minimum_samples")
            if isinstance(target, bool) or not isinstance(target, (int, float)) or target <= 0:
                raise ValueError("RCL workbench performance.ask_p95_ms must be positive")
            if isinstance(samples, bool) or not isinstance(samples, int) or samples <= 0:
                raise ValueError("RCL workbench performance.ask_minimum_samples must be positive")
            return float(target), samples
    raise ValueError("RCL workbench provider is missing from search config")


def _post_ask(base_url, query, timeout):
    body = json.dumps({"query": query, "limit": 10, "ask": True}).encode("utf-8")
    request = urllib.request.Request(
        base_url.rstrip("/") + SEARCH_PATH,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=timeout) as response:
        if response.status != 200:
            raise RuntimeError("HTTP status %d" % response.status)
        return json.loads(response.read().decode("utf-8"))


def citation_contract_error(payload):
    if not isinstance(payload, dict):
        return "search response is not an object"
    answer = payload.get("answer")
    if not isinstance(answer, dict) or not isinstance(answer.get("text"), str) or not answer["text"].strip():
        return "answer text is missing"
    citations = answer.get("citations")
    if not isinstance(citations, list) or not citations:
        return "answer citations are missing"
    results = payload.get("results")
    if not isinstance(results, list):
        return "supplied results are missing"
    result_ids = {
        result.get("id")
        for result in results
        if isinstance(result, dict) and isinstance(result.get("id"), str) and result.get("id")
    }
    answer_text = answer["text"].strip()
    if answer_text in result_ids or (":" in answer_text and not any(char.isspace() for char in answer_text)):
        return "answer text is an opaque result ID"
    cited_ids = []
    for citation in citations:
        if not isinstance(citation, dict):
            return "answer citation is malformed"
        result_id = citation.get("resultId")
        if not isinstance(result_id, str) or result_id not in result_ids:
            return "answer cites an id outside its supplied results"
        cited_ids.append(result_id)
    if len(cited_ids) != len(set(cited_ids)):
        return "answer repeats a citation"
    return None


def service_took_ms(payload):
    """Read the service's pre-answer retrieval/status duration, if present."""
    value = payload.get("tookMs")
    if isinstance(value, bool):
        return None
    if isinstance(value, int) and value >= 0:
        return value
    if isinstance(value, str):
        try:
            parsed = int(value)
        except ValueError:
            return None
        return parsed if parsed >= 0 else None
    return None


def summarize(values):
    if not values:
        return None
    return {
        "p50_ms": percentile(values, 50),
        "p95_ms": percentile(values, 95),
        "max_ms": max(values),
    }


def run_gate(base_url, timeout, target_ms, minimum_samples, request_fn=_post_ask):
    latencies = []
    measurements = []
    failures = []
    for index, query in enumerate(ASK_QUERIES, 1):
        started = time.perf_counter()
        try:
            payload = request_fn(base_url, query, timeout)
            elapsed_ms = (time.perf_counter() - started) * 1000
            contract_error = citation_contract_error(payload)
            if contract_error:
                failures.append({"index": index, "query": query, "error": contract_error, "elapsed_ms": elapsed_ms})
            else:
                latencies.append(elapsed_ms)
                measurement = {
                    "index": index,
                    "query": query,
                    "elapsed_ms": round(elapsed_ms, 2),
                }
                service_ms = service_took_ms(payload)
                if service_ms is not None:
                    measurement["search_and_status_ms"] = service_ms
                    measurement["post_search_residual_ms"] = round(
                        max(0.0, elapsed_ms - service_ms), 2
                    )
                measurements.append(measurement)
        except (urllib.error.URLError, TimeoutError, OSError, ValueError, RuntimeError) as exc:
            elapsed_ms = (time.perf_counter() - started) * 1000
            failures.append({"index": index, "query": query, "error": str(exc), "elapsed_ms": elapsed_ms})
    p50 = percentile(latencies, 50) if latencies else None
    p95 = percentile(latencies, 95) if latencies else None
    max_ms = max(latencies) if latencies else None
    service_times = [item["search_and_status_ms"] for item in measurements if "search_and_status_ms" in item]
    residual_times = [item["post_search_residual_ms"] for item in measurements if "post_search_residual_ms" in item]
    passed = (
        not failures
        and len(latencies) >= minimum_samples
        and p95 <= target_ms
    )
    return {
        "samples": len(ASK_QUERIES),
        "successful": len(latencies),
        "failed": len(failures),
        "minimum_samples": minimum_samples,
        "p50_ms": p50,
        "p95_ms": p95,
        "max_ms": max_ms,
        "p95_target_ms": target_ms,
        "citation_contract": "all answer citations resolve to unique supplied result ids",
        "timing_breakdown": {
            "service_field": "SearchResponse.tookMs, measured before the AI Gateway answer call",
            "search_and_status": summarize(service_times),
            "post_search_residual": summarize(residual_times),
            "samples_with_service_timing": len(service_times),
        },
        "measurements": measurements,
        "failures": failures,
        "passed": passed,
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://localhost:17193", help="RCL API base URL")
    parser.add_argument("--timeout", type=float, default=25.0, help="per-request timeout in seconds")
    parser.add_argument("--config", type=Path, default=SEARCH_CONFIG, help="search.json path")
    args = parser.parse_args(argv)
    if args.timeout <= 0:
        parser.error("--timeout must be positive")
    try:
        target, minimum_samples = configured_slo(args.config)
        result = run_gate(args.base_url, args.timeout, target, minimum_samples)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(json.dumps({"passed": False, "error": str(exc)}, indent=2), file=sys.stderr)
        return 2
    print(json.dumps(result, indent=2))
    return 0 if result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
