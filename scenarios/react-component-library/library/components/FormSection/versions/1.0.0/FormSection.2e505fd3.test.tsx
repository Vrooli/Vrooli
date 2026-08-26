import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FormSection } from "./FormSection.tsx";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

describe("FormSection", () => {
  it("renders authored metadata and toggles a collapsible section", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderWithProviders(
      <FormSection
        title="Offer rules"
        description="Record the revisit condition."
        summary="Required fields"
        errorCount={2}
        collapsible
        defaultOpen
        onOpenChange={onOpenChange}
      >
        <label htmlFor="rule">Threshold</label>
        <input id="rule" />
      </FormSection>,
      { withoutRouter: true },
    );

    expect(screen.getByText(/Record the revisit condition\./)).toBeVisible();
    expect(screen.getByRole("status")).toHaveTextContent("2");
    const toggle = screen.getByRole("button", { name: /Collapse Offer rules/ });
    await user.click(toggle);
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(screen.queryByLabelText(/Threshold/)).not.toBeInTheDocument();
  });

  it("renders non-collapsible actions without a toggle", () => {
    renderWithProviders(
      <FormSection title="Always open" actions={<button type="button">Add</button>}>
        <p>Content</p>
      </FormSection>,
      { withoutRouter: true },
    );

    expect(screen.getByRole("button", { name: /Add/ })).toBeVisible();
    expect(screen.queryByRole("button", { name: /Collapse Always open/ })).not.toBeInTheDocument();
  });
});
