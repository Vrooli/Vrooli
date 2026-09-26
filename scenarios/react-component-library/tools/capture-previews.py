#!/usr/bin/env python3
"""Capture each released version's anatomy story into preview.png."""

import argparse
import json
import os
from pathlib import Path
from urllib.parse import quote

from playwright.sync_api import sync_playwright


ROOT = Path(__file__).resolve().parents[1]
LIBRARY = ROOT / "library"
RENDERABLE_ASSET_KINDS = {"primitive", "component", "pattern", "page-template", "navigation"}


def versions(asset: str | None, version: str | None, all_versions: bool):
    roots = []
    categories = tuple(path for path in sorted(LIBRARY.iterdir()) if path.is_dir() and path.name != ".retired")
    wanted = asset.strip() if asset else None
    for category in categories:
        if asset:
            name = asset.rsplit(":", 1)[-1].replace("/", "")
            candidate = category / name
            if candidate.is_dir():
                roots.append(candidate)
        else:
            roots.extend(sorted(category.glob("*")))
    found = []
    for root in roots:
        manifest_path = root / "component.json"
        if not manifest_path.is_file():
            continue
        manifest = json.loads(manifest_path.read_text())
        library_id = manifest.get("libraryId", "")
        if manifest.get("assetKind") not in RENDERABLE_ASSET_KINDS:
            continue
        if wanted and wanted not in {library_id, manifest.get("catalogId", "")}:
            continue
        for version_dir in sorted((root / "versions").glob("*")):
            if not version_dir.is_dir() or "-" in version_dir.name:
                continue
            if version and version_dir.name != version:
                continue
            if not all_versions and not asset:
                continue
            source = any(version_dir.glob("*.tsx")) or any(version_dir.glob("*.ts"))
            if source and library_id:
                story_path = version_dir / "story.json"
                if not story_path.is_file():
                    continue
                story_contract = json.loads(story_path.read_text())
                stories = story_contract.get("stories", [])
                anatomy = next((story for story in stories if story.get("role") == "anatomy"), None)
                if anatomy is None:
                    anatomy = next((story for story in stories if story.get("id") == "default"), None)
                if anatomy is None and stories:
                    anatomy = stories[0]
                if not anatomy or not anatomy.get("id"):
                    continue
                found.append((library_id, version_dir.name, version_dir, anatomy["id"]))
    return found


def capture(
    base_url: str,
    library_id: str,
    version: str,
    directory: Path,
    story_id: str,
    page,
    navigation_timeout_ms: int,
    ready_timeout_ms: int,
) -> dict:
    target = directory / "preview.png"
    url = f"{base_url.rstrip('/')}/preview/{quote(library_id, safe='')}/harness.html?story={quote(story_id, safe='')}&version={quote(version)}"
    try:
        response = page.goto(url, wait_until="domcontentloaded", timeout=navigation_timeout_ms)
        if response is not None and response.status >= 400:
            return {
                "asset": library_id,
                "version": version,
                "story": story_id,
                "status": "failed",
                "reason": f"preview returned HTTP {response.status}",
            }
        try:
            page.locator('[data-preview-ready="true"]').first.wait_for(timeout=ready_timeout_ms)
            status = "captured"
            warning = None
        except Exception as ready_error:
            error = page.locator("#preview-error")
            if error.count() and error.is_visible():
                return {
                    "asset": library_id,
                    "version": version,
                    "story": story_id,
                    "status": "failed",
                    "reason": error.inner_text()[:500],
                }
            status = "captured-with-warning"
            warning = f"preview did not signal ready within {ready_timeout_ms}ms: {ready_error}"
        boundary = page.locator("[data-preview-capture-boundary]").first
        if boundary.count():
            boundary.screenshot(path=str(target))
        else:
            page.screenshot(path=str(target), full_page=True)
        result = {"asset": library_id, "version": version, "story": story_id, "status": status, "path": str(target)}
        if warning:
            result["warning"] = warning
        return result
    except Exception as error:
        return {"asset": library_id, "version": version, "story": story_id, "status": "failed", "reason": str(error)}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--asset")
    parser.add_argument("--version")
    parser.add_argument("--all", action="store_true")
    parser.add_argument("--refresh", action="store_true", help="recapture existing preview.png files")
    parser.add_argument("--navigation-timeout-ms", type=int, default=30000)
    parser.add_argument("--ready-timeout-ms", type=int, default=15000)
    parser.add_argument("--base-url", default=os.environ.get("RCL_UI_BASE_URL", "http://127.0.0.1:23906"))
    args = parser.parse_args()
    if not args.all and not args.asset:
        parser.error("provide --asset <library-id> or --all")
    if args.version and not args.asset:
        parser.error("--version requires --asset")
    targets = versions(args.asset, args.version, args.all)
    if not targets:
        print(json.dumps({"captured": 0, "failed": 0, "results": []}))
        return 0
    results = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True, executable_path="/usr/bin/google-chrome")
        page = browser.new_page(viewport={"width": 1280, "height": 900}, device_scale_factor=1)
        for library_id, version, directory, story_id in targets:
            target = directory / "preview.png"
            if not args.refresh and target.is_file() and target.stat().st_size > 0:
                results.append(
                    {
                        "asset": library_id,
                        "version": version,
                        "story": story_id,
                        "status": "skipped",
                        "path": str(target),
                    }
                )
                continue
            results.append(
                capture(
                    args.base_url,
                    library_id,
                    version,
                    directory,
                    story_id,
                    page,
                    args.navigation_timeout_ms,
                    args.ready_timeout_ms,
                )
            )
        browser.close()
    print(
        json.dumps(
            {
                "captured": sum(item["status"].startswith("captured") for item in results),
                "failed": sum(item["status"] == "failed" for item in results),
                "skipped": sum(item["status"] == "skipped" for item in results),
                "results": results,
            },
            indent=2,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
