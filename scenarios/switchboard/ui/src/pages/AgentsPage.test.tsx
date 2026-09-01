import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { AgentsPage } from "./AgentsPage";

describe("AgentsPage", () => {
  // [REQ:SWBD-P1-008]
  it("requires a description and confirms a draft before writing", () => {
    renderWithProviders(<AgentsPage />);

    fireEvent.click(screen.getByRole("button", { name: strings.console.agents.prepareDraft }));
    expect(screen.queryByText(/Review before writing/)).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(strings.console.agents.descriptionLabel), {
      target: { value: "  A household planning assistant  " },
    });
    fireEvent.click(screen.getByRole("button", { name: strings.console.agents.prepareDraft }));
    expect(screen.getAllByText(/A household planning assistant/)).toHaveLength(2);

    fireEvent.click(screen.getByRole("button", { name: strings.console.agents.confirmWrite }));
    expect(screen.getByText(strings.console.agents.confirmed)).toBeInTheDocument();
  });
});
