import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ResourceTree } from "./ResourceTree";

const secret = {
  resource_name: "vault",
  secret_key: "VAULT_TOKEN",
  secret_type: "token",
  required: true,
  classification: "service",
  handling_strategy: "prompt",
  requires_user_input: true
};

describe("ResourceTree", () => {
  afterEach(cleanup);

  it("explains an empty filter result", () => {
    renderWithProviders(
      <ResourceTree
        groups={[]}
        expandedResources={new Set()}
        selectedSecretId={null}
        excludedResources={new Set()}
        excludedSecrets={new Set()}
        overriddenSecrets={new Set()}
        onToggleResource={vi.fn()}
        onToggleResourceExclusion={vi.fn()}
        onSelectSecret={vi.fn()}
        onToggleSecretExclusion={vi.fn()}
      />
    );

    expect(screen.getByText("No resources match the current filter.")).toBeInTheDocument();
  });

  it("routes an expanded resource and selected secret through the tree", () => {
    const onToggleResource = vi.fn();
    const onToggleResourceExclusion = vi.fn();
    const onSelectSecret = vi.fn();
    const onToggleSecretExclusion = vi.fn();
    renderWithProviders(
      <ResourceTree
        groups={[{ resourceName: "vault", secrets: [secret], strategizedCount: 1, totalCount: 1, blockingCount: 0, hasBlockers: false, isFullyExcluded: false }]}
        expandedResources={new Set(["vault"])}
        selectedSecretId={{ resource: "vault", key: "VAULT_TOKEN" }}
        excludedResources={new Set()}
        excludedSecrets={new Set()}
        overriddenSecrets={new Set(["vault:VAULT_TOKEN"])}
        onToggleResource={onToggleResource}
        onToggleResourceExclusion={onToggleResourceExclusion}
        onSelectSecret={onSelectSecret}
        onToggleSecretExclusion={onToggleSecretExclusion}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: /vault/ }));
    fireEvent.click(screen.getByTitle("Exclude resource"));
    fireEvent.click(screen.getByRole("button", { name: /VAULT_TOKEN/ }));
    fireEvent.click(screen.getByTitle("Exclude from export"));
    expect(onToggleResource).toHaveBeenCalledWith("vault");
    expect(onToggleResourceExclusion).toHaveBeenCalledWith("vault");
    expect(onSelectSecret).toHaveBeenCalledWith("vault", "VAULT_TOKEN");
    expect(onToggleSecretExclusion).toHaveBeenCalledWith("vault", "VAULT_TOKEN");
  });
});
