import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { ExperienceSurface } from "./ExperienceSurface";
import { renderWithProviders } from "../../test-utils";

const tokensLabel = "Tokens";
const readyTokensLabel = "Ready tokens";

describe("ExperienceSurface", () => {
  afterEach(cleanup);

  it("exposes loading state and an assistive status message", () => {
    renderWithProviders(
      <ExperienceSurface surfaceId="tokens" state="loading" statusMessage="Loading token types">
        <span>{tokensLabel}</span>
      </ExperienceSurface>,
    );

    const surface = screen.getByText(tokensLabel).closest("section");
    expect(surface).toHaveAttribute("data-experience-surface", "tokens");
    expect(surface).toHaveAttribute("data-experience-state", "loading");
    expect(surface).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("status")).toHaveTextContent("Loading token types");
  });

  it("keeps ready content quiet when no live update is needed", () => {
    renderWithProviders(
      <ExperienceSurface surfaceId="tokens" state="ready" statusMessage="Ready">
        <span>{readyTokensLabel}</span>
      </ExperienceSurface>,
    );

    expect(screen.getByText(readyTokensLabel).closest("section")).not.toHaveAttribute("aria-busy");
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
