import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders } from "./test-utils";
import { CoveragePage } from "./pages/CoveragePage";
import { installPageAPIState, preparePageStory } from "./page-stories";
import { setLocale } from "./i18n";
import coverageStory from "./pages/CoveragePage.story.json";

afterEach(() => {
  cleanup();
  document.getElementById("rcl-story-result")?.remove();
  window.history.replaceState({}, "", "/");
});

describe("page story API state", () => {
  it("supplies declared responses and prevents undeclared mutations", async () => {
    const original = vi.fn().mockResolvedValue(new Response("live"));
    const host = { fetch: original as typeof fetch };
    const restore = installPageAPIState(
      [{ path: "/api/coverage", method: "POST", status: 503, body: { code: "unavailable" } }],
      host,
    );
    expect((await host.fetch("/api/coverage", { method: "POST" })).status).toBe(503);
    expect((await host.fetch("/api/publish", { method: "POST" })).status).toBe(503);
    expect(original).not.toHaveBeenCalled();
    await host.fetch("/health");
    expect(original).toHaveBeenCalledOnce();
    restore();
    expect(host.fetch).toBe(original);
  });

  it("runs the CoveragePage contract through the shared evaluator", async () => {
    await setLocale("en");
    window.history.replaceState(
      {},
      "",
      "/coverage?__rcl_page_story=CoveragePage&__rcl_story=service-unavailable",
    );
    const run = preparePageStory(new URLSearchParams(window.location.search));
    expect(run).toBeDefined();
    const { container } = renderWithProviders(<CoveragePage />, { route: "/coverage" });
    if (!run) throw new Error("CoveragePage story was not discovered");
    try {
      const expected = coverageStory.stories[0]?.expect[0]?.value;
      if (!expected) throw new Error("Coverage story must declare visible failure text");
      await screen.findByText(expected);
      await run.run(container);
    } finally {
      run.dispose();
    }
    const evidence = JSON.parse(document.getElementById("rcl-story-result")?.textContent ?? "null");
    expect(evidence).toMatchObject({ passed: true, failures: [] });
    expect(container).toHaveAttribute("data-rcl-story-status", "passed");
  });

  it("rejects a story requested on the wrong route before mounting", () => {
    expect(() =>
      preparePageStory(
        new URLSearchParams("__rcl_page_story=CoveragePage&__rcl_story=service-unavailable"),
      ),
    ).toThrow("requires route /coverage");
  });
});
