import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { StatusSparkline } from "./StatusSparkline";

describe("StatusSparkline", () => {
  it("renders recent status bars and the empty state", () => {
    const { container, rerender } = renderWithProviders(
      <StatusSparkline statuses={["ok", "warning", "critical"]} maxBars={2} barHeight={20} />,
    );
    expect(container.querySelectorAll(".w-1\\.5")).toHaveLength(2);

    rerender(<StatusSparkline statuses={[]} />);
    expect(screen.getByText("No data")).toBeInTheDocument();
  });
});
