import { afterEach, describe, it, expect } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { Layout } from "./Layout";

afterEach(() => cleanup());

describe("Layout", () => {
  it("renders the layout with header", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getAllByText("Deployment Manager").length).toBeGreaterThan(0);
  });

  it("renders navigation items", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getAllByText("Dashboard").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Profiles").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Analyze").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Deployments").length).toBeGreaterThan(0);
  });

  it("renders children content", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getByText("Test Content")).toBeDefined();
  });
});
