import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { CatalogResultPanel } from "./ComponentsCatalogResultPanel";

describe("CatalogResultPanel", () => {
  it("renders the filtered-empty result as a labelled RCL empty state", () => {
    renderWithProviders(
      <CatalogResultPanel
        catalog={{ isLoading: false, isError: false }}
        group="components"
        filtered={[]}
        sort="name"
        visible={[]}
        rows={new Map()}
        domains={[]}
        view="gallery"
        setLimit={vi.fn()}
      />,
    );

    const region = screen.getByTestId("feedback.empty-state");
    const title = screen.getByRole("heading", {
      level: 2,
      name: strings.workbench.components.emptyTitle,
    });
    expect(region).toHaveAttribute("role", "region");
    expect(region).toHaveAttribute("data-density", "compact");
    expect(region).toHaveAttribute("aria-labelledby", title.id);
    expect(title).toBeVisible();
    expect(screen.getByText(strings.workbench.components.emptyBody)).toBeVisible();
  });
});
