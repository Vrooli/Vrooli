import { describe, it, expect, vi, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

vi.mock("./docsContent", () => ({
  getDocContent: (path: string) => {
    if (path === "ERR.md") return Promise.reject(new Error("chunk unavailable"));
    return Promise.resolve(path === "PRD.md" ? "# PRD heading\n\nbody content" : null);
  },
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
  it("loads and renders only the selected markdown document", async () => {
    renderAt("PRD.md");
    expect(await screen.findByRole("heading", { name: "PRD heading", level: 1 })).toBeInTheDocument();
    expect(screen.getByText(/^body content$/)).toBeInTheDocument();
  });

  it("renders the not-found copy for an unknown doc path", async () => {
    renderAt("nope.md");
    expect(await screen.findByText(strings.docs.notFoundTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.docs.notFoundDescription)).toBeInTheDocument();
  });

  it("shows a recoverable message when a selected document chunk fails to load", async () => {
    renderAt("ERR.md");
    expect(await screen.findByRole("alert")).toHaveTextContent(strings.docs.loadError);
  });

  it("renders a back-to-list link pointing at /docs", async () => {
    renderAt("PRD.md");
    await screen.findByRole("heading", { name: "PRD heading", level: 1 });
    const back = screen.getByRole("link", { name: strings.docs.backToList });
    expect(back).toHaveAttribute("href", "/docs");
  });
});
