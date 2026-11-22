import json
import sys

from playwright.sync_api import expect, sync_playwright


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: status-pagination.browser.py <same-origin harness URL>")
    base_url = sys.argv[1]
    page_url = f"{base_url}/e2e/status-pagination.html?repoId=1"
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            executable_path="/usr/bin/google-chrome",
            headless=True,
            args=["--no-sandbox", "--disable-dev-shm-usage"],
        )
        page = browser.new_page()
        page.set_default_timeout(120_000)
        page.goto(page_url, wait_until="domcontentloaded")
        state = page.get_by_test_id("status-state")
        expect(state).to_have_attribute("data-state", "complete")
        expect(state).to_contain_text("Complete: 2049 status entries")
        result = json.loads(page.get_by_test_id("status-result").inner_text())

        expected_unstaged = [f"tracked-{index:06d}.txt" for index in range(1024)]
        expected_untracked = sorted([f"untracked-{index:06d}.txt" for index in range(1024)] + ["refresh-added-after-first-page.txt"])
        assert result["files"]["unstaged"] == expected_unstaged, "browser aggregator lost or reordered tracked paths"
        assert result["files"]["untracked"] == expected_untracked, "browser aggregator lost or reordered untracked paths"
        for category in ("staged", "unstaged", "untracked", "conflicts", "ignored"):
            paths = result["files"][category]
            assert len(paths) == len(set(paths)), f"duplicate {category} path across pages"
        assert result["counts"] == {"staged": 0, "unstaged": 1024, "untracked": 1025, "conflicts": 0, "ignored": 0}

        page.request.post(f"{base_url}/__test/always-mutate")
        page.get_by_role("button", name="Refresh status").click()
        expect(state).to_have_attribute("data-state", "error")
        expect(state).to_contain_text("repository status kept changing while loading pages")
        print(json.dumps({"complete_state": "passed", "error_state": "passed", "initial_category_entries": 2049}))
        browser.close()


if __name__ == "__main__":
    main()
