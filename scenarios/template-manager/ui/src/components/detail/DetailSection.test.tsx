import { fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { renderWithProviders } from "../../test-utils";

import { DetailSection } from "./DetailSection";

describe("DetailSection", () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it("renders a static section (no toggle) without a storageKey", () => {
    renderWithProviders(
      <DetailSection title="Overview" testId="overview">
        <p data-testid="body">Body content</p>
      </DetailSection>,
    );
    expect(screen.getByTestId("body")).toBeInTheDocument();
    expect(screen.queryByTestId("overview-toggle")).not.toBeInTheDocument();
  });

  it("collapses and expands when given a storageKey", () => {
    renderWithProviders(
      <DetailSection title="Findings" testId="findings" storageKey="findings-test">
        <p data-testid="body">Finding body</p>
      </DetailSection>,
    );
    const toggle = screen.getByTestId("findings-toggle");
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId("body")).toBeInTheDocument();

    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("body")).not.toBeInTheDocument();
  });

  it("honors defaultOpen=false", () => {
    renderWithProviders(
      <DetailSection title="Scope" testId="scope" storageKey="scope-test" defaultOpen={false}>
        <p data-testid="body">Scope body</p>
      </DetailSection>,
    );
    expect(screen.getByTestId("scope-toggle")).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("body")).not.toBeInTheDocument();
  });
});
