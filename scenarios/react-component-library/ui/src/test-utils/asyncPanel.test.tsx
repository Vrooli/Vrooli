import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import { AsyncPanel as AssetDetailAsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import { AsyncPanel as InspectorAsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import { AssetDetailShell } from "@vrooli/react-component-library/AssetDetailShell/1";
import { InspectorLayout } from "@vrooli/react-component-library/InspectorLayout/1";

// Fixture children and authored fallbacks, named once so the queries match
// the fixture instead of restating its copy.
const loadedContent = "Loaded content";
const emptyFallback = "No history yet";
const partialFallback = "Partial history";
const errorFallback = "Unavailable";

describe("AsyncPanel", () => {
  it.each([
    ["loading", "Loading…"],
    ["empty", "Nothing to show yet."],
  ] as const)("renders the %s fallback", (state, message) => {
    renderWithProviders(<AsyncPanel surfaceId="history" state={state} />);
    expect(screen.getAllByText(message).length).toBeGreaterThan(0);
  });

  it.each(["ready", "static"] as const)("renders children for the %s state", (state) => {
    renderWithProviders(
      <AsyncPanel surfaceId="history" state={state}>
        <strong>{loadedContent}</strong>
      </AsyncPanel>,
    );
    expect(screen.getByText(loadedContent)).toBeVisible();
  });

  it("uses authored content for empty and partial states", () => {
    const { rerender } = renderWithProviders(
      <AsyncPanel
        surfaceId="history"
        state="empty"
        empty={<strong>{emptyFallback}</strong>}
        partial={<strong>{partialFallback}</strong>}
      />,
    );
    expect(screen.getByText(emptyFallback)).toBeVisible();
    rerender(
      <AsyncPanel
        surfaceId="history"
        state="partial"
        empty={<strong>{emptyFallback}</strong>}
        partial={<strong>{partialFallback}</strong>}
      />,
    );
    expect(screen.getByText(partialFallback)).toBeVisible();
  });

  it("preserves semantic lifecycle telemetry for partial content", () => {
    renderWithProviders(<AsyncPanel surfaceId="history" state="partial" />);
    expect(screen.getByRole("status")).toHaveAccessibleName("Some information is unavailable.");
    expect(document.querySelector('[data-experience-surface="history"]')).toHaveAttribute(
      "data-experience-state",
      "partial",
    );
  });

  it("provides a retry affordance for its fallback error state", () => {
    const retry = vi.fn();
    renderWithProviders(<AsyncPanel surfaceId="history" state="error" onRetry={retry} />);
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(retry).toHaveBeenCalledOnce();
  });

  it("renders authored error content without a retry button", () => {
    renderWithProviders(
      <AsyncPanel surfaceId="history" state="error" error={<strong>{errorFallback}</strong>} />,
    );
    expect(screen.getByText(errorFallback)).toBeVisible();
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it.each([AsyncPanel, AssetDetailAsyncPanel, InspectorAsyncPanel])(
    "keeps the adopted AsyncPanel copy's lifecycle contract intact",
    (Panel) => {
      const { rerender, container } = renderWithProviders(
        <Panel surfaceId="adopted" state="loading" />,
      );
      expect(container.querySelector('[data-experience-state="loading"]')).toBeInTheDocument();

      rerender(
        <Panel surfaceId="adopted" state="loading" loading={<span>Loading details</span>} />,
      );
      expect(container).toHaveTextContent("Loading details");

      rerender(
        <Panel surfaceId="adopted" state="ready">
          <span>Ready content</span>
        </Panel>,
      );
      expect(container).toHaveTextContent("Ready content");

      rerender(<Panel surfaceId="adopted" state="empty" />);
      expect(container).toHaveTextContent("Nothing to show yet.");

      rerender(<Panel surfaceId="adopted" state="partial" />);
      expect(container).toHaveTextContent("Some information is unavailable.");

      rerender(<Panel surfaceId="adopted" state="error" />);
      expect(container).toHaveTextContent("This section could not be loaded.");

      rerender(<Panel surfaceId="adopted" state="error" onRetry={() => undefined} />);
      expect(container).toHaveTextContent("Retry");

      rerender(<Panel surfaceId="adopted" state="error" error={<span>Try again later</span>} />);
      expect(container).toHaveTextContent("Try again later");

      rerender(
        <Panel surfaceId="adopted" state="static">
          <span>Static content</span>
        </Panel>,
      );
      expect(container).toHaveTextContent("Static content");
    },
  );

  it("keeps optional shell actions and toolbars accessible", () => {
    const { rerender } = renderWithProviders(
      <AssetDetailShell
        title="Button"
        preview={<div>Preview</div>}
        metadata={<div>Metadata</div>}
      />,
    );
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    rerender(
      <AssetDetailShell
        title="Button"
        preview={<div>Preview</div>}
        metadata={<div>Metadata</div>}
        actions={<button type="button">Save</button>}
      />,
    );
    expect(screen.getByRole("button", { name: "Save" })).toBeVisible();

    rerender(
      <InspectorLayout
        title="Inspector"
        canvas={<div>Canvas</div>}
        inspector={<div>Inspector</div>}
      />,
    );
    expect(screen.getByRole("heading", { name: "Inspector" })).toBeInTheDocument();
    rerender(
      <InspectorLayout
        title="Inspector"
        canvas={<div>Canvas</div>}
        inspector={<div>Inspector</div>}
        toolbar={<button type="button">Reset</button>}
      />,
    );
    expect(screen.getByRole("button", { name: "Reset" })).toBeVisible();
  });
});
