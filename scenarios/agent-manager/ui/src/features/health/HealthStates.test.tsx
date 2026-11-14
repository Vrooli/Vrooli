import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { HealthAuditDrawer } from "./components/HealthAuditDrawer.js";
import { HealthPage } from "./HealthPage.js";
import { renderWithProviders } from "../../test-utils/index.js";

afterEach(() => vi.unstubAllGlobals());

const reply = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

/** Answers each health endpoint from a table; anything unlisted is an empty document. */
function stubHealth(routes: Record<string, unknown | (() => Response)>) {
  vi.stubGlobal("fetch", vi.fn(async (input: string | URL | Request) => {
    const url = String(input);
    for (const [path, answer] of Object.entries(routes)) {
      if (url.includes(path)) return typeof answer === "function" ? (answer as () => Response)() : reply(answer);
    }
    return reply({});
  }));
}

describe("HealthPage states", () => {
  it("says so when model health cannot be read, and when the runner list is empty", async () => {
    stubHealth({ "/health/models": () => reply({ message: "health store unavailable" }, 503), "/health/runners": { runners: [] } });
    renderWithProviders(createElement(HealthPage));
    expect(await screen.findByText(/Failed to load model health/)).toBeTruthy();
    await userEvent.click(screen.getByTestId("health-tab-runners"));
    expect(await screen.findByTestId("runner-health-empty")).toBeTruthy();
  });

  it("says so when runner health cannot be read, and links to Investigations", async () => {
    stubHealth({ "/health/models": { models: [] }, "/health/runners": () => reply({ message: "runner store unavailable" }, 503) });
    renderWithProviders(createElement(HealthPage));
    await userEvent.click(screen.getByTestId("health-tab-runners"));
    expect(await screen.findByText(/Failed to load runner health/)).toBeTruthy();
    expect(screen.getByRole("link", { name: /Investigations/ }).getAttribute("href")).toBe("/investigations");
  });

  it("opens a runner's audit history from the Runners tab and closes it again", async () => {
    stubHealth({
      "/health/models": { models: [] },
      "/health/runners": { runners: [{ runner: "claude-code", status: "ok", last_checked: "2026-07-29T00:00:00Z" }] },
      "/health/audit": { rows: [{ id: 3, status: "ok", timestamp: "2026-07-29T00:00:00Z", runnerType: "claude-code", triggeredBy: "scheduler" }] },
    });
    renderWithProviders(createElement(HealthPage));
    await userEvent.click(screen.getByTestId("health-tab-runners"));
    await userEvent.click(await screen.findByRole("button", { name: "Show audit history for runner Claude Code" }));
    expect(await screen.findByTestId("audit-row-3")).toBeTruthy();
    await userEvent.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByTestId("audit-row-3")).toBeNull());
  });

  it("shows empty tables when the API answers without rows", async () => {
    stubHealth({ "/health/models": {}, "/health/runners": {} });
    renderWithProviders(createElement(HealthPage));
    expect(await screen.findByTestId("model-health-empty")).toBeTruthy();
    await userEvent.click(screen.getByTestId("health-tab-runners"));
    expect(await screen.findByTestId("runner-health-empty")).toBeTruthy();
  });

  it("shows a loading note while the first read is in flight", async () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));
    renderWithProviders(createElement(HealthPage));
    expect((await screen.findAllByText("Loading…")).length).toBeGreaterThan(0);
  });
});

describe("HealthAuditDrawer", () => {
  const open = (filters: Parameters<typeof HealthAuditDrawer>[0]["filters"], isOpen = true) =>
    renderWithProviders(createElement(HealthAuditDrawer, { open: isOpen, filters, onOpenChange: vi.fn() }));

  it("titles a runner audit by runner and lists rows with reason and message in plain text", async () => {
    stubHealth({ "/health/audit": { rows: [
      { id: 1, status: "failed", timestamp: "2026-07-29T00:00:00Z", runnerType: "codex", modelId: "gpt-5", triggeredBy: "scheduler", reason: "quota", message: "limit reached" },
      { id: 2, status: "ok", timestamp: "2026-07-28T00:00:00Z", runnerType: "codex", modelId: "", triggeredBy: "operator" },
    ] } });
    open({ scope: "runner", runner: "codex" });
    expect(await screen.findByText("Audit history — runner codex")).toBeTruthy();
    expect(await screen.findByTestId("audit-row-1")).toBeTruthy();
    expect(screen.getByText("quota")).toBeTruthy();
    expect(screen.getByText("limit reached")).toBeTruthy();
    expect(screen.getByTestId("audit-row-2").textContent).toContain("triggered by operator");
    expect(screen.getByTestId("audit-row-2").textContent).not.toContain("Reason:");
  });

  it("titles a model audit with 'all' for an unset runner or model, and says when there are no rows", async () => {
    stubHealth({ "/health/audit": { rows: [] } });
    open({ scope: "model" });
    expect(await screen.findByText("Audit history — all / all")).toBeTruthy();
    expect(await screen.findByTestId("audit-empty")).toBeTruthy();
  });

  it("reports a failed read", async () => {
    stubHealth({ "/health/audit": () => reply({ message: "audit unavailable" }, 503) });
    open({ scope: "model", runner: "codex", model: "gpt-5" });
    expect((await screen.findByTestId("audit-error")).textContent).toContain("audit unavailable");
  });

  it("renders nothing while closed", async () => {
    stubHealth({});
    open(null, false);
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("uses a neutral title when no filters are chosen yet", async () => {
    stubHealth({});
    open(null);
    expect(await screen.findByText("Audit history")).toBeTruthy();
  });
});
