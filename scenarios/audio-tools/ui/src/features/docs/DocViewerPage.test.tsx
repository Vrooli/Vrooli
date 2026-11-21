import { describe, it, expect, vi, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

vi.mock("./docsContent", () => ({
  getDocContent: (path: string) =>
    path === "PRD.md" ? "# PRD heading\n\nbody content" : null,
  listDocPaths: () => ["PRD.md"],
}));

import { DocViewerPage } from "./DocViewerPage";

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

afterEach(() => {
  cleanup();
});

function renderAt(path: string) {
  return renderWithProviders(
    <MemoryRouter future={routerFuture} initialEntries={[`/docs/${path}`]}>
      <Routes>
        <Route path="/docs/*" element={<DocViewerPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("DocViewerPage", () => {
  it("renders the markdown content for a known doc path", () => {
    renderAt("PRD.md");
    expect(screen.getByRole("heading", { name: "PRD heading", level: 1 })).toBeInTheDocument();
    expect(screen.getByText(/^body content$/)).toBeInTheDocument();
  });

  it("renders the not-found copy for an unknown doc path", () => {
    renderAt("nope.md");
    expect(screen.getByText(strings.docs.notFoundTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.docs.notFoundDescription)).toBeInTheDocument();
  });

  it("renders a back-to-list link pointing at /docs", () => {
    renderAt("PRD.md");
    const back = screen.getByRole("link", { name: strings.docs.backToList });
    expect(back).toHaveAttribute("href", "/docs");
  });
});
