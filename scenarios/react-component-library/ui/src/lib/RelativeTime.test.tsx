import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RelativeTime } from "../../../library/components/RelativeTime/versions/1.0.5/RelativeTime";
import { renderWithProviders } from "../test-utils";

describe("RelativeTime", () => {
  it("keeps localized text distinct from the machine timestamp and absolute detail", () => {
    renderWithProviders(
      <RelativeTime
        value="3 days ago"
        dateTime="2026-11-19T12:30:00.000Z"
        title="Nov 19, 2026, 12:30 PM"
      />,
    );
    const time = screen.getByTestId("primitives.relative-time");
    expect(time).toHaveTextContent("3 days ago");
    expect(time).toHaveAttribute("dateTime", "2026-11-19T12:30:00.000Z");
    expect(time).toHaveAttribute("title", "Nov 19, 2026, 12:30 PM");
  });

  it("keeps the old value-only behavior for existing consumers", () => {
    renderWithProviders(<RelativeTime value="just now" />);
    const time = screen.getByTestId("primitives.relative-time");
    expect(time).toHaveTextContent("just now");
    expect(time).toHaveAttribute("dateTime", "just now");
  });
});
