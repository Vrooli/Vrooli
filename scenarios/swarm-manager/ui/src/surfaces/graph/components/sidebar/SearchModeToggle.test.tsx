/**
 * Tests for SearchModeToggle. Verifies mode switching and the disabled state
 * when AI search is unavailable.
 */

import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SearchModeToggle } from "./SearchModeToggle";

describe("SearchModeToggle", () => {
  it("calls onChange with 'ai' when the inline AI toggle is clicked from plain mode", async () => {
    const onChange = vi.fn();
    render(<SearchModeToggle mode="plain" onChange={onChange} aiAvailable={true} />);
    const user = userEvent.setup();
    await user.click(screen.getByTestId("search-mode-ai"));
    expect(onChange).toHaveBeenCalledWith("ai");
  });

  it("reflects the active mode via aria-pressed", () => {
    render(<SearchModeToggle mode="ai" onChange={() => {}} aiAvailable={true} />);
    expect(screen.getByTestId("search-mode-ai")).toHaveAttribute("aria-pressed", "true");
  });

  it("calls onChange with 'plain' when clicked from AI mode", async () => {
    const onChange = vi.fn();
    render(<SearchModeToggle mode="ai" onChange={onChange} aiAvailable={true} />);
    const user = userEvent.setup();
    await user.click(screen.getByTestId("search-mode-ai"));
    expect(onChange).toHaveBeenCalledWith("plain");
  });

  it("hides the toggle when AI search is unavailable", () => {
    render(
      <SearchModeToggle
        mode="plain"
        onChange={vi.fn()}
        aiAvailable={false}
        unavailableReason="Qdrant not reachable"
      />,
    );
    expect(screen.queryByTestId("search-mode-ai")).not.toBeInTheDocument();
  });
});
