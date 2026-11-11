import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { ExperienceSurface } from "./ExperienceSurface";
import { renderWithProviders } from "../../test-utils";

describe("ExperienceSurface", () => {
  afterEach(() => cleanup());

  it("exposes a loading state and politely announces its status", () => {
    renderWithProviders(
      <ExperienceSurface surfaceId="controller-list" state="loading" statusMessage="Loading controllers">
        <p>Controller list</p>
      </ExperienceSurface>,
    );

    const surface = screen.getByTestId("experience-surface-controller-list");
    expect(surface).toHaveAttribute("data-experience-surface", "controller-list");
    expect(surface).toHaveAttribute("data-experience-state", "loading");
    expect(surface).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("status", { name: "Loading controllers" })).toBeInTheDocument();
    expect(screen.getByText("Controller list")).toBeInTheDocument();
  });

  it("keeps ready content available without announcing a stale status", () => {
    renderWithProviders(
      <ExperienceSurface surfaceId="controller-list" state="ready" statusMessage="Loading controllers" as="main">
        <h1>Controllers connected</h1>
      </ExperienceSurface>,
    );

    expect(screen.getByRole("main")).toHaveAttribute("data-experience-state", "ready");
    expect(screen.getByRole("heading", { name: "Controllers connected" })).toBeInTheDocument();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
