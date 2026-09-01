import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { Figure } from "./Figure";

describe("Figure — the ink is on the figure itself", () => {
  it("stamps the ink as data so a greyscale render stays unambiguous", () => {
    renderWithProviders(<Figure value={12400} format="currency.compact" unit="usd" ink="hollow" scale="wall" />);
    const figure = screen.getByLabelText("$12.4k").closest("[data-figure]");
    expect(figure).toHaveAttribute("data-ink", "hollow");
    expect(figure).toHaveAttribute("data-value", "12400");
  });
  it("draws a placeholder frame when there is nothing to show", () => {
    renderWithProviders(<Figure value={null} ink="unavailable" scale="display" placeholder="––" />);
    expect(screen.getByLabelText("––")).toBeInTheDocument();
  });
  it("renders every glyph as its own span so only changed digits can roll", () => {
    const { container } = renderWithProviders(<Figure value={1284} format="integer" ink="solid" scale="display" />);
    expect(container.querySelectorAll(".cc-digit")).toHaveLength(5);
  });
});

describe("Figure — rolling digits", () => {
  it("marks only the digits that changed so the rest hold still", async () => {
    const { rerender, container } = renderWithProviders(<Figure value={1284} format="integer" ink="solid" scale="display" />);
    rerender(<Figure value={1294} format="integer" ink="solid" scale="display" />);
    await new Promise((resolve) => setTimeout(resolve, 0));
    const rolled = Array.from(container.querySelectorAll(".cc-digit-roll")).map((node) => node.textContent);
    expect(rolled).toEqual(["9"]);
  });
  it("rolls every digit when the figure grows a place", async () => {
    const { rerender, container } = renderWithProviders(<Figure value={99} format="integer" ink="solid" scale="display" />);
    rerender(<Figure value={100} format="integer" ink="solid" scale="display" />);
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(container.querySelectorAll(".cc-digit-roll")).toHaveLength(3);
  });
});
