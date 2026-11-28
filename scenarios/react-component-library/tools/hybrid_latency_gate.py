#!/usr/bin/env python3
"""Run a repeatable 100-query live latency gate for RCL hybrid search."""

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
CORPUS_SEEDS = (
    "landing page", "plant care app", "dashboard", "settings panel", "checkout flow",
    "profile page", "analytics overview", "mobile navigation", "search results",
    "empty state", "loading indicator", "error message", "calendar view", "task manager",
    "finance app", "recipe organizer", "travel planner", "fitness tracker", "music player",
    "project board", "user onboarding", "notification center", "product catalog",
    "editorial commerce", "data table",
)
# The ordered repetition keeps corpus size and mix fixed while ensuring exactly 100 samples.
QUERY_CORPUS = tuple(CORPUS_SEEDS[i % len(CORPUS_SEEDS)] for i in range(100))


def percentile(values, percent):
    """Return a linearly interpolated percentile (0 <= percent <= 100)."""
    if not values:
        raise ValueError("percentile requires at least one value")
    if not 0 <= percent <= 100:
        raise ValueError("percent must be between 0 and 100")
    ordered = sorted(values)
    rank = (len(ordered) - 1) * percent / 100
    low = math.floor(rank)
    high = math.ceil(rank)
    return ordered[low] + (ordered[high] - ordered[low]) * (rank - low)


def configured_p95_target(path=SEARCH_CONFIG):
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    providers = data.get("providers", [])
    for provider in providers:
        if provider.get("provider_id") == "react-component-library.workbench":
            value = provider.get("performance", {}).get("p95_ms")
            if isinstance(value, (int, float)) and value > 0:
                return float(value)
    raise ValueError("RCL workbench performance.p95_ms is missing or invalid")


def _post_search(base_url, query, timeout):
    body = json.dumps({"query": query, "limit": 10}).encode("utf-8")
    request = urllib.request.Request(
        base_url.rstrip("/") + SEARCH_PATH, data=body,
        headers={"Content-Type": "application/json"}, method="POST")
    with urllib.request.urlopen(request, timeout=timeout) as response:
        if response.status != 200:
            raise RuntimeError("HTTP status %d" % response.status)
        return json.loads(response.read().decode("utf-8"))


def is_landing_page_workspace_first(payload):
    results = payload.get("results", [])
    if not results:
        return False
    first = results[0]
    # The curated workspace is the concrete, outcome-first result for this
    # query. Accept the stable kind contract, with the current seed title as a
    # fallback for older response encodings that omit kind.
    kind = str(first.get("kind", "")).strip().casefold()
    return kind in {"workspace", "search_kind_workspace"} or (
        not kind and str(first.get("title", "")).strip().casefold() == "marlow")


def run_gate(base_url, timeout, target_ms, request_fn=_post_search):
    latencies = []
    failures = []
    landing_page_workspace_ok = None
    for index, query in enumerate(QUERY_CORPUS, 1):
        started = time.perf_counter()
        try:
            payload = request_fn(base_url, query, timeout)
            elapsed_ms = (time.perf_counter() - started) * 1000
            latencies.append(elapsed_ms)
            if query == "landing page":
                landing_page_workspace_ok = is_landing_page_workspace_first(payload)
        except (urllib.error.URLError, TimeoutError, OSError, ValueError, RuntimeError) as exc:
            elapsed_ms = (time.perf_counter() - started) * 1000
            failures.append({"index": index, "query": query, "error": str(exc), "elapsed_ms": elapsed_ms})
    p50 = percentile(latencies, 50) if latencies else None
    p95 = percentile(latencies, 95) if latencies else None
    max_ms = max(latencies) if latencies else None
    passed = not failures and len(latencies) == 100 and p95 <= target_ms and landing_page_workspace_ok is True
    return {"samples": 100, "successful": len(latencies), "failed": len(failures),
            "p50_ms": p50, "p95_ms": p95, "max_ms": max_ms,
            "p95_target_ms": target_ms, "landing_page_workspace_first": landing_page_workspace_ok,
            "failures": failures, "passed": passed}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://localhost:17193", help="RCL API base URL")
    parser.add_argument("--timeout", type=float, default=10.0, help="per-request timeout in seconds")
    parser.add_argument("--config", type=Path, default=SEARCH_CONFIG, help="search.json path")
    args = parser.parse_args(argv)
    if args.timeout <= 0:
        parser.error("--timeout must be positive")
    try:
        result = run_gate(args.base_url, args.timeout, configured_p95_target(args.config))
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(json.dumps({"passed": False, "error": str(exc)}, indent=2), file=sys.stderr)
        return 2
    print(json.dumps(result, indent=2))
    return 0 if result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
