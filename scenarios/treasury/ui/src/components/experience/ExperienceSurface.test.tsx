import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { ExperienceSurface } from "./ExperienceSurface";

describe("ExperienceSurface", () => {
  afterEach(cleanup);

  it("marks loading surfaces busy and announces their status", () => {
    renderWithProviders(
      <ExperienceSurface surfaceId="approvals" state="loading" statusMessage="Loading approvals">
        <span>Queue</span>
      </ExperienceSurface>,
    );

    const surface = screen.getByText("Queue").closest("section");
    expect(surface).toHaveAttribute("data-experience-surface", "approvals");
    expect(surface).toHaveAttribute("data-experience-state", "loading");
    expect(surface).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("status")).toHaveTextContent("Loading approvals");
  });

  it.each(["ready", "empty", "static"] as const)(
    "does not expose a live status for the %s state",
    (state) => {
      renderWithProviders(
        <ExperienceSurface surfaceId="approvals" state={state} statusMessage="Not announced">
          Queue
        </ExperienceSurface>,
      );

      expect(screen.queryByRole("status")).not.toBeInTheDocument();
      expect(screen.getByText("Queue").closest("section")).not.toHaveAttribute("aria-busy");
    },
  );

  it.each(["partial", "error"] as const)("announces the %s state without marking it busy", (state) => {
    renderWithProviders(
      <ExperienceSurface surfaceId="approvals" state={state} statusMessage={`${state} result`}>
        Queue
      </ExperienceSurface>,
    );

    expect(screen.getByRole("status")).toHaveTextContent(`${state} result`);
    expect(screen.getByText("Queue").closest("section")).not.toHaveAttribute("aria-busy");
  });
});
