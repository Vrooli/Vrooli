import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { StaleBadge } from "./StaleBadge";

describe("StaleBadge", () => {
  it("renders nothing when ts is null", () => {
    const { container } = render(<StaleBadge ts={null} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders nothing when ts is undefined", () => {
    const { container } = render(<StaleBadge ts={undefined} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders STALE with a title when ts is set", () => {
    render(<StaleBadge ts="2026-04-18T12:00:00Z" />);
    const badge = screen.getByTestId("stale-badge");
    expect(badge.textContent).toBe("STALE");
    expect(badge.getAttribute("title")).toContain("2026-04-18T12:00:00Z");
  });
});
