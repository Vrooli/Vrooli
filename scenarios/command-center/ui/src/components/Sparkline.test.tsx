import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { Sparkline } from "./Sparkline";

describe("Sparkline", () => {
  it("draws an authored series dashed with hollow points, so it reads as a drawing", () => {
    const { container } = renderWithProviders(<Sparkline series={[1, 4, 2, 8]} illustrative />);
    expect(container.querySelector("path")).toHaveAttribute("stroke-dasharray", "3 4");
    expect(container.querySelectorAll("circle[fill='none']")).toHaveLength(4);
  });
  it("draws a measured series solid", () => {
    const { container } = renderWithProviders(<Sparkline series={[1, 4]} illustrative={false} />);
    expect(container.querySelector("path")).not.toHaveAttribute("stroke-dasharray");
  });
  it("renders nothing for a single point", () => {
    const { container } = renderWithProviders(<Sparkline series={[3]} illustrative />);
    expect(container.querySelector("svg")).toBeNull();
  });
});
