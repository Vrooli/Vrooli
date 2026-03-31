/**
 * Tests for DetailPageLayout.
 *
 * Verifies header rendering, body content, and mobile FAB/BottomSheet.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

  it("does not render FAB on desktop", () => {
    render(
      <DetailPageLayout
        header={<div>Header</div>}
        mobileActions={<div>Actions</div>}
      >
        <div>Body</div>
      </DetailPageLayout>,
    );

    expect(screen.queryByTestId("detail-mobile-actions-fab")).not.toBeInTheDocument();
  });

  it("renders FAB on mobile when mobileActions provided", () => {
    mockIsMobile = true;

    render(
      <DetailPageLayout
        header={<div>Header</div>}
        mobileActions={<div data-testid="actions-content">Actions</div>}
      >
        <div>Body</div>
      </DetailPageLayout>,
    );

    expect(screen.getByTestId("detail-mobile-actions-fab")).toBeInTheDocument();
  });

  it("does not render FAB on mobile when no mobileActions", () => {
    mockIsMobile = true;

    render(
      <DetailPageLayout header={<div>Header</div>}>
        <div>Body</div>
      </DetailPageLayout>,
    );

    expect(screen.queryByTestId("detail-mobile-actions-fab")).not.toBeInTheDocument();
  });

  it("opens BottomSheet when FAB is clicked", async () => {
    mockIsMobile = true;
    const user = userEvent.setup();

    render(
      <DetailPageLayout
        header={<div>Header</div>}
        mobileActions={<div data-testid="actions-content">Actions</div>}
        mobileActionsTitle="Test Actions"
      >
        <div>Body</div>
      </DetailPageLayout>,
    );

    await user.click(screen.getByTestId("detail-mobile-actions-fab"));

    expect(screen.getByTestId("actions-content")).toBeInTheDocument();
  });
});
