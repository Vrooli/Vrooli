import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";

const api = vi.hoisted(() => ({
  acquireSession: vi.fn(),
  killSession: vi.fn(),
  listDevices: vi.fn(),
  listStrategies: vi.fn(),
  releaseSession: vi.fn(),
  runFlow: vi.fn(),
  validateFlow: vi.fn(),
}));

vi.mock("../api/deviceControl", () => api);

import { FlowsPage } from "./FlowsPage";

const device = {
  id: "android-phone-1",
  name: "Galaxy A03s",
  kind: "physical",
  serial: "R9TT608Q6MH",
  model: "SM_A037U",
  os_version: "Android 13",
  transport: "usb",
  strategy_id: "android-adb",
  status: "available",
  capabilities: [{ name: "screenshot", status: "available" }],
};
const session = { id: "lease-1", device_id: device.id, actor: "browser-operator", state: "held", lease_token: "token", expires_at: "later", created_at: "now" };

beforeEach(() => {
  vi.clearAllMocks();
  api.listDevices.mockResolvedValue({ devices: [device] });
  api.listStrategies.mockResolvedValue({ strategies: [{ id: "android-adb", description: "ADB", status: "ready", tiers: ["replay", "physical"], executable_step_kinds: ["observe"], capabilities: {}, promotable: true }] });
  api.acquireSession.mockResolvedValue({ session });
  api.killSession.mockResolvedValue({});
  api.releaseSession.mockResolvedValue({ session: { ...session, state: "released" } });
});

describe("FlowsPage", () => {
  it("exposes an immediate kill control while a physical flow is running", async () => {
    let resolveRun: ((value: unknown) => void) | undefined;
    api.runFlow.mockReturnValue(new Promise((resolve) => { resolveRun = resolve; }));
    const user = userEvent.setup();
    renderWithProviders(<FlowsPage />);

    const runButton = await screen.findByRole("button", { name: strings.pages.flows.acquireAndRun });
    await user.click(runButton);
    const killButton = await screen.findByRole("button", { name: strings.pages.flows.killActiveSession });
    expect(api.acquireSession).toHaveBeenCalledWith(device.id, "browser-operator");

    await user.click(killButton);
    expect(api.killSession).toHaveBeenCalledWith(session.id);
    resolveRun?.({ run_id: "run-1", disposition: "killed", chapters: [], evidence: [] });
    await waitFor(() => expect(api.releaseSession).toHaveBeenCalledWith(session.id));
  });

  it("renders retained image evidence inline with its checksum", async () => {
    api.runFlow.mockResolvedValue({
      run_id: "run-1",
      disposition: "passed",
      chapters: [{ id: "observe", disposition: "passed", message: "completed" }],
      evidence: [{ id: "evidence-1", kind: "image", checksum: "abc123", size_bytes: 128, redaction_verified: true }],
    });
    const user = userEvent.setup();
    renderWithProviders(<FlowsPage />);

    await user.click(await screen.findByRole("button", { name: strings.pages.flows.acquireAndRun }));
    const image = await screen.findByTestId(selectors.pages.flowEvidenceImage);
    expect(image).toHaveAttribute("src", expect.stringContaining("/api/v1/evidence/evidence-1"));
    expect(screen.getByTestId(selectors.pages.flowRunReview)).toHaveTextContent("abc123");
  });
});
