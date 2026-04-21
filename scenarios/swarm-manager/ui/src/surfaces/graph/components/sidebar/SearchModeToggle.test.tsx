/**
 * Tests for SearchModeToggle. Verifies mode switching and the disabled state
 * when AI search is unavailable.
 */

import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SearchModeToggle } from "./SearchModeToggle";

describe("SearchModeToggle", () => {
  it("calls onChange with 'ai' when the AI button is clicked", async () => {
    const onChange = vi.fn();
    render(<SearchModeToggle mode="plain" onChange={onChange} aiAvailable={true} />);
    const user = userEvent.setup();
    await user.click(screen.getByTestId("search-mode-ai"));
    expect(onChange).toHaveBeenCalledWith("ai");
  });

  it("reflects the active mode via aria-pressed", () => {
    render(<SearchModeToggle mode="ai" onChange={() => {}} aiAvailable={true} />);
    expect(screen.getByTestId("search-mode-ai")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("search-mode-plain")).toHaveAttribute("aria-pressed", "false");
  });

  it("disables the AI button when AI search is unavailable", async () => {
    const onChange = vi.fn();
    render(
      <SearchModeToggle
        mode="plain"
        onChange={onChange}
        aiAvailable={false}
        unavailableReason="Qdrant not reachable"
      />,
    );
    const aiBtn = screen.getByTestId("search-mode-ai");
    expect(aiBtn).toBeDisabled();
    expect(aiBtn).toHaveAttribute("title", "Qdrant not reachable");
    const user = userEvent.setup();
    await user.click(aiBtn);
    expect(onChange).not.toHaveBeenCalled();
  });
});
