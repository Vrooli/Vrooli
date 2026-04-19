import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { GapBadge } from "./GapBadge";

describe("GapBadge", () => {
  it("renders the GAP label with data-testid when status is gap", () => {
    render(<GapBadge status="gap" whatIsNeeded="Need backend endpoint" />);
    const badge = screen.getByTestId("gap-badge");
    expect(badge).toBeInTheDocument();
    expect(badge.textContent).toBe("GAP");
    expect(badge.getAttribute("title")).toBe("Need backend endpoint");
  });

  it("renders the PARTIAL label when status is partial", () => {
    render(<GapBadge status="partial" />);
    const badge = screen.getByTestId("gap-badge");
    expect(badge.textContent).toBe("PARTIAL");
  });

  it("renders nothing when status is live", () => {
    const { container } = render(<GapBadge status="live" />);
    expect(container.firstChild).toBeNull();
    expect(screen.queryByTestId("gap-badge")).toBeNull();
  });
});
