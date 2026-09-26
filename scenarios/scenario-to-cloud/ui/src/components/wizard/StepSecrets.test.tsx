import "@testing-library/jest-dom";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { vi } from "vitest";

// provider-free-exception: secret-step behavior is exercised with mocked hooks and has no provider dependency.

import { StepSecrets } from "./StepSecrets";
import type { BundleSecretPlan, SecretsManifest } from "../../types/secrets";

type DeploymentProps = ComponentProps<typeof StepSecrets>["deployment"];

const plan = (overrides: Partial<BundleSecretPlan>): BundleSecretPlan => ({
  id: "secret-id",
  class: "user_prompt",
  required: true,
  target: { type: "env", name: "SECRET_KEY" },
  ...overrides,
});

function makeDeployment(overrides: Record<string, unknown> = {}): DeploymentProps {
  return {
    parsedManifest: { ok: true, value: {} },
    secretsManifest: null,
    secretsError: null,
    isFetchingSecrets: false,
    secretsFetched: false,
    fetchSecrets: vi.fn(),
    providedSecrets: {},
    setProvidedSecrets: vi.fn(),
    customSecrets: [],
    addCustomSecret: vi.fn(),
    removeCustomSecret: vi.fn(),
    updateCustomSecret: vi.fn(),
    customSecretsValidation: { errors: {}, isValid: true },
    ...overrides,
  } as unknown as DeploymentProps;
}

describe("StepSecrets", () => {
  it("loads requirements once and supports loading, retry, and no-secret states", async () => {
    const fetchSecrets = vi.fn();
    const deployment = makeDeployment({ parsedManifest: { ok: false, error: "manifest not ready" }, fetchSecrets });
    render(<StepSecrets deployment={deployment} />);
    expect(screen.getByRole("button", { name: "Fetch Secrets" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Fetch Secrets" }));
    expect(fetchSecrets).toHaveBeenCalled();

    const loading = makeDeployment({ isFetchingSecrets: true });
    const { rerender } = render(<StepSecrets deployment={loading} />);
    expect(screen.getByText("Loading secrets requirements...")).toBeInTheDocument();
    const failed = makeDeployment({ parsedManifest: { ok: false, error: "manifest not ready" }, secretsError: "service unavailable" });
    rerender(<StepSecrets deployment={failed} />);
    expect(screen.getByText("service unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(failed.fetchSecrets).toHaveBeenCalledOnce();
    const noSecrets = makeDeployment({ secretsFetched: true });
    rerender(<StepSecrets deployment={noSecrets} />);
    expect(screen.getByText("No Secrets Required")).toBeInTheDocument();
  });

  it("renders every secret class, required credential state, and summary", () => {
    const secretsManifest: SecretsManifest = {
      bundle_secrets: [
        plan({ id: "infra", class: "infrastructure", target: { type: "env", name: "VROOLI_ENV" }, description: "Runtime environment" }),
        plan({ id: "auto", class: "per_install_generated", target: { type: "env", name: "AUTO_SECRET" } }),
        plan({ id: "user", class: "user_prompt", target: { type: "env", name: "API_KEY" }, prompt: { label: "API key", description: "Used for API access" } }),
        plan({ id: "remote", class: "remote_fetch", target: { type: "file", name: "/tmp/token" } }),
      ],
      summary: { total_secrets: 4, infrastructure: 1, per_install_generated: 1, user_prompt: 1, remote_fetch: 1 },
    };
    const setProvidedSecrets = vi.fn();
    const deployment = makeDeployment({ secretsFetched: true, secretsManifest, setProvidedSecrets, providedSecrets: { API_KEY: "already-set" } });
    render(<StepSecrets deployment={deployment} />);
    expect(screen.getByText("Infrastructure")).toBeInTheDocument();
    expect(screen.getByText("Auto-Generated")).toBeInTheDocument();
    expect(screen.getByText("Required Credentials")).toBeInTheDocument();
    expect(screen.getByText("Remote Fetch")).toBeInTheDocument();
    expect(screen.getByText("VROOLI_ENV")).toBeInTheDocument();
    expect(screen.getByText("AUTO_SECRET")).toBeInTheDocument();
    expect(screen.getByLabelText("API key")).toHaveValue("already-set");
    fireEvent.change(screen.getByLabelText("API key"), { target: { value: "new-value" } });
    expect(setProvidedSecrets).toHaveBeenCalledWith("API_KEY", "new-value");
    expect(screen.getByText(/All required secrets must be provided/)).toBeInTheDocument();
  });

  it("adds, edits, validates, and removes custom secrets", () => {
    const addCustomSecret = vi.fn();
    const removeCustomSecret = vi.fn();
    const updateCustomSecret = vi.fn();
    const deployment = makeDeployment({
      secretsFetched: true,
      secretsManifest: { bundle_secrets: [], summary: { total_secrets: 0, infrastructure: 0, per_install_generated: 0, user_prompt: 0, remote_fetch: 0 } },
      customSecrets: [
        { id: "invalid", key: "bad", value: "", description: "" },
        { id: "valid", key: "GOOD_KEY", value: "value", description: "desc" },
      ],
      customSecretsValidation: { errors: { invalid: "Key is invalid" }, isValid: false },
      addCustomSecret,
      removeCustomSecret,
      updateCustomSecret,
    });
    render(<StepSecrets deployment={deployment} />);
    expect(screen.getByText("Key is invalid")).toBeInTheDocument();
    expect(screen.getByText("Valid")).toBeInTheDocument();
    const keyInputs = screen.getAllByPlaceholderText("MY_API_KEY");
    const firstKeyInput = keyInputs[0];
    if (!firstKeyInput) throw new Error("expected custom secret key input");
    fireEvent.change(firstKeyInput, { target: { value: "new_key" } });
    expect(updateCustomSecret).toHaveBeenCalledWith("invalid", "key", "NEW_KEY");
    const valueInputs = screen.getAllByPlaceholderText("Enter secret value...");
    const firstValueInput = valueInputs[0];
    if (!firstValueInput) throw new Error("expected custom secret value input");
    fireEvent.change(firstValueInput, { target: { value: "secret-value" } });
    expect(updateCustomSecret).toHaveBeenCalledWith("invalid", "value", "secret-value");
    const removeButtons = screen.getAllByTitle("Remove secret");
    const firstRemoveButton = removeButtons[0];
    if (!firstRemoveButton) throw new Error("expected a custom secret remove button");
    fireEvent.click(firstRemoveButton);
    expect(removeCustomSecret).toHaveBeenCalledWith("invalid");
    fireEvent.click(screen.getByRole("button", { name: "Add Custom Secret" }));
    expect(addCustomSecret).toHaveBeenCalledOnce();
  });
});
