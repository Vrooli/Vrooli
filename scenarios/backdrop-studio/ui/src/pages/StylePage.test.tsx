import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Route, Routes } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { makeStudioMocks, renderWithProviders } from "../test-utils";
import { StylePage } from "./StylePage";

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

function renderAt(styleId: string) {
  return renderWithProviders(
    <Routes>
      <Route path="/styles/:styleId" element={<StylePage />} />
    </Routes>,
    { routerEntries: [`/styles/${styleId}`] },
  );
}

describe("StylePage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the treatment chain, its parameters, and the perceptual verdict", async () => {
    renderAt("cyanotype-arcade");
    expect(await screen.findByTestId("style-chain")).toBeInTheDocument();
    expect(screen.getByTestId("style-parameters")).toHaveTextContent("lpi");
    // The verdict is shown with its margin, not as a bare pass badge: a style
    // clearing its bar by a hair is one retune from being refused.
    expect(await screen.findByTestId("quality-meters")).toBeInTheDocument();
    expect(screen.getByTestId("quality-metric-subject_survival")).toHaveAttribute(
      "data-clears",
      "true",
    );
  });

  it("renders both mockups with real copy rather than placeholder bars", async () => {
    renderAt("cyanotype-arcade");
    expect(await screen.findByTestId("mockup-landing")).toBeInTheDocument();
    expect(screen.getByTestId("mockup-store")).toBeInTheDocument();
    expect(screen.getByText(strings.pages.style.copyHeadline)).toBeInTheDocument();
  });

  it("announces loading before the catalog arrives", () => {
    renderAt("cyanotype-arcade");
    expect(screen.getByTestId(selectors.pages.style)).toHaveAttribute(
      "data-experience-state",
      "loading",
    );
  });

  it("says so when the id is not in the catalog", async () => {
    renderAt("not-a-style");
    expect(await screen.findByText(strings.pages.style.notFound)).toBeInTheDocument();
  });

  it("reports a failed render without losing the rest of the page", async () => {
    const { submitRender } = await import("../api/studio");
    vi.mocked(submitRender).mockRejectedValue(new Error("no generation capability"));
    renderAt("cyanotype-arcade");
    expect(await screen.findByRole("alert")).toHaveTextContent("pages.style.renderError");
    expect(screen.getByTestId("style-chain")).toBeInTheDocument();
  });
});
