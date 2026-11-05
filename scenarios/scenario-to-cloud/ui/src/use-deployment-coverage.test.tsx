import { act, renderHook, waitFor } from "@testing-library/react";
import { vi } from "vitest";

const api = vi.hoisted(() => ({
  initManifest: vi.fn(), validateManifest: vi.fn(), buildBundle: vi.fn(), runPreflight: vi.fn(),
  fetchSecretsManifest: vi.fn(), createDeployment: vi.fn(), executeDeployment: vi.fn(),
  getDeployment: vi.fn(), findInProgressDeployment: vi.fn(),
}));
vi.mock("./lib/api", () => api);

import { useDeployment } from "./hooks/useDeployment";

describe("deployment wizard state machine", () => {
  beforeEach(() => {
    localStorage.clear();
    api.initManifest.mockResolvedValue({ manifest: null });
    api.validateManifest.mockResolvedValue({ issues: [], manifest: { scenario: { id: "demo" } } });
    api.buildBundle.mockResolvedValue({ artifact: { path: "/tmp/demo.tgz", sha256: "sha", size_bytes: 10 } });
    api.runPreflight.mockResolvedValue({ ok: true, checks: [] });
    api.fetchSecretsManifest.mockResolvedValue({ secrets: { bundle_secrets: [] } });
    api.createDeployment.mockResolvedValue({ deployment: { id: "dep-1" } });
    api.executeDeployment.mockResolvedValue({ ok: true });
    api.findInProgressDeployment.mockResolvedValue(null);
  });

  it("covers manifest history, validation, bundle/preflight, secrets, deployment, resets, and completion", async () => {
    const { result } = renderHook(() => useDeployment());
    await waitFor(() => expect(result.current.manifestJson).toBeTruthy());
    act(() => result.current.setManifestJson(JSON.stringify({ scenario: { id: "demo" }, target: { vps: { host: "example" } } })));
    act(() => result.current.undo());
    act(() => result.current.redo());
    await act(async () => { await result.current.validate(); });
    await act(async () => { await result.current.build(); });
    await act(async () => { await result.current.runPreflight(); });
    expect(result.current.bundleArtifact).not.toBeNull();
    act(() => result.current.setSSHKeyPath("/tmp/key"));
    act(() => result.current.goToStep(1));
    await act(async () => { await result.current.fetchSecrets(); });
    act(() => result.current.setProvidedSecrets("TOKEN", "value"));
    act(() => result.current.addCustomSecret());
    const customId = result.current.customSecrets[0]?.id;
    if (customId) {
      act(() => result.current.updateCustomSecret(customId, "key", "EXTRA_TOKEN"));
      act(() => result.current.updateCustomSecret(customId, "value", "extra"));
      act(() => result.current.removeCustomSecret(customId));
    }
    act(() => result.current.goNext());
    act(() => result.current.goBack());
    await act(async () => { await result.current.deploy(); });
    act(() => result.current.onDeploymentComplete(true));
    act(() => result.current.onDeploymentComplete(false, "failed"));
    act(() => result.current.resetManifestWithScenario("demo", { ui: 3000 }));
    act(() => result.current.resetManifestToDefaults());
    act(() => result.current.reset());
    expect(result.current.deploymentStatus).toBe("idle");
  });

  it("handles invalid input and API failures without leaving busy state latched", async () => {
    api.validateManifest.mockRejectedValue(new Error("validation down"));
    api.buildBundle.mockRejectedValue(new Error("build down"));
    api.runPreflight.mockRejectedValue(new Error("preflight down"));
    api.createDeployment.mockRejectedValue(new Error("deploy down"));
    const { result } = renderHook(() => useDeployment());
    await waitFor(() => expect(result.current.manifestJson).toBeTruthy());
    act(() => result.current.setManifestJson("{"));
    await act(async () => { await result.current.validate(); });
    expect(result.current.validationError).toMatch(/Invalid JSON/);
    await act(async () => { await result.current.build(); });
    expect(result.current.bundleError).toMatch(/Invalid JSON/);
    await act(async () => { await result.current.runPreflight(); });
    expect(result.current.preflightError).toMatch(/Invalid JSON/);
    await act(async () => { await result.current.deploy(); });
    expect(result.current.deploymentError).toMatch(/Invalid manifest/);

    act(() => result.current.setManifestJson(JSON.stringify({ scenario: { id: "demo" } })));
    await act(async () => { await result.current.validate(); });
    await act(async () => { await result.current.build(); });
    await act(async () => { await result.current.runPreflight(); });
    await act(async () => { await result.current.deploy(); });
    expect(result.current.validationError).toBe("validation down");
    expect(result.current.bundleError).toBe("build down");
    expect(result.current.preflightError).toBe("preflight down");
    expect(result.current.deploymentError).toBe("deploy down");
  });

  it("evaluates every wizard gate, required secret, override, and custom-secret validation", async () => {
    api.fetchSecretsManifest.mockResolvedValue({ secrets: {
      bundle_secrets: [{ id: "TOKEN", class: "user_prompt", required: true, target: { type: "env", name: "TOKEN" } }],
    } });
    api.runPreflight.mockResolvedValue({ ok: false, checks: [{ id: "ssh", status: "fail" }] });
    const { result } = renderHook(() => useDeployment());
    await waitFor(() => expect(result.current.manifestJson).toBeTruthy());
    act(() => result.current.setManifestJson(JSON.stringify({ scenario: { id: "demo" } })));
    act(() => result.current.goToStep(1));
    expect(result.current.canProceed).toBe(false);
    await act(async () => { await result.current.fetchSecrets(); });
    expect(result.current.canProceed).toBe(false);
    act(() => result.current.setProvidedSecrets("TOKEN", "secret"));
    expect(result.current.canProceed).toBe(true);
    act(() => result.current.addCustomSecret());
    expect(result.current.customSecretsValidation.isValid).toBe(false);
    const customSecret = result.current.customSecrets[0];
    if (!customSecret) throw new Error("expected custom secret");
    const customId = customSecret.id;
    act(() => result.current.updateCustomSecret(customId, "key", "bad-key"));
    expect(result.current.customSecretsValidation.isValid).toBe(false);
    act(() => result.current.updateCustomSecret(customId, "key", "VROOLI_INTERNAL_TEST"));
    expect(result.current.customSecretsValidation.errors[customId]).toMatch(/reserved/);
    act(() => result.current.updateCustomSecret(customId, "key", "VALID_KEY"));
    expect(result.current.customSecretsValidation.errors[customId]).toMatch(/Value is required/);
    act(() => result.current.updateCustomSecret(customId, "value", "value"));
    expect(result.current.customSecretsValidation.isValid).toBe(true);
    act(() => result.current.goToStep(2));
    expect(result.current.canProceed).toBe(false);
    act(() => result.current.setBundleArtifact({ path: "/tmp/a", sha256: "a", size_bytes: 1 }));
    expect(result.current.canProceed).toBe(true);
    act(() => result.current.goToStep(3));
    await act(async () => { await result.current.runPreflight(); });
    expect(result.current.canProceed).toBe(false);
    act(() => result.current.setPreflightOverride(true));
    expect(result.current.canProceed).toBe(true);
    act(() => result.current.goToStep(4));
    expect(result.current.canProceed).toBe(false);
    act(() => result.current.onDeploymentComplete(true));
    expect(result.current.canProceed).toBe(true);
  });

  it("reconnects saved deployments across active, completed, failed, and missing records", async () => {
    const states = [
      { status: "deploying", expected: "deploying" },
      { status: "deployed", expected: "success" },
      { status: "failed", expected: "failed" },
    ] as const;
    for (const item of states) {
      localStorage.setItem("scenario-to-cloud:deployment", JSON.stringify({
        manifestJson: JSON.stringify({ scenario: { id: "demo" } }), currentStep: 0,
        timestamp: Date.now(), deploymentId: "dep-1", deploymentStatus: "deploying",
      }));
      api.getDeployment.mockResolvedValue({ deployment: { status: item.status, error_message: "failed record" } });
      const { result, unmount } = renderHook(() => useDeployment());
      await waitFor(() => expect(result.current.deploymentStatus).toBe(item.expected));
      unmount();
    }
    localStorage.setItem("scenario-to-cloud:deployment", JSON.stringify({
      manifestJson: JSON.stringify({ scenario: { id: "demo" } }), currentStep: 0,
      timestamp: Date.now(), deploymentId: "gone", deploymentStatus: "deploying",
    }));
    api.getDeployment.mockRejectedValue(new Error("gone"));
    const { result } = renderHook(() => useDeployment());
    await waitFor(() => expect(result.current.deploymentStatus).toBe("idle"));
  });
});
