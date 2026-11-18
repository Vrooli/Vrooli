#!/usr/bin/env python3
"""Regression for the Browser Automation Studio disabled-profile migration."""

from __future__ import annotations

import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[2]
INVENTORY_PATH = ROOT / "scenarios/landing-page-business-suite/.vrooli/presentation-preservation/inventory.json"
MISSING = object()

EXPECTED_SOURCE_CAPTURES = {
    "public-section-hero": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/HeroSection.tsx",
    "public-section-features": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/FeaturesSection.tsx",
    "public-section-pricing": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/PricingSection.tsx",
    "public-section-download": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/DownloadSection.tsx",
    "public-section-faq": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/FAQSection.tsx",
    "public-section-cta": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/CTASection.tsx",
    "public-section-footer": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/FooterSection.tsx",
    "public-section-testimonials": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/TestimonialsSection.tsx",
    "public-section-video": "scenarios/landing-page-business-suite/ui/src/surfaces/public-landing/sections/VideoSection.tsx",
    "fallback-ui-helper": "scenarios/landing-page-business-suite/ui/src/shared/lib/fallbackLandingConfig.ts",
    "fallback-data-source": "scenarios/landing-page-business-suite/.vrooli/fallback/fallback.json",
    "fallback-api-helper": "scenarios/landing-page-business-suite/api/internal/landing/landing_config_fallback.go",
}


def canonical(value: Any) -> bytes:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()


def digest_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def digest(value: Any) -> str:
    return digest_bytes(canonical(value))


def source_sha256(relative_path: str) -> str:
    path = ROOT / relative_path
    assert path.is_file(), f"source is missing: {relative_path}"
    return digest_bytes(path.read_bytes())


def get_path(value: Any, dotted_path: str) -> Any:
    """Read the inventory's dotted paths, including JSON-array [index] segments."""

    tokens = re.findall(r"([^.\[\]]+)|\[(\d+)\]", dotted_path)
    current = value
    for token_pair in tokens:
        token = token_pair[0] or token_pair[1]
        if isinstance(current, list):
            current = current[int(token)]
        elif isinstance(current, dict):
            current = current[token]
        else:
            return MISSING
    return current


def get_json_pointer(value: Any, pointer: str) -> Any:
    """Resolve an inventory-local JSON Pointer used by immutable snapshots."""

    assert pointer.startswith("#/"), f"unsupported immutable payload ref: {pointer}"
    current = value
    for token in pointer[2:].split("/"):
        token = token.replace("~1", "/").replace("~0", "~")
        if isinstance(current, list):
            current = current[int(token)]
        elif isinstance(current, dict):
            current = current[token]
        else:
            raise AssertionError(f"immutable payload ref entered a scalar: {pointer}")
    return current


def leaves(value: Any):
    if isinstance(value, dict):
        for child in value.values():
            yield from leaves(child)
    elif isinstance(value, list):
        for child in value:
            yield from leaves(child)
    else:
        yield value


def assert_equal(actual: Any, expected: Any, message: str) -> None:
    if actual != expected:
        raise AssertionError(message)


def verify_source_captures(inventory: dict[str, Any]) -> list[str]:
    """Verify the independent legacy-source snapshot set.

    The capture bytes, not the current source files, are the recovery authority.
    A live source may be removed or changed by the admin migration; such drift
    is reported in the receipt and does not create a second preservation
    authority.
    """

    manifest_ref = inventory.get("sourceCaptureManifest")
    expected_manifest_ref = "scenarios/landing-page-business-suite/.vrooli/presentation-preservation/source-captures.json"
    assert_equal(manifest_ref, expected_manifest_ref, "source capture manifest ref changed")
    manifest_path = ROOT / expected_manifest_ref
    assert manifest_path.is_file(), f"source capture manifest is missing: {expected_manifest_ref}"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    assert_equal(manifest["schemaVersion"], 1, "unsupported source capture manifest schema")
    assert_equal(manifest["captureId"], "bas-public-sections-source-captures-v1", "unexpected source capture identity")
    assert_equal(manifest["immutable"], True, "source captures must be immutable recovery anchors")
    captures = {capture["id"]: capture for capture in manifest["captures"]}
    assert_equal(set(captures), set(EXPECTED_SOURCE_CAPTURES), "source capture coverage is incomplete or duplicated")

    capture_root = manifest_path.parent.resolve()
    live_source_drift = []
    capture_paths = set()
    for capture_id, expected_source_path in EXPECTED_SOURCE_CAPTURES.items():
        capture = captures[capture_id]
        assert_equal(capture["sourcePath"], expected_source_path, f"source path changed: {capture_id}")
        capture_path = (manifest_path.parent / capture["capturePath"]).resolve()
        assert capture_root in capture_path.parents, f"capture escapes preservation root: {capture_id}"
        assert capture_path not in capture_paths, f"capture path is reused: {capture_id}"
        capture_paths.add(capture_path)
        assert capture_path.is_file(), f"source capture is missing: {capture_id}"
        capture_bytes = capture_path.read_bytes()
        assert_equal(capture["bytes"], len(capture_bytes), f"capture byte count changed: {capture_id}")
        assert_equal(capture["sha256"], digest_bytes(capture_bytes), f"capture digest changed: {capture_id}")

        live_source = ROOT / expected_source_path
        if live_source.is_file() and live_source.read_bytes() != capture_bytes:
            live_source_drift.append(expected_source_path)
    return live_source_drift


def main() -> int:
    inventory = json.loads(INVENTORY_PATH.read_text(encoding="utf-8"))
    profile = inventory["after"]["profile"]
    source_snapshots = inventory["sourceSnapshots"]
    source_capture_live_drift = verify_source_captures(inventory)

    assert_equal(inventory["inventoryId"], "bas-disabled-profile-migration-v1", "unexpected inventory identity")
    assert_equal(profile["appKey"], "browser-automation-studio", "wrong migrated app key")
    assert_equal(profile["enabled"], False, "BAS must remain disabled")
    assert_equal(profile["visibility"], "private", "BAS must remain private")
    assert_equal(profile["publication"], "draft", "BAS must remain a draft")
    assert_equal(profile["publicBundleMembership"], False, "disabled BAS leaked into public membership")
    assert_equal(profile["management"]["pmTeam"], "disabled", "PM team must remain disabled")

    # Source digests make the inventory auditable before a parent renderer
    # migration. Component drift is expected once the parent removes generic
    # BAS defaults; the immutable digest and copied profile must still remain.
    component_source_drift = []
    for snapshot in [source_snapshots["fallback"], source_snapshots["components"]["hero"], source_snapshots["components"]["features"]]:
        source_path = ROOT / snapshot["sourcePath"]
        if not source_path.is_file() or source_sha256(snapshot["sourcePath"]) != snapshot["sourceSha256"]:
            if snapshot["sourcePath"] in {source_snapshots["components"]["hero"]["sourcePath"], source_snapshots["components"]["features"]["sourcePath"]}:
                component_source_drift.append(snapshot["sourcePath"])
            elif source_path.is_file():
                raise AssertionError(f"source digest drifted: {snapshot['sourcePath']}")
    for snapshot in source_snapshots["configVariants"]:
        source_path = ROOT / snapshot["sourcePath"]
        if source_path.is_file() and source_sha256(snapshot["sourcePath"]) != snapshot["sourceSha256"]:
            raise AssertionError(f"variant digest drifted: {snapshot['sourcePath']}")

    # The immutable payload refs point into the inventory itself. The live
    # fallback/config files are only optional source-digest observations; a
    # parent migration may move or rewrite them after this regression lands.
    immutable_refs = inventory["before"]["immutablePayloadRefs"]
    before_payloads = {name: get_json_pointer(inventory, pointer) for name, pointer in immutable_refs.items()}
    assert_equal(set(before_payloads), set(inventory["before"]["sourcePayloadDigests"]), "immutable payload refs do not cover all before payloads")
    before_download = before_payloads["fallback.download"]

    for payload_name, before_payload_digest in inventory["before"]["sourcePayloadDigests"].items():
        recovered = before_payloads[payload_name]
        assert_equal(digest(recovered), before_payload_digest, f"before payload digest mismatch: {payload_name}")

    # The exact before payloads and the after profile must agree at every field.
    assert_equal(profile["availability"], before_download, "fallback BAS availability was rewritten")
    assert_equal(profile["identity"], {key: before_download[key] for key in ("name", "tagline", "description")}, "BAS identity changed")
    assert_equal(profile["marketing"]["pricing"], before_payloads["fallback.pricing"], "fallback pricing material changed")
    for snapshot in source_snapshots["configVariants"]:
        current = before_payloads[f"configVariants.{snapshot['slug']}"]
        preserved = profile["marketing"]["configVariants"][snapshot["slug"]]
        assert_equal(preserved, current, f"variant payload changed: {snapshot['slug']}")

    # Component copy is not executable JSON, so verify every preserved string is still recoverable from its source.
    component_sources = {
        name: (ROOT / snapshot["sourcePath"]).read_text(encoding="utf-8")
        for name, snapshot in source_snapshots["components"].items()
        if (ROOT / snapshot["sourcePath"]).is_file()
    }
    for component_name, component_payload in profile["marketing"]["componentDefaults"].items():
        if component_name not in component_sources:
            continue
        for value in leaves(component_payload):
            if isinstance(value, str) and value and value not in component_sources[component_name]:
                raise AssertionError(f"preserved {component_name} copy is not recoverable from source: {value!r}")

    checked_fields = 0
    ledger_root = {
        "fallback": {"download": before_payloads["fallback.download"], "pricing": before_payloads["fallback.pricing"]},
        "components": {"hero": before_payloads["components.hero"], "features": before_payloads["components.features"]},
        "configVariants": {
            name.split(".", 1)[1]: value
            for name, value in before_payloads.items()
            if name.startswith("configVariants.")
        },
    }
    for entry in inventory["fieldLedger"]:
        assert_equal(entry["disposition"], "exact-preserved", f"field is not marked exact-preserved: {entry['afterPath']}")
        assert entry["sourceSha256"], f"field has no source digest: {entry['afterPath']}"
        before_value = get_path(ledger_root, entry["beforePath"])
        if before_value is MISSING:
            raise AssertionError(f"field ledger points to missing immutable before field: {entry['beforePath']}")
        after_value = get_path({"profile": profile}, entry["afterPath"])
        if after_value is MISSING:
            raise AssertionError(f"field ledger points to missing after field: {entry['afterPath']}")
        assert_equal(after_value, before_value, f"field ledger changed value: {entry['beforePath']} -> {entry['afterPath']}")
        checked_fields += 1
    assert checked_fields >= 1000, f"field ledger is unexpectedly sparse: {checked_fields}"

    # Asset refs are preserved even when their underlying bytes are unavailable.
    assets = inventory["assetLedger"]
    assert assets, "asset ledger is empty"
    for asset in assets:
        assert asset["status"], f"asset status is missing: {asset['id']}"
        assert asset["disposition"], f"asset disposition is missing: {asset['id']}"
    installer_urls = {platform["artifact_url"] for platform in profile["availability"]["platforms"]}
    preserved_installer_urls = {asset["exactRef"] for asset in assets if asset["kind"] == "installer-ref"}
    assert_equal(preserved_installer_urls, installer_urls, "installer refs were not preserved exactly")
    assert any(asset["status"] == "unlocated" for asset in assets), "unlocated screenshots/artifacts were not recorded"
    assert inventory["unlocatedArtifacts"], "remaining unlocated artifacts must be explicit"

    profile_digest = digest(profile)
    assert_equal(profile_digest, inventory["after"]["profileDigestSha256"], "after profile digest mismatch")
    assert_equal(profile_digest, inventory["migrationSnapshots"]["afterProfileDigestSha256"], "after snapshot digest mismatch")
    receipt = {
        "status": "passed",
        "inventory": str(INVENTORY_PATH.relative_to(ROOT)),
        "beforePayloadsRecovered": len(before_payloads),
        "beforePayloadDigestCount": len(inventory["before"]["sourcePayloadDigests"]),
        "afterProfileDigestSha256": profile_digest,
        "fieldsChecked": checked_fields,
        "assetsChecked": len(assets),
        "componentSourceDrift": component_source_drift,
        "sourceCaptureCount": len(EXPECTED_SOURCE_CAPTURES),
        "sourceCaptureLiveDrift": source_capture_live_drift,
        "remaining": inventory["unlocatedArtifacts"],
    }
    print(json.dumps(receipt, ensure_ascii=False, sort_keys=True))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (AssertionError, KeyError, StopIteration, json.JSONDecodeError) as error:
        print(f"BAS preservation regression failed: {error}", file=sys.stderr)
        raise SystemExit(1)
