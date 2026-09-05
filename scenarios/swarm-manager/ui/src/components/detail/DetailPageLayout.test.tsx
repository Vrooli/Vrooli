/**
 * Tests for DetailPageLayout.
 *
 * Verifies header/body rendering and that no legacy mobile FAB exists —
 * actions belong to the DetailPageHeader overflow menu.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { DetailPageLayout } from "./DetailPageLayout";

let mockIsMobile = false;

beforeEach(() => {
  mockIsMobile = false;

  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: query === "(max-width: 768px)" ? mockIsMobile : false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

describe("DetailPageLayout", () => {
  it("renders header and body content", () => {
    render(
      <DetailPageLayout header={<div data-testid="test-header">Header</div>}>
        <div data-testid="test-body">Body</div>
      </DetailPageLayout>,
    );

    expect(screen.getByTestId("detail-page-layout")).toBeInTheDocument();
    expect(screen.getByTestId("test-header")).toBeInTheDocument();
    expect(screen.getByTestId("test-body")).toBeInTheDocument();
  });

  it("applies body class overrides", () => {
    render(
      <DetailPageLayout header={<div>Header</div>} bodyClassName="test-body-class">
        <div data-testid="test-body">Body</div>
      </DetailPageLayout>,
    );

    expect(screen.getByTestId("test-body").parentElement).toHaveClass("test-body-class");
  });

  it("keeps body gutters by default", () => {
    render(
      <DetailPageLayout header={<div>Header</div>}>
        <div data-testid="test-body">Body</div>
      </DetailPageLayout>,
    );

    expect(screen.getByTestId("test-body").parentElement).toHaveClass("px-2");
    expect(screen.getByTestId("detail-page-layout")).not.toHaveAttribute("data-full-bleed");
  });

  it("drops body gutters and bounds the height when full-bleed", () => {
    // The Files tab renders its own header; page gutters left it inset from
    // the tab bar and stopped it filling the available height.
    render(
      <DetailPageLayout header={<div>Header</div>} fullBleed>
        <div data-testid="test-body">Body</div>
      </DetailPageLayout>,
    );

    const body = screen.getByTestId("test-body").parentElement;
    expect(body).not.toHaveClass("px-2");
    expect(body).toHaveClass("min-h-0");
    expect(screen.getByTestId("detail-page-layout")).toHaveClass("h-full", "overflow-hidden");
  });

  it("never renders the legacy mobile actions FAB", () => {
    mockIsMobile = true;

    render(
      <DetailPageLayout header={<div>Header</div>}>
        <div>Body</div>
      </DetailPageLayout>,
    );

    expect(screen.queryByTestId("detail-mobile-actions-fab")).not.toBeInTheDocument();
  });
});
