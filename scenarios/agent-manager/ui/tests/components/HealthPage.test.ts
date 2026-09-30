import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { beforeEach, describe, it, expect, vi } from "vitest";
import { HealthPage } from "../../src/features/health/HealthPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  globalThis.fetch = fetchMock as unknown as typeof globalThis.fetch;
});

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  }) as Response;
}

function routeResponses(byPath: Record<string, unknown>) {
  fetchMock.mockImplementation(async (input: string | URL | Request) => {
    const url = typeof input === "string" ? input : input.toString();
    for (const [pathPrefix, body] of Object.entries(byPath)) {
      if (url.startsWith(pathPrefix)) {
        return jsonResponse(body);
      }
    }
    return jsonResponse({ error: `unmocked URL: ${url}` }, 404);
  });
}

describe("HealthPage", () => {
  it("renders the model snapshot with status badges and surfaces failed rows first", async () => {
    routeResponses({
      "/api/v1/health/models": {
        models: [
          {
            runner: "codex",
            model: "gpt-5.2-codex",
            status: "ok",
            last_checked: new Date().toISOString(),
          },
          {
            runner: "claude-code",
            model: "claude-opus-4-7",
            status: "failed",
            last_checked: new Date().toISOString(),
            reason: "rate_limit",
            message: "anthropic 429",
          },
        ],
      },
      "/api/v1/health/runners": { runners: [] },
	  "/api/v1/health/model-policy-drift": { status: "warning", measured: 4, total: 4, interval_hours: 168, findings: [{ runner: "codex", type: "catalog_stale", severity: "warning", message: "stale", fingerprint: "f" }] },
    });

    renderWithProviders(createElement(HealthPage));

    await waitFor(() => {
      expect(screen.getByText("gpt-5.2-codex")).toBeTruthy();
    });
    expect(screen.getByText("claude-opus-4-7")).toBeTruthy();
    expect(screen.getByText("Failed")).toBeTruthy();
    expect(screen.getByText("OK")).toBeTruthy();
	  expect(screen.getByTestId("model-policy-drift-status").textContent).toContain("warning");

    const rows = screen.getAllByTestId(/^model-health-row-/);
    // Failed row sorts to the top.
    expect(rows[0].getAttribute("data-testid")).toBe("model-health-row-claude-code-claude-opus-4-7");
  });

  it("opens the audit drawer when the audit button is clicked", async () => {
    routeResponses({
      "/api/v1/health/models": {
        models: [
          {
            runner: "codex",
            model: "gpt-5.2-codex",
            status: "failed",
            last_checked: new Date().toISOString(),
            reason: "rate_limit",
          },
        ],
      },
      "/api/v1/health/runners": { runners: [] },
	  "/api/v1/health/model-policy-drift": { status: "healthy", measured: 4, total: 4, interval_hours: 168 },
      "/api/v1/health/audit": {
        rows: [
          {
            id: 7,
            timestamp: new Date().toISOString(),
            runnerType: "codex",
            modelId: "gpt-5.2-codex",
            status: "failed",
            reason: "rate_limit",
            triggeredBy: "run-abc",
          },
        ],
        limit: 100,
        scope: "model",
      },
    });

    renderWithProviders(createElement(HealthPage));

    const auditButton = await screen.findByLabelText("Show audit history for codex / gpt-5.2-codex");
    await userEvent.click(auditButton);

    await waitFor(() => {
      expect(screen.getByTestId("audit-row-7")).toBeTruthy();
    });
  });
});
