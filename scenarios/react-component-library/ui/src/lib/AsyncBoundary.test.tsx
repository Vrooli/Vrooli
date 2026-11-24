import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AsyncBoundary } from "../../../library/components/AsyncBoundary/versions/1.0.12/AsyncBoundary";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("AsyncBoundary state copy and presentation", () => {
  it("keeps inline stale content and accepts localized copy without adding a frame", () => {
    renderWithProviders(
      <AsyncBoundary
        status="partial-error"
        presentation="inline"
        stateTitle="Refresh failed"
        stateDetail="Showing saved data: request denied."
      >
        <p>last good response</p>
      </AsyncBoundary>,
    );
    const boundary = document.querySelector("[data-rcl-async-boundary]");
    expect(boundary).not.toBeNull();
    expect(boundary).toHaveAttribute("data-rcl-async-status", "partial-error");
    expect(boundary).toHaveAttribute("data-presentation", "inline");
    expect(boundary).toHaveTextContent("Refresh failed");
    expect(boundary).toHaveTextContent("Showing saved data: request denied.");
    expect(boundary).toHaveTextContent("last good response");
  });

  it("uses product-local loading copy while retaining the shared loading state", async () => {
    renderWithProviders(
      <AsyncBoundary
        status="pending"
        loadingDelay={0}
        loadingLabel="Reading files"
        loadingDetail="A fresh read takes a moment."
      />,
    );
    const boundary = document.querySelector("[data-rcl-async-boundary]");
    expect(boundary).not.toBeNull();
    expect(boundary).toHaveAttribute("data-rcl-async-status", "pending");
    await waitFor(() => expect(boundary).toHaveTextContent("Reading files"));
    expect(boundary).toHaveTextContent("A fresh read takes a moment.");
  });
});
