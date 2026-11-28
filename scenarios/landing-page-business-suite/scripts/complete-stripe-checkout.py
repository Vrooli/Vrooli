#!/usr/bin/env python3
"""Complete a Stripe test-mode hosted Checkout flow using public test cards.

Read the Checkout Session URL from stdin so it does not appear in process argv.
The runner accepts no card data from the caller and emits only sanitized IDs
and status. It does not contact Stripe's API or use credentials.
"""

import argparse
import json
import re
import shutil
import sys
import time
from urllib.parse import urlparse


CARD_BY_OUTCOME = {
    "success": "4242424242424242",
    "decline": "4000000000000002",
    "3ds": "4000002500003155",
}
SESSION_ID_RE = re.compile(r"(cs_(?:test|live)_[A-Za-z0-9]+)")


def parse_url(raw_url):
    parsed = urlparse(raw_url.strip())
    match = SESSION_ID_RE.search(parsed.path)
    if (
        parsed.scheme != "https"
        or parsed.hostname not in {"checkout.stripe.com", "checkout.stripe.dev"}
        or not match
    ):
        raise ValueError("Expected a Stripe-hosted Checkout Session URL.")
    session_id = match.group(1)
    if not session_id.startswith("cs_test_"):
        raise ValueError("Only Stripe test-mode Checkout Sessions are allowed.")
    return session_id


def fill_any(frames, labels, value):
    for frame in frames:
        for label in labels:
            locator = frame.get_by_label(re.compile(label, re.IGNORECASE)).first
            try:
                if locator.is_visible(timeout=400):
                    locator.fill(value, timeout=3000)
                    return True
            except Exception:
                # Checkout fields may be in any of several Stripe-owned frames.
                pass
    return False


def click_any(frames, labels, timeout_ms=5000):
    deadline = time.monotonic() + timeout_ms / 1000
    while time.monotonic() < deadline:
        for frame in frames:
            for label in labels:
                locator = frame.get_by_role("button", name=re.compile(label, re.IGNORECASE)).first
                try:
                    if locator.is_visible(timeout=250):
                        locator.click(timeout=3000)
                        return True
                except Exception:
                    pass
        time.sleep(0.2)
    return False


def read_page_text(page):
    try:
        return page.locator("body").inner_text(timeout=3000).lower()
    except Exception:
        return ""


def run(session_id, raw_url, outcome):
    try:
        from playwright.sync_api import sync_playwright
    except ImportError as exc:
        raise RuntimeError("Python Playwright is unavailable.") from exc

    status = "incomplete"
    with sync_playwright() as playwright:
        launch_options = {"headless": True}
        chrome = shutil.which("google-chrome")
        if chrome:
            launch_options["executable_path"] = chrome
        browser = playwright.chromium.launch(**launch_options)
        try:
            page = browser.new_page()
            page.set_default_timeout(15000)
            page.goto(raw_url, wait_until="domcontentloaded", timeout=45000)
            frames = [page, *page.frames]

            # Stripe Checkout has presented Payment Element fields in the main
            # document or Stripe-owned iframes, so search accessible labels in
            # each frame instead of depending on generated CSS selectors.
            card_filled = fill_any(frames, [r"card number", r"card"], CARD_BY_OUTCOME[outcome])
            expiry_filled = fill_any(frames, [r"expiration", r"expiry", r"mm\s*/\s*yy"], "12/34")
            cvc_filled = fill_any(frames, [r"security code", r"cvc", r"cvv"], "123")
            if not card_filled or not expiry_filled or not cvc_filled:
                raise RuntimeError("Required hosted Checkout payment fields were not found.")

            # These are optional and only filled if the active form asks for them.
            fill_any(frames, [r"email"], "checkout-test@example.com")
            fill_any(frames, [r"name on card", r"full name"], "Checkout Test")
            fill_any(frames, [r"postal code", r"zip"], "94107")

            if not click_any(frames, [r"^pay\b", r"^subscribe\b", r"^start trial\b", r"^complete purchase\b", r"^confirm\b"], 10000):
                raise RuntimeError("Hosted Checkout submit button was not found.")

            if outcome == "3ds":
                # Common explicit actions shown by Stripe's test issuer challenge.
                click_any(
                    [page, *page.frames],
                    [r"complete authentication", r"authenticate", r"authorize", r"complete", r"success"],
                    8000,
                )

            deadline = time.monotonic() + 45
            while time.monotonic() < deadline:
                text = read_page_text(page)
                current_url = page.url
                if outcome == "decline" and re.search(
                    r"declined|was declined|try another|payment failed|card was declined", text
                ):
                    status = "declined"
                    break
                if outcome != "decline" and (
                    re.search(r"/success(?:[/?#]|$)", current_url, re.IGNORECASE)
                    or re.search(r"payment successful|thank you for your payment|order confirmed", text)
                ):
                    status = "succeeded"
                    break
                page.wait_for_timeout(500)
        finally:
            browser.close()
    return {"session_id": session_id, "requested_outcome": outcome, "status": status}


def main():
    parser = argparse.ArgumentParser(
        description="Complete a Stripe test-mode hosted Checkout flow. Read the Checkout URL from stdin."
    )
    parser.add_argument("outcome", choices=tuple(CARD_BY_OUTCOME))
    args = parser.parse_args()

    raw_url = sys.stdin.readline().strip()
    if not raw_url:
        parser.error("Checkout Session URL must be provided on stdin.")
    try:
        session_id = parse_url(raw_url)
        evidence = run(session_id, raw_url, args.outcome)
        print(json.dumps(evidence, separators=(",", ":")))
        return 0 if evidence["status"] != "incomplete" else 2
    except Exception as exc:
        message = str(exc)
        if message.startswith("Expected a Stripe-hosted") or message.startswith("Only Stripe test-mode"):
            print(message, file=sys.stderr)
        else:
            # Never expose URLs, query values, card values, or browser details.
            print("Checkout runner failed; details were suppressed.", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
