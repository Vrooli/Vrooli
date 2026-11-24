import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CapabilityActions } from "./CapabilityActions";
import { applyCapability, previewCapability, verifyCapability, type CapabilityStatus } from "../../api/capabilities";

vi.mock("../../api/capabilities", () => ({
  applyCapability: vi.fn(),
  previewCapability: vi.fn(),
  verifyCapability: vi.fn(),
}));

const status: CapabilityStatus = {
  descriptor: {
    version: "operator-capability/v1",
    id: "target-secret",
    owner: "fixture.owner",
    title: "Target-scoped secret",
    inputs: [
      { id: "destination", kind: "path", label: "Destination", required: true },
      { id: "token", kind: "secret", label: "Access token", required: true },
    ],
    policy: { requires_confirmation: false, idempotent: true, retryable: true },
    evidence: { secret_free: true },
  },
  state: "needs_operator_input",
  missing_inputs: ["token"],
  updated_at: "now",
};

describe("CapabilityActions target changes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.history.replaceState({}, "", "/");
  });

  it("binds first-touch to the adoption query while executing locally and clears the secret after apply", async () => {
    window.history.replaceState({}, "", "/setup/credentials?target=local&adoption_id=adopted-vps%3Afixture-1");
    const firstTouch: CapabilityStatus = {
      ...status,
      descriptor: {
        ...status.descriptor,
        id: "compute-manager-adoption-first-touch",
        title: "Connect an adopted VPS",
        inputs: [
          { id: "instance_id", kind: "path", label: "Adoption ID", required: true },
          { id: "idempotency_key", kind: "path", label: "Request identity", required: false },
          { id: "ssh_password", kind: "secret", label: "SSH password", required: true },
        ],
        policy: { requires_confirmation: true, idempotent: true, retryable: true },
        lifecycle: { preview: true, apply: true, verify: true, revoke: false, recover: false },
      },
      missing_inputs: ["ssh_password"],
    };
    vi.mocked(previewCapability).mockResolvedValue({
      capability_id: firstTouch.descriptor.id,
      plan_id: "first-touch:adopted-vps:fixture-1",
      state: "ready_to_preview",
      mutations: [{ id: "bridge-first-touch", summary: "Forward one request-scoped credential", reversible: false }],
    });
    vi.mocked(applyCapability).mockResolvedValue({ capability_id: firstTouch.descriptor.id, state: "retryable_failure", outcome: "bridge_enrolling", retryable: true });
    vi.mocked(verifyCapability).mockResolvedValue({ capability_id: firstTouch.descriptor.id, outcome: "bridge_evidence_pending", retryable: true, evidence: [] });
    render(<CapabilityActions target="local" statuses={[firstTouch]} onRefresh={vi.fn()} />);

    expect(screen.queryByLabelText("Adoption ID")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Request identity")).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("SSH password"), { target: { value: "one-time-password" } });
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    await screen.findByTestId("capability-preview-compute-manager-adoption-first-touch");
    expect(previewCapability).toHaveBeenCalledWith(expect.objectContaining({
      capability_id: firstTouch.descriptor.id,
      inputs: expect.objectContaining({ instance_id: "adopted-vps:fixture-1", ssh_password: "one-time-password" }),
      idempotency_key: expect.stringMatching(/^[0-9a-f]{48}$/),
    }), "local");
    fireEvent.click(screen.getByTestId("capability-confirm-compute-manager-adoption-first-touch"));
    fireEvent.click(await screen.findByRole("button", { name: "Apply reviewed capability" }));
    await waitFor(() => expect(applyCapability).toHaveBeenCalledWith(expect.objectContaining({
      capability_id: firstTouch.descriptor.id,
      inputs: expect.objectContaining({ instance_id: "adopted-vps:fixture-1", ssh_password: "one-time-password" }),
      idempotency_key: expect.stringMatching(/^[0-9a-f]{48}$/),
    }), "local"));
    await waitFor(() => expect(screen.getByLabelText("SSH password")).toHaveValue(""));
    expect(screen.getByTestId("capability-result-compute-manager-adoption-first-touch")).not.toHaveTextContent("one-time-password");
    fireEvent.click(screen.getByRole("button", { name: "Verify now" }));
    await waitFor(() => expect(verifyCapability).toHaveBeenCalledWith(firstTouch.descriptor.id, "local", { instance_id: "adopted-vps:fixture-1" }));
  });

  it("shows and submits the generated review fingerprint before an operator confirms", async () => {
    vi.mocked(previewCapability).mockResolvedValue({
      capability_id: "principal-grant",
      plan_id: "review-plan",
      state: "ready_to_preview",
      mutations: [{ id: "issue", summary: "Issue the two listed service principals", reversible: false }],
    });
    const fingerprint = "a".repeat(64);
    const reviewed: CapabilityStatus = {
      ...status,
      descriptor: {
        ...status.descriptor,
        id: "principal-grant",
        title: "Authorize Compute Manager phone decisions",
        inputs: [{ id: "plan_digest", kind: "path", label: "Reviewed plan fingerprint", required: true, default: fingerprint }],
        policy: { requires_confirmation: true, idempotent: true, retryable: true },
      },
      missing_inputs: [],
    };
    render(<CapabilityActions target="local" statuses={[reviewed]} onRefresh={vi.fn()} />);

    expect(screen.getByLabelText("Reviewed plan fingerprint")).toHaveValue(fingerprint);
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    expect(await screen.findByTestId("capability-preview-principal-grant")).toHaveTextContent("Issue the two listed service principals");
    expect(previewCapability).toHaveBeenCalledWith({ capability_id: "principal-grant", confirm: false, inputs: { plan_digest: fingerprint } }, "local");
    expect(screen.getByTestId("capability-confirm-principal-grant")).toBeInTheDocument();
  });

  it("clears draft values and reviewed mutations when the selected target changes", async () => {
    vi.mocked(previewCapability).mockResolvedValue({
      capability_id: "target-secret",
      plan_id: "plan-local",
      state: "ready_to_preview",
      mutations: [{ id: "write", summary: "Write target config", reversible: true }],
    });
    const view = render(<CapabilityActions target="local" statuses={[status]} onRefresh={vi.fn()} />);
    fireEvent.change(screen.getByLabelText("Destination"), { target: { value: "/local/config" } });
    const secret = screen.getByLabelText("Access token");
    fireEvent.change(secret, { target: { value: "local-only-secret" } });
    expect(secret).toHaveValue("local-only-secret");
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    expect(await screen.findByTestId("capability-preview-target-secret")).toHaveTextContent("Write target config");

    view.rerender(<CapabilityActions target="remote-2" statuses={[status]} onRefresh={vi.fn()} />);

    expect(screen.getByLabelText("Access token")).toHaveValue("");
    expect(screen.getByLabelText("Destination")).toHaveValue("");
    await waitFor(() => expect(screen.queryByTestId("capability-preview-target-secret")).not.toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Preview" })).toBeDisabled();
  });
});
