import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ResourceTreeItem } from "./ResourceTreeItem";
import { SecretListItem } from "./SecretListItem";

const secret = {
  resource_name: "vault",
  secret_key: "VAULT_TOKEN",
  secret_type: "string",
  required: true,
  classification: "secret",
  handling_strategy: "",
  requires_user_input: false
};

describe("ResourceTreeItem", () => {
  afterEach(cleanup);

  it("expands resources and routes resource and secret exclusion controls", () => {
    const onToggleExpand = vi.fn();
    const onToggleResourceExclusion = vi.fn();
    const onSelectSecret = vi.fn();
    const onToggleSecretExclusion = vi.fn();
    renderWithProviders(
      <ResourceTreeItem
        group={{ resourceName: "vault", secrets: [secret], strategizedCount: 0, totalCount: 1, blockingCount: 1, hasBlockers: true, isFullyExcluded: false }}
        isExpanded
        selectedSecretId={null}
        expandedResources={new Set()}
        excludedSecrets={new Set()}
        overriddenSecrets={new Set()}
        onToggleExpand={onToggleExpand}
        onToggleResourceExclusion={onToggleResourceExclusion}
        onSelectSecret={onSelectSecret}
        onToggleSecretExclusion={onToggleSecretExclusion}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: /vault/ }));
    fireEvent.click(screen.getByTitle("Exclude resource"));
    fireEvent.click(screen.getByRole("button", { name: /VAULT_TOKEN/ }));
    fireEvent.click(screen.getByTitle("Exclude from export"));
    expect(onToggleExpand).toHaveBeenCalledOnce();
    expect(onToggleResourceExclusion).toHaveBeenCalledOnce();
    expect(onSelectSecret).toHaveBeenCalledWith("vault", "VAULT_TOKEN");
    expect(onToggleSecretExclusion).toHaveBeenCalledWith("vault", "VAULT_TOKEN");
  });
});

describe("SecretListItem", () => {
  afterEach(cleanup);

  it("represents selected overrides and excluded defaults", () => {
    const onSelect = vi.fn();
    const onToggleExclude = vi.fn();
    const { rerender } = renderWithProviders(
      <SecretListItem secret={{ ...secret, handling_strategy: "prompt" }} isSelected isExcluded={false} isOverridden onSelect={onSelect} onToggleExclude={onToggleExclude} />
    );
    expect(screen.getByTitle("Has scenario override")).toBeInTheDocument();
    expect(screen.getByText("prompt")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /VAULT_TOKEN/ }));
    expect(onSelect).toHaveBeenCalledOnce();

    rerender(<SecretListItem secret={secret} isSelected={false} isExcluded isOverridden={false} onSelect={onSelect} onToggleExclude={onToggleExclude} />);
    expect(screen.getByTitle("Include in export")).toBeInTheDocument();
    expect(screen.getByTitle("Blocking - no strategy")).toBeInTheDocument();
    fireEvent.click(screen.getByTitle("Include in export"));
    expect(onToggleExclude).toHaveBeenCalledOnce();
  });
});
