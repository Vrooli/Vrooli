/**
 * Additional coverage for DocViewerPage — branches 14-29, 46-47.
 *
 * Missing coverage:
 *   - Line 15: `path ? getDocContent(path) : null` — the falsy-path branch
 *     (empty `path` produces `content = null` immediately, without calling getDocContent)
 *   - Line 29: `path || t(strings.docs.title)` — Panel title falls back to
 *     the docs title key when `path` is empty
 *   - Lines 46-47: `deriveTitle("")` → "Docs"
 */
import { describe, it, expect, vi, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

vi.mock("./docsContent", () => ({
  getDocContent: (path: string) =>
    path === "PRD.md" ? "# Document heading\n\nbody" : null,
  listDocPaths: () => ["PRD.md"],
}));

import { DocViewerPage } from "./DocViewerPage";

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

afterEach(() => {
  cleanup();
});

function renderAt(path: string) {
  return renderWithProviders(
    <MemoryRouter future={routerFuture} initialEntries={[path]}>
      <Routes>
        <Route path="/docs/*" element={<DocViewerPage />} />
        {/* Route with no wildcard so `params["*"]` is undefined → "" */}
        <Route path="/bare" element={<DocViewerPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("DocViewerPage — missing-path branch (lines 14-29, 46-47)", () => {
  it("renders not-found copy and the docs title when navigated to /docs/ with an empty wildcard", () => {
    // /docs/ with no further path → params["*"] === ""
    renderAt("/docs/");
    // Panel title should be the docs.title key (the `|| t(strings.docs.title)` branch)
    expect(screen.getByText(strings.docs.title)).toBeInTheDocument();
    // content === null so not-found copy appears
    expect(screen.getByText(strings.docs.notFoundTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.docs.notFoundDescription)).toBeInTheDocument();
  });

  it("renders not-found copy for a known-bad path (no entry in docs map)", () => {
    renderAt("/docs/MISSING.md");
    expect(screen.getByText(strings.docs.notFoundTitle)).toBeInTheDocument();
  });

  it("derives the page title from the filename without extension", () => {
    // Use PRD.md which exists in our mock; title should be "PRD" (no .md)
    renderAt("/docs/PRD.md");
    // PageHeader receives title="PRD"; it renders it somewhere in the DOM
    expect(screen.getByText(/^PRD$/)).toBeInTheDocument();
  });

  it("derives the page title as 'Docs' for an empty path (line 46-47)", () => {
    renderAt("/docs/");
    // deriveTitle("") → "Docs"
    expect(screen.getByText(/^Docs$/)).toBeInTheDocument();
  });
});
