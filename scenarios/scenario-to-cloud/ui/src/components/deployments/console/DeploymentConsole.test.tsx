import "@testing-library/jest-dom";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/renderWithProviders";
import { FIXTURE_DEPLOYMENT_ID, makeMatrix, makeObservation, makePlan, makeRecoveryPoint, makeStanding } from "../../../test-utils/consoleFixtures";
import { operationPointerKey, saveOperationPointer } from "../../../lib/consoleApi";
import { DeploymentConsole } from "./DeploymentConsole";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

type Router = Record<string, (init?: RequestInit, url?: string) => Response | Promise<Response>>;

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  url: string;
  onopen: null | (() => void) = null;
  onerror: null | (() => void) = null;
  listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
  constructor(url: string) {
    this.url = url;
    FakeEventSource.instances.push(this);
  }
  addEventListener(type: string, cb: (e: MessageEvent) => void) {
    (this.listeners[type] ??= []).push(cb);
  }
  close() {}
}

const deployment = {
  id: FIXTURE_DEPLOYMENT_ID,
  name: "Fixture",
  scenario_id: "fixture-scenario",
  status: "deployed",
  environment: "production",
  target: { machine_id: "fixture-host-203.0.113.10", node_id: "node-1", enrollment_generation: 3, transport: "bridge" },
  fence: 3,
};

describe("DeploymentConsole", () => {
  const fetchMock = vi.fn();
  let router: Router;

  function route(method: string, path: string, handler: Router[string]) {
    router[`${method} ${path}`] = handler;
  }

  beforeEach(() => {
    sessionStorage.clear();
    FakeEventSource.instances = [];
    vi.stubGlobal("EventSource", FakeEventSource);
    router = {};
    fetchMock.mockReset();
    fetchMock.mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const path = url.replace(/^.*\/api\/v1/, "").split("?")[0] ?? "";
      const key = `${init?.method ?? "GET"} ${path}`;
      const handler = router[key];
      if (!handler) return jsonResponse({ error: { code: "deployment_not_found", message: `unrouted ${key}` } }, 404);
      return handler(init, url);
    });
    vi.stubGlobal("fetch", fetchMock);
    route("GET", "/authz/matrix", () => jsonResponse(makeMatrix()));
    route("POST", `/deployments/${FIXTURE_DEPLOYMENT_ID}/plan`, () => jsonResponse(makePlan()));
    route("GET", `/deployments/${FIXTURE_DEPLOYMENT_ID}/health/observation`, () => jsonResponse({ schema_version: "1", observation: makeObservation() }));
    route("GET", `/deployments/${FIXTURE_DEPLOYMENT_ID}/recovery-points`, () => jsonResponse({ schema_version: "1", recovery_points: [makeRecoveryPoint()] }));
    route("GET", `/deployments/${FIXTURE_DEPLOYMENT_ID}/operations`, () => jsonResponse({ schema_version: "1", deployment_id: FIXTURE_DEPLOYMENT_ID, operations: [], timestamp: "t" }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // [REQ:STC-P0-038] [REQ:STC-P0-039] Review → apply (digest + uuid request key) → durable operation,
  // with the pointer persisted before the operation view renders.
  it("reviews, applies with the reviewed digest and follows the durable operation to success", async () => {
    const user = userEvent.setup();
    let applyBody: Record<string, unknown> | null = null;
    let waits = 0;
    route("POST", `/deployments/${FIXTURE_DEPLOYMENT_ID}/plan/apply`, (init) => {
      applyBody = JSON.parse(init?.body as string);
      return jsonResponse({ schema_version: "1", operation_id: "op-new", plan_digest: "sha256:plan-digest-fixture", state: "admitted" }, 202);
    });
    route("GET", "/operations/op-new", () => jsonResponse(makeStanding({ operation_id: "op-new", state: "admitted", completed_steps: [], active_step: "host.prepare", step_receipts: [] })));
    route("GET", "/operations/op-new/wait", () => {
      waits += 1;
      return jsonResponse(makeStanding({ operation_id: "op-new", state: "succeeded", terminal: true, active_step: undefined, completed_steps: ["host.prepare", "release.stage"], next_action: undefined, result: { outcome: "succeeded", completed_steps: 2, message: "Release activated" } }));
    });

    renderWithProviders(<DeploymentConsole deployment={deployment} />);
    await waitFor(() => expect(screen.getByTestId("console-release")).toHaveAttribute("data-state", "ready"));
    expect(screen.getByTestId("console-release-match")).toBeInTheDocument();
    expect(screen.getByTestId("console-health-status")).toHaveTextContent("Healthy");
    await waitFor(() => expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "empty"));
    await expectNoA11yViolations(document.body);

    await user.click(screen.getByTestId("console-review-open"));
    await waitFor(() => expect(screen.getByTestId("console-review")).toHaveAttribute("data-state", "ready"));
    expect(screen.getByRole("heading", { name: "Deploy release aaaaaaaaaaaa" })).toHaveFocus();
    await expectNoA11yViolations(document.body);

    await user.click(screen.getByTestId("console-review-apply"));
    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByTestId("console-destructive-data")).toHaveTextContent("postgres:main");
    await user.type(screen.getByTestId("console-destructive-confirm-input"), FIXTURE_DEPLOYMENT_ID.slice(0, 8));
    await user.click(screen.getByTestId("console-destructive-confirm"));

    await waitFor(() => expect(applyBody).not.toBeNull());
    expect(applyBody).toMatchObject({ plan_digest: "sha256:plan-digest-fixture" });
    expect((applyBody as unknown as Record<string, string>).request_key).toMatch(/^[0-9a-f-]{36}$/);
    await waitFor(() => expect(JSON.parse(sessionStorage.getItem(operationPointerKey(FIXTURE_DEPLOYMENT_ID)) ?? "{}")).toMatchObject({ operation_id: "op-new", plan_digest: "sha256:plan-digest-fixture" }));
    await waitFor(() => expect(screen.getByTestId("console-operation-state")).toHaveTextContent("succeeded"));
    expect(waits).toBeGreaterThanOrEqual(1);
    expect(screen.getByTestId("console-operation-result")).toHaveTextContent("Release activated");
    expect(screen.getByTestId("console-review-open")).toHaveTextContent("Review a new plan");
    expect(screen.queryByTestId("console-review")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("console-operation-detach"));
    await waitFor(() => expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "empty"));
    expect(sessionStorage.getItem(operationPointerKey(FIXTURE_DEPLOYMENT_ID))).toBeNull();
  });

  // [REQ:STC-P0-039] An interrupted operation resumes from durable state on reload.
  it("resumes an interrupted operation from the stored pointer, streams step messages and preserves local dialog state across refreshes", async () => {
    const user = userEvent.setup();
    saveOperationPointer({ deployment_id: FIXTURE_DEPLOYMENT_ID, operation_id: "op-1234567890", plan_digest: "sha256:plan-digest-fixture" });
    let round = 0;
    route("GET", "/operations/op-1234567890", () => jsonResponse(makeStanding()));
    route("GET", "/operations/op-1234567890/wait", async () => {
      round += 1;
      await new Promise((resolve) => setTimeout(resolve, 30));
      return jsonResponse(makeStanding({ active_step: `step-${round}`, completed_steps: ["host.prepare"], updated_at: `2026-09-09T12:00:${String(20 + round).padStart(2, "0")}Z` }));
    });

    renderWithProviders(<DeploymentConsole deployment={deployment} />);
    await waitFor(() => expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "interrupted"));
    expect(screen.getByTestId("console-operation-resumed")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Operation" })).toHaveFocus();
    expect(screen.getByTestId("console-operation-reattach")).toHaveTextContent("scenario-to-cloud operation wait op-1234567890");
    await waitFor(() => expect(FakeEventSource.instances.length).toBeGreaterThan(0));
    expect(FakeEventSource.instances[0]?.url).toContain("operation_id=op-1234567890");

    await user.click(screen.getByTestId("console-operation-cancel"));
    await user.type(screen.getByTestId("console-destructive-confirm-input"), "op-1");
    const before = round;
    await waitFor(() => expect(round).toBeGreaterThan(before + 1), { timeout: 3000 });
    await waitFor(() => expect(screen.getByTestId("console-operation-steps")).toHaveTextContent(`step-${round}`));
    expect(screen.getByTestId("console-destructive-confirm-input")).toHaveValue("op-1");
    await user.keyboard("{Escape}");
    expect(screen.getByTestId("console-operation-cancel")).toHaveFocus();
  });

  // [REQ:STC-P0-039] Unavailable authority names the specific missing permission.
  it("explains a denied plan with the missing scope and keeps the rest of the page readable", async () => {
    route("POST", `/deployments/${FIXTURE_DEPLOYMENT_ID}/plan`, () => jsonResponse({ error: { code: "forbidden_scope", message: "scope missing", retryable: false, details: { required_scope: "scenario-to-cloud:read" } } }, 403));
    route("GET", `/deployments/${FIXTURE_DEPLOYMENT_ID}/recovery-points`, () => jsonResponse({ error: { code: "forbidden_target", message: "target not granted", retryable: false } }, 403));
    renderWithProviders(<DeploymentConsole deployment={deployment} />);
    await waitFor(() => expect(screen.getByTestId("console-release")).toHaveAttribute("data-state", "denied"));
    expect(screen.getByTestId("console-release-denied-scope")).toHaveTextContent("scenario-to-cloud:read");
    await waitFor(() => expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "denied"));
    expect(screen.getByTestId("console-recovery-denied-target")).toHaveTextContent("machine:fixture-host-203.0.113.10");
    expect(screen.getByTestId("console-recovery-denied-scope")).toHaveTextContent("scenario-to-cloud:read");
    expect(screen.getByTestId("console-health-status")).toHaveTextContent("Healthy");
    await expectNoA11yViolations(document.body);
  });

  it("renders the onboarding handoff for a needs_input plan in review and recovery", async () => {
    const user = userEvent.setup();
    route("POST", `/deployments/${FIXTURE_DEPLOYMENT_ID}/plan`, () =>
      jsonResponse(makePlan({ outcome: "needs_input", handoff: { owner: "vrooli-onboarding", kind: "resume_handoff", reference: "vrooli-onboarding://deployments/x/resume/1", missing: ["credential:stripe"] } })),
    );
    renderWithProviders(<DeploymentConsole deployment={deployment} />);
    await waitFor(() => expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "needs-input"));
    expect(screen.getByTestId("console-recovery-handoff-link")).toHaveAttribute("href", "vrooli-onboarding://deployments/x/resume/1");
    await user.click(screen.getByTestId("console-review-open"));
    await waitFor(() => expect(screen.getByTestId("console-review")).toHaveAttribute("data-state", "needs-input"));
    expect(screen.getByTestId("console-review-apply")).toBeDisabled();
    expect(screen.getByTestId("console-review-handoff-link")).toHaveFocus();
    await expectNoA11yViolations(document.body);
    await user.click(screen.getByTestId("console-review-close"));
    expect(screen.queryByTestId("console-review")).not.toBeInTheDocument();
  });

  it("attaches an externally admitted operation and routes recovery next actions to the recovery heading", async () => {
    const user = userEvent.setup();
    route("GET", "/operations/op-ext", () => jsonResponse(makeStanding({ operation_id: "op-ext", state: "failed", terminal: true, error: { code: "release_verification_failed", message: "digest mismatch" }, next_action: { owner: "scenario-to-cloud", kind: "recovery", reference: "/api/v1/deployments/x/recovery" } })));
    const { rerender } = renderWithProviders(<DeploymentConsole deployment={deployment} />);
    await waitFor(() => expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "empty"));
    rerender(<DeploymentConsole deployment={deployment} attachRequest={{ deployment_id: FIXTURE_DEPLOYMENT_ID, operation_id: "op-ext", plan_digest: "sha256:p" }} />);
    await waitFor(() => expect(screen.getByTestId("console-operation-state")).toHaveTextContent("failed"));
    expect(screen.getByTestId("console-operation-refusal")).toHaveTextContent("release_verification_failed");
    await user.click(screen.getByTestId("console-operation-next-action"));
    expect(screen.getByRole("heading", { name: "Recovery" })).toHaveFocus();
    await waitFor(() => expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "ready"));
    await expectNoA11yViolations(document.body);
  });

  // [REQ:STC-P0-039] Keyboard-only order follows reading order: release, health, operation, recovery.
  it("keeps a logical keyboard order across panels", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DeploymentConsole deployment={deployment} />);
    await waitFor(() => expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "empty"));
    await waitFor(() => expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "ready"));
    const order: string[] = [];
    for (let i = 0; i < 12; i += 1) {
      await user.tab();
      const active = document.activeElement as HTMLElement | null;
      if (!active || active === document.body) break;
      order.push(active.getAttribute("data-testid") ?? active.getAttribute("aria-label") ?? active.textContent ?? "");
    }
    const first = order.findIndex((label) => label.includes("desired release digest"));
    const review = order.indexOf("console-review-open");
    const recovery = order.findIndex((label) => label === "rp-0001");
    expect(first).toBeGreaterThanOrEqual(0);
    expect(review).toBeGreaterThan(first);
    expect(recovery).toBeGreaterThan(review);
  });
});
