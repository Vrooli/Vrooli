import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { AsyncBoundary } from "./AsyncBoundary";

describe("AsyncBoundary", () => {
  it("renders ready and pending content, including the delayed skeleton", async () => {
    const { rerender } = renderWithProviders(
      <AsyncBoundary status="idle">
        <span>Ready content</span>
      </AsyncBoundary>,
    );
    expect(screen.getByText("Ready content")).toBeInTheDocument();

    rerender(
      <AsyncBoundary status="pending" pending={<span>Custom pending</span>} detectOffline={false} />,
    );
    expect(screen.getByText("Custom pending")).toBeInTheDocument();

    rerender(<AsyncBoundary status="pending" loadingDelay={0} detectOffline={false} />);
    await waitFor(() =>
      expect(document.querySelector("[data-rcl-async-skeleton]")).toBeInTheDocument(),
    );
  });

  it.each(["refreshing", "stale", "partial-error", "offline"] as const)(
    "preserves useful content while %s",
    (status) => {
      renderWithProviders(
        <AsyncBoundary status={status} offline={status === "offline"} detectOffline={false}>
          <span>Saved content</span>
        </AsyncBoundary>,
      );
      expect(screen.getByText("Saved content")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();
    },
  );

  it("runs retry actions for preserving and error states", async () => {
    const user = userEvent.setup();
    const retry = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderWithProviders(
      <AsyncBoundary status="offline" offline detectOffline={false} retry={retry}>
        <span>Saved content</span>
      </AsyncBoundary>,
    );
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledOnce();

    rerender(
      <AsyncBoundary
        status="error"
        detectOffline={false}
        errorTitle="Unavailable"
        error="Source is down"
        retry={retry}
      />,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Source is down");
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledTimes(2);
  });
});
