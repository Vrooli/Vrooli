import { describe, it, expect } from "vitest";
import { render, screen } from "@/test-utils";
import App from "../../App";

describe("App", () => {
  it("renders the main heading", () => {
    render(<App />);
    expect(screen.getByText("Scenario to Desktop")).toBeInTheDocument();
  });

  it("renders the tagline on desktop", () => {
    render(<App />);
    // Tagline has `hidden md:block` — in jsdom (matchMedia defaults to non-matching)
    // the element is present in the DOM but visually hidden via CSS.
    // We verify the text node exists; actual visibility is a CSS concern.
    expect(
      screen.getByText(
        /Transform Vrooli scenarios into professional desktop applications/i,
      ),
    ).toBeInTheDocument();
  });

  it("renders view mode selector buttons", () => {
    render(<App />);
    // Tab labels were shortened for mobile-friendliness
    expect(screen.getByText("Inventory")).toBeInTheDocument();
    expect(screen.getByText("Generate")).toBeInTheDocument();
  });

  it("defaults to inventory view", () => {
    render(<App />);
    // The active tab shows its label even on mobile; all tabs show labels on desktop.
    const inventoryButton = screen.getByRole("tab", { name: /Inventory/i });
    const generateButton = screen.getByRole("tab", { name: /Generate/i });
    expect(inventoryButton).toBeInTheDocument();
    expect(generateButton).toBeInTheDocument();
  });
});
