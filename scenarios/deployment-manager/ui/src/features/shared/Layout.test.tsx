import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { Layout } from "./Layout";

describe("Layout", () => {
  it("renders the layout with header", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getByText("Deployment Manager")).toBeDefined();
  });

  it("renders navigation items", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getByText("Dashboard")).toBeDefined();
    expect(screen.getByText("Profiles")).toBeDefined();
    expect(screen.getByText("Analyze")).toBeDefined();
    expect(screen.getByText("Deployments")).toBeDefined();
  });

  it("renders children content", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getByText("Test Content")).toBeDefined();
  });
});
