"""Phone viewport journey through the real React page and test-mode LPBS route."""
import json
import hashlib
import re
import sys
from playwright.sync_api import expect, sync_playwright

url, result_path, screenshot_path = sys.argv[1:4]
responses = []
browser_errors = []
failed_requests = []

with sync_playwright() as playwright:
    browser = playwright.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True, args=["--no-sandbox"])
    page = browser.new_page(
        viewport={"width": 390, "height": 844}, device_scale_factor=3, is_mobile=True, has_touch=True,
    )
    page.on("console", lambda message: browser_errors.append(f"console {message.type}: {message.text}") if message.type == "error" else None)
    page.on("pageerror", lambda error: browser_errors.append(f"page error: {error}"))
    page.on("requestfailed", lambda request: failed_requests.append(f"{request.method} {request.url}: {request.failure}"))
    page.on("response", lambda response: responses.append({
        "method": response.request.method, "url": response.url, "status": response.status,
        "sha256": response.headers.get("x-content-sha256", ""),
    }))
    page.goto(url, wait_until="domcontentloaded")
    try:
        expect(page.get_by_role("heading", name="Company Market Scan")).to_be_visible()
    except AssertionError:
        print("Browser diagnostics:", json.dumps({"errors": browser_errors, "failed_requests": failed_requests, "body": page.locator("body").inner_text()[:2000]}))
        raise
    expect(page.get_by_text("$49.00 USD", exact=True)).to_be_visible()
    expect(page.get_by_text("Target delivery: 60 minutes")).to_be_visible()
    expect(page.get_by_text("Legal review is open and operator approval is pending", exact=False)).to_be_visible()
    expect(page.get_by_role("link", name="JSON metadata")).to_have_attribute("href", "/services/company-market-scan.json")
    page.get_by_role("link", name="Continue without an account").click()
    expect(page.get_by_role("heading", name="Review and continue")).to_be_visible()
    expect(page.get_by_text("Test mode · no live payment")).to_be_visible()
    expect(page.get_by_text("$49.00", exact=True)).to_be_visible()
    expect(page.get_by_role("region", name="Selected test checkout channel")).to_contain_text("Person in browser")
    page.get_by_role("textbox", name="Market or vendor category").fill("Apple FY2026 public filings")
    page.get_by_role("checkbox", name=re.compile("No funds move")).check()
    page.get_by_role("button", name="Create $49.00 test authorization").click()
    expect(page.get_by_role("button", name="Verify delivery and capture test authorization")).to_be_visible(timeout=15000)

    order_text = page.locator(".site-service-checkout-success p").first.inner_text()
    order_match = re.search(r"Order ([0-9a-f-]+) for job ([0-9a-f-]+)", order_text)
    assert order_match, order_text
    order_id, job_id = order_match.groups()

    page.get_by_role("button", name="Verify delivery and capture test authorization").click()
    expect(page.get_by_role("button", name="Reconcile accepted Money Ledger receipt")).to_be_visible(timeout=15000)
    page.get_by_role("button", name="Reconcile accepted Money Ledger receipt").click()
    expect(page.get_by_role("heading", name="Test order complete")).to_be_visible(timeout=15000)
    receipt_text = page.locator(".site-service-checkout-receipt").inner_text()
    receipt_match = re.search(r"Accepted test receipt\s+(\S+)\s+· posting\s+(\S+)", receipt_text)
    assert receipt_match, receipt_text
    receipt_id, posting_id = receipt_match.groups()

    with page.expect_download() as download_info:
        page.get_by_role("link", name="Download your order result").click()
    download = download_info.value
    delivery_path = result_path + ".md"
    download.save_as(delivery_path)
    with open(delivery_path, "rb") as delivery_file:
        delivery = delivery_file.read()
    assert f"# Test result for order {order_id} / job {job_id}".encode() in delivery
    assert b"Deterministic test fixture from the prepared public sample; this is not custom research." in delivery
    assert b"Public-company scan sample: Apple Inc." in delivery
    assert b"https://www.sec.gov/Archives/edgar/data/320193/000032019326000026/aapl-20260926.htm" in delivery
    delivery_sha256 = hashlib.sha256(delivery).hexdigest()
    delivery_response = next((item for item in responses if item["method"] == "GET" and item["url"].endswith(f"/service-orders/{order_id}/result")), None)
    assert delivery_response and delivery_response["status"] == 200, delivery_response
    assert delivery_response["sha256"] == delivery_sha256, delivery_response

    dimensions = page.evaluate("({width: innerWidth, document: document.documentElement.scrollWidth})")
    assert dimensions["width"] == 390 and dimensions["document"] <= dimensions["width"], dimensions
    relevant = [item for item in responses if "/api/v1/service-orders" in item["url"]]
    assert any(item["method"] == "GET" and item["url"].endswith("/offer?offer_id=company-market-scan&offer_version=1") and item["status"] == 200 for item in relevant), relevant
    assert any(item["method"] == "POST" and item["url"].endswith("/service-orders") and item["status"] == 200 for item in relevant), relevant
    assert any(item["url"].endswith(f"/service-orders/{order_id}/verify-capture") and item["status"] == 200 for item in relevant), relevant
    assert any(item["url"].endswith(f"/service-orders/{order_id}/reconcile") and item["status"] == 200 for item in relevant), relevant
    page.evaluate("window.scrollTo(0, 0)")
    page.wait_for_timeout(100)
    page.screenshot(path=screenshot_path, full_page=True)
    with open(result_path, "w", encoding="utf-8") as result_file:
        json.dump({
            "viewport": dimensions, "offer_id": "company-market-scan", "offer_version": 1,
            "amount_minor": 4900, "currency": "USD", "door": "browser",
            "order_id": order_id, "job_id": job_id, "ledger_receipt_id": receipt_id,
            "ledger_posting_id": posting_id, "requests": relevant,
            "delivered_result": {"format": "markdown", "fixture": True, "path": delivery_path, "sha256": delivery_sha256, "sha256_header": delivery_response["sha256"], "http_status": delivery_response["status"], "order_id": order_id, "job_id": job_id},
        }, result_file)
    browser.close()
