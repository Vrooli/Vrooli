import "@testing-library/jest-dom";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";

import App from "./App";
import { renderWithProviders } from "./test-utils/renderWithProviders";
import { FIXTURE_DEPLOYMENT_ID, makeMatrix, makeObservation, makePlan, makeRecoveryPoint, makeStanding } from "./test-utils/consoleFixtures";
import { saveOperationPointer } from "./lib/consoleApi";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

const deploymentRecord = {
  id: FIXTURE_DEPLOYMENT_ID,
  name: "Fixture deployment",
  scenario_id: "fixture-scenario",
  status: "deployed",
  environment: "production",
  target: { machine_id: "fixture-host-203.0.113.10", node_id: "node-1", enrollment_generation: 3, transport: "bridge", locator: { host: "203.0.113.10" } },
  fence: 3,
  manifest: { scenario: { id: "fixture-scenario" }, edge: { domain: "fixture.example" } },
  created_at: "2026-09-09T10:00:00Z",
  updated_at: "2026-09-09T12:00:00Z",
  last_deployed_at: "2026-09-09T11:00:00Z",
};

class FakeEventSource {
  onopen: null | (() => void) = null;
  onerror: null | (() => void) = null;
  addEventListener() {}
  close() {}
}

function stubApi(overrides: Record<string, () => Response> = {}) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const path = url.replace(/^.*\/api\/v1/, "").split("?")[0] ?? "";
      const key = `${init?.method ?? "GET"} ${path}`;
      const override = overrides[key];
      if (override) return override();
      if (path === "/health") return jsonResponse({ status: "healthy", service: "Scenario To Cloud API" });
      if (path === "/authz/matrix") return jsonResponse(makeMatrix());
      if (path === `/deployments/${FIXTURE_DEPLOYMENT_ID}`) return jsonResponse({ deployment: deploymentRecord, timestamp: "t" });
      if (path === `/deployments/${FIXTURE_DEPLOYMENT_ID}/plan`) return jsonResponse(makePlan());
      if (path === `/deployments/${FIXTURE_DEPLOYMENT_ID}/health/observation`) return jsonResponse({ schema_version: "1", observation: makeObservation() });
      if (path === `/deployments/${FIXTURE_DEPLOYMENT_ID}/recovery-points`) return jsonResponse({ schema_version: "1", recovery_points: [makeRecoveryPoint()] });
      if (path === `/deployments/${FIXTURE_DEPLOYMENT_ID}/operations`) return jsonResponse({ schema_version: "1", deployment_id: FIXTURE_DEPLOYMENT_ID, operations: [], timestamp: "t" });
      if (path === `/deployments/${FIXTURE_DEPLOYMENT_ID}/live-state`) return jsonResponse({ error: { code: "reach_unavailable", message: "no reach" } }, 503);
      if (path.startsWith(`/deployments/${FIXTURE_DEPLOYMENT_ID}/investigations`)) return jsonResponse({ investigations: [], timestamp: "t" });
      if (path === "/operations/op-1234567890") return jsonResponse(makeStanding());
      if (path === "/operations/op-1234567890/wait") return jsonResponse(makeStanding({ state: "succeeded", terminal: true, next_action: undefined }));
      return jsonResponse({ error: { code: "deployment_not_found", message: `not found ${key}` } }, 404);
    }),
  );
}

describe("application accessibility", () => {
  beforeEach(() => {
    window.history.replaceState(null, "", "#dashboard");
    localStorage.clear();
    sessionStorage.clear();
    vi.stubGlobal("EventSource", FakeEventSource);
    stubApi();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders the dashboard without axe violations", async () => {
    renderWithProviders(<App />);

    expect(await screen.findByText("Deploy Scenarios to the Cloud")).toBeInTheDocument();
    await expectNoA11yViolations(document.body);
  });

  // [REQ:STC-P0-038] [REQ:STC-P0-039] The deployment console, its review and
  // its recovery surfaces are axe-clean and reachable by keyboard from the page.
  it("renders the deployment console, the plan review and the recovery preview without axe violations", async () => {
    const user = userEvent.setup();
    window.history.replaceState(null, "", `#deployments/${FIXTURE_DEPLOYMENT_ID}`);
    renderWithProviders(<App />);

    expect(await screen.findByRole("heading", { name: "Fixture deployment" })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("console-release")).toHaveAttribute("data-state", "ready"));
    await waitFor(() => expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "empty"));
    await waitFor(() => expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "ready"));
    expect(screen.getByTestId("console-identity-target")).toHaveTextContent("machine:fixture-host-203.0.113.10");
    await expectNoA11yViolations(document.body);

    await user.click(screen.getByTestId("console-review-open"));
    await waitFor(() => expect(screen.getByTestId("console-review")).toHaveAttribute("data-state", "ready"));
    await expectNoA11yViolations(document.body);

    await user.click(screen.getByTestId("console-review-apply"));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    await expectNoA11yViolations(document.body);
    await user.keyboard("{Escape}");
    expect(screen.getByTestId("console-review-apply")).toHaveFocus();

    await user.click(screen.getByRole("button", { name: "rp-0001" }));
    await user.click(screen.getByTestId("console-recovery-restore"));
    expect(screen.getByTestId("console-destructive-data")).toHaveTextContent("postgres:main");
    await expectNoA11yViolations(document.body);
    await user.keyboard("{Escape}");

    await user.click(screen.getByTestId("console-advanced-toggle"));
    expect(screen.getByRole("tab", { name: "Terminal" })).toBeInTheDocument();
    await expectNoA11yViolations(document.body);
  });

  it("renders a denied console and a resumed operation without axe violations", async () => {
    stubApi({
      [`POST /deployments/${FIXTURE_DEPLOYMENT_ID}/plan`]: () => jsonResponse({ error: { code: "forbidden_scope", message: "scope missing", retryable: false, details: { required_scope: "scenario-to-cloud:read" } } }, 403),
    });
    saveOperationPointer({ deployment_id: FIXTURE_DEPLOYMENT_ID, operation_id: "op-1234567890", plan_digest: "sha256:plan-digest-fixture" });
    window.history.replaceState(null, "", `#deployments/${FIXTURE_DEPLOYMENT_ID}`);
    renderWithProviders(<App />);

    await waitFor(() => expect(screen.getByTestId("console-release")).toHaveAttribute("data-state", "denied"));
    expect(screen.getByTestId("console-release-denied-scope")).toHaveTextContent("scenario-to-cloud:read");
    await waitFor(() => expect(screen.getByTestId("console-operation-resumed")).toBeInTheDocument());
    await expectNoA11yViolations(document.body);
  });
});
