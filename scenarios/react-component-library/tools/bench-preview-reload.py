#!/usr/bin/env python3
"""Measure save-triggered warm iframe reloads without library mutations.

Run with Python Playwright and Google Chrome installed:

    python3 scenarios/react-component-library/tools/bench-preview-reload.py

The command builds a temporary Go preview server with the production RCL
preview/harness/runtime handlers and a 50 KiB synthetic component in a
temporary source root and SQLite database. It starts Vite for a benchmark host
that imports the production FixtureFrame. A mock PUT validates the source and
returns 204 in memory; timing begins after that acknowledgement and ends when
the real preview harness posts preview-ready and FixtureFrame promotes the
candidate visibly. Temporary processes and files are removed on exit. This is
latency evidence, not BAS capture evidence.
"""

from __future__ import annotations

import argparse
import json
import math
import os
import platform
import selectors
import socket
import statistics
import subprocess
import sys
import tempfile
import time
import urllib.request
from pathlib import Path
from typing import Any

from playwright.sync_api import sync_playwright


ROOT = Path(__file__).resolve().parents[1]
UI_ROOT = ROOT / "ui"
API_ROOT = ROOT / "api"
SOURCE_BYTES = 50 * 1024
ASSET_ID = "react-component-library:ReloadBenchmark"


def synthetic_source() -> str:
    prefix = f'''/** @libraryId {ASSET_ID}
 * @version 1.0.0
 */
import React from "react";
const padding = "'''
    suffix = '''";
export const ReloadBenchmark = () => React.createElement("div", { "data-testid": "synthetic-component" }, padding.slice(0, 32));
'''
    filler_bytes = SOURCE_BYTES - len((prefix + suffix).encode("utf-8"))
    if filler_bytes < 0:
        raise RuntimeError("Synthetic source header exceeds 50 KiB")
    source = prefix + ("x" * filler_bytes) + suffix
    if len(source.encode("utf-8")) != SOURCE_BYTES:
        raise RuntimeError("Synthetic source fixture is not exactly 50 KiB")
    return source


def free_port() -> int:
    with socket.socket() as server:
        server.bind(("127.0.0.1", 0))
        return int(server.getsockname()[1])


def read_server_descriptor(process: subprocess.Popen[str], timeout: float = 60) -> dict[str, Any]:
    if process.stdout is None:
        raise RuntimeError("Temporary preview server has no descriptor pipe")
    selector = selectors.DefaultSelector()
    selector.register(process.stdout, selectors.EVENT_READ)
    try:
        if not selector.select(timeout):
            raise TimeoutError("Timed out waiting for the isolated preview server startup")
        line = process.stdout.readline()
    finally:
        selector.close()
    if not line:
        raise RuntimeError(f"Temporary preview server exited with status {process.poll()}")
    return json.loads(line)


def wait_for_http(url: str, process: subprocess.Popen[bytes], timeout: float = 30) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"Vite exited early with status {process.returncode}")
        try:
            with urllib.request.urlopen(url, timeout=1) as response:
                if response.status == 200:
                    return
        except Exception:
            time.sleep(0.1)
    raise TimeoutError(f"Timed out waiting for Vite at {url}")


def run(args: argparse.Namespace) -> dict[str, Any]:
    source = synthetic_source()
    with tempfile.TemporaryDirectory(prefix="rcl-preview-reload-bench-") as temp_dir:
        server_binary = Path(temp_dir) / "preview-reload-bench-server"
        build = subprocess.run(
            [
                "go",
                "build",
                "-tags",
                "previewreloadbench",
                "-o",
                str(server_binary),
                "./cmd/preview-reload-bench-server",
            ],
            cwd=API_ROOT,
            capture_output=True,
            text=True,
            check=False,
        )
        if build.returncode:
            raise RuntimeError(f"Could not build temporary preview server: {build.stderr.strip()}")
        preview_server = subprocess.Popen(
            [str(server_binary)], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True
        )
        vite: subprocess.Popen[bytes] | None = None
        try:
            descriptor = read_server_descriptor(preview_server)
            preview_url = str(descriptor["url"])
            if int(descriptor["source_bytes"]) != SOURCE_BYTES:
                raise RuntimeError("Temporary preview server source fixture has the wrong size")

            # The initial harness compiles and renders once before timing. Each
            # measured save then exercises the actual warm production handler.
            harness_url = f"{preview_url}/preview/{ASSET_ID}/harness.html?story=default&version=1.0.0"
            try:
                with urllib.request.urlopen(harness_url, timeout=15) as response:
                    if response.status != 200:
                        raise RuntimeError(f"Warm-up harness returned HTTP {response.status}")
            except Exception as error:
                body = error.read().decode("utf-8", errors="replace") if hasattr(error, "read") else str(error)
                raise RuntimeError(f"Warm-up harness request failed: {body[:1000]}") from error

            port = free_port()
            vite = subprocess.Popen(
                ["pnpm", "exec", "vite", "--host", "127.0.0.1", "--port", str(port), "--strictPort"],
                cwd=UI_ROOT,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            page_url = f"http://127.0.0.1:{port}/bench-preview-reload.html"
            wait_for_http(page_url, vite)
            with sync_playwright() as playwright:
                browser = playwright.chromium.launch(headless=True, channel="chrome")
                try:
                    page = browser.new_page(viewport={"width": 1100, "height": 800})
                    page.set_default_timeout(args.timeout_ms)
                    browser_errors: list[str] = []
                    page.on("pageerror", lambda error: browser_errors.append(f"pageerror: {error}"))
                    page.on(
                        "console",
                        lambda message: browser_errors.append(f"console {message.type}: {message.text}")
                        if message.type == "error"
                        else None,
                    )

                    def acknowledge_in_memory_save(route: Any) -> None:
                        body = route.request.post_data or ""
                        if len(body.encode("utf-8")) != SOURCE_BYTES or body != source:
                            route.fulfill(status=400, body="synthetic source mismatch")
                            return
                        route.fulfill(status=204, body="")

                    page.route("**/bench-save", acknowledge_in_memory_save)

                    def route_preview(route: Any) -> None:
                        path = route.request.url.split(f"127.0.0.1:{port}", 1)[-1]
                        response = route.fetch(url=f"{preview_url}{path}", timeout=args.timeout_ms)
                        route.fulfill(response=response)

                    page.route("**/preview/**", route_preview)
                    page.goto(page_url, wait_until="domcontentloaded")
                    try:
                        page.wait_for_function(
                            """() => {
                              const frame = document.querySelector('[data-testid="fixture-iframe"]');
                              return document.querySelector('[data-fixture-frame]')?.dataset.fixtureStatus === 'ready'
                                && frame?.style.visibility === 'visible';
                            }"""
                        )
                    except Exception as error:
                        raise RuntimeError(
                            f"Initial isolated API harness did not promote; status="
                            f"{page.locator('[data-fixture-frame]').get_attribute('data-fixture-status')}; "
                            f"browser={browser_errors}"
                        ) from error
                    page.evaluate("window.__rclReloadBenchResults = []")

                    latencies: list[float] = []
                    total = args.warmup + args.samples
                    for index in range(total):
                        page.locator('[data-testid="bench-save"]').click()
                        page.wait_for_function(
                            """() => window.__rclReloadBenchFailure
                              || (window.__rclReloadBenchResults || []).length > 0""",
                            timeout=args.timeout_ms,
                        )
                        failure = page.evaluate("window.__rclReloadBenchFailure || null")
                        if failure:
                            raise RuntimeError(str(failure))
                        sample = page.evaluate("window.__rclReloadBenchResults.shift()")
                        if index >= args.warmup:
                            latencies.append(float(sample["milliseconds"]))

                    browser_version = browser.version
                finally:
                    browser.close()

            ordered = sorted(latencies)
            p95 = ordered[max(0, math.ceil(len(ordered) * 0.95) - 1)]
            return {
                "benchmark": "rcl-preview-reload-v1",
                "result": "pass" if p95 < 1000 else "fail",
                "protocol": {
                    "fixture_bytes": int(descriptor["source_bytes"]),
                    "warmup_samples": args.warmup,
                    "measured_samples": len(latencies),
                    "start": "after in-memory mock save PUT returns HTTP 204, before sourceRevision update",
                    "end": "candidate preview-ready received and candidate iframe promoted visible",
                    "persistent_writes": False,
                    "preview_harness": "production preview handler, esbuild bundle, React runtime, and FixtureFrame with isolated root/database",
                    "save_path": "PUT acknowledged in memory; no library or scenario API writes",
                    "bas_capture_evidence": False,
                },
                "environment": {
                    "platform": platform.platform(),
                    "machine": platform.machine(),
                    "processor": platform.processor() or None,
                    "logical_cpus": os.cpu_count(),
                    "python": platform.python_version(),
                    "node": subprocess.run(["node", "--version"], capture_output=True, text=True, check=True).stdout.strip(),
                    "chrome": browser_version,
                    "isolated_preview_url": preview_url,
                },
                "latency_ms": {
                    "p50": round(statistics.median(ordered), 2),
                    "p95": round(p95, 2),
                    "max": round(max(ordered), 2),
                    "min": round(min(ordered), 2),
                    "samples": [round(value, 2) for value in latencies],
                },
                "target_ms": 1000,
            }
        finally:
            if vite is not None:
                vite.terminate()
                try:
                    vite.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    vite.kill()
                    vite.wait()
            preview_server.terminate()
            try:
                preview_server.wait(timeout=5)
            except subprocess.TimeoutExpired:
                preview_server.kill()
                preview_server.wait()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--samples", type=int, default=30)
    parser.add_argument("--warmup", type=int, default=5)
    parser.add_argument("--timeout-ms", type=int, default=5000)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    if args.samples < 5 or args.warmup < 0 or args.timeout_ms <= 0:
        parser.error("use at least 5 measured samples, nonnegative warmups, and a positive timeout")
    report = run(args)
    rendered = json.dumps(report, indent=2)
    if args.output:
        args.output.write_text(rendered + "\n", encoding="utf-8")
    print(rendered)
    return 0 if report["result"] == "pass" else 1


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as error:
        print(f"benchmark failed: {error}", file=sys.stderr)
        raise SystemExit(2)
