import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ManifestSecretDetailPanel } from "./ManifestSecretDetailPanel";

const secret = {
  resource_name: "vault",
  secret_key: "VAULT_TOKEN",
  secret_type: "token",
  required: true,
  classification: "service",
  description: "Vault access token",
  handling_strategy: "prompt",
  requires_user_input: true,
  prompt: { label: "Vault token", description: "Enter the token" }
};

describe("ManifestSecretDetailPanel", () => {
  it("prompts the operator to select a secret before showing controls", () => {
    renderWithProviders(
      <ManifestSecretDetailPanel
        secret={null}
        isOverridden={false}
        isExcluded={false}
        isSaving={false}
        isDeleting={false}
        onUpdatePendingChange={() => {}}
        onSave={() => {}}
        onReset={() => {}}
        onToggleExclude={() => {}}
      />
    );
    expect(screen.getByText("Select a secret to view details")).toBeInTheDocument();
  });

  it("edits prompt strategy fields and routes save, reset, exclusion, and resource handoff actions", () => {
    const onUpdatePendingChange = vi.fn();
    const onSave = vi.fn();
    const onReset = vi.fn();
    const onToggleExclude = vi.fn();
    const onOpenInResourcePanel = vi.fn();
    renderWithProviders(
      <ManifestSecretDetailPanel
        secret={secret}
        isOverridden
        isExcluded={false}
        pendingEdit={{ original: secret, changes: { handling_strategy: "prompt" }, isDirty: true }}
        isSaving={false}
        isDeleting={false}
        onUpdatePendingChange={onUpdatePendingChange}
        onSave={onSave}
        onReset={onReset}
        onToggleExclude={onToggleExclude}
        onOpenInResourcePanel={onOpenInResourcePanel}
      />
    );

    fireEvent.change(screen.getByDisplayValue("Vault token"), { target: { value: "Desktop token" } });
    expect(onUpdatePendingChange).toHaveBeenCalledWith({ prompt_label: "Desktop token" });
    fireEvent.click(screen.getByText("Requires user input (can't be skipped)"));
    expect(onUpdatePendingChange).toHaveBeenCalledWith({ requires_user_input: false });
    fireEvent.click(screen.getByText("Exclude"));
    expect(onToggleExclude).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText("Open"));
    expect(onOpenInResourcePanel).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText("Save Override"));
    expect(onSave).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText("Reset to Default"));
    expect(onReset).toHaveBeenCalledOnce();
  });

  it("accepts valid generator templates and keeps invalid JSON out of pending changes", () => {
    const onUpdatePendingChange = vi.fn();
    renderWithProviders(
      <ManifestSecretDetailPanel
        secret={{ ...secret, handling_strategy: "generate", requires_user_input: false }}
        isOverridden={false}
        isExcluded={false}
        isSaving={false}
        isDeleting={false}
        onUpdatePendingChange={onUpdatePendingChange}
        onSave={() => {}}
        onReset={() => {}}
        onToggleExclude={() => {}}
      />
    );
    const template = screen.getByPlaceholderText(/alphanumeric/);
    fireEvent.change(template, { target: { value: '{"length":32}' } });
    expect(onUpdatePendingChange).toHaveBeenCalledWith({ generator_template: { length: 32 } });
    onUpdatePendingChange.mockClear();
    fireEvent.change(template, { target: { value: "{" } });
    expect(onUpdatePendingChange).not.toHaveBeenCalled();
    fireEvent.change(template, { target: { value: "  " } });
    expect(onUpdatePendingChange).toHaveBeenCalledWith({ generator_template: undefined });
  });

  it("explains strip and delegate choices while respecting disabled save and reset states", () => {
    const { rerender } = renderWithProviders(
      <ManifestSecretDetailPanel
        secret={{ ...secret, classification: "infrastructure", description: "", handling_strategy: "strip", required: false, tier_strategies: { "tier-1-local": "delegate" } }}
        isOverridden
        isExcluded
        pendingEdit={{ original: secret, changes: { handling_strategy: "strip" }, isDirty: false }}
        isSaving
        isDeleting
        onUpdatePendingChange={() => {}}
        onSave={() => {}}
        onReset={() => {}}
        onToggleExclude={() => {}}
      />
    );
    expect(screen.getByText(/excluded from the deployment bundle/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Reverting..." })).toBeDisabled();
    expect(screen.getByText("Include")).toBeInTheDocument();
    expect(screen.getByText("tier-1-local")).toBeInTheDocument();

    rerender(
      <ManifestSecretDetailPanel
        secret={{ ...secret, classification: "user", handling_strategy: "delegate", tier_strategies: undefined }}
        isOverridden={false}
        isExcluded={false}
        isSaving={false}
        isDeleting={false}
        onUpdatePendingChange={() => {}}
        onSave={() => {}}
        onReset={() => {}}
        onToggleExclude={() => {}}
      />
    );
    expect(screen.getByText(/managed by the cloud provider/)).toBeInTheDocument();
    expect(screen.queryByText("Other Tier Strategies")).not.toBeInTheDocument();
  });
});
