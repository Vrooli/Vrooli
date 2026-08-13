import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { AsyncPanel } from "../components/AsyncPanel/versions/1.0.0/AsyncPanel";

describe("AsyncPanel", () => {
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
});
