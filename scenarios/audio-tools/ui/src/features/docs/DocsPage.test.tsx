import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { DocsPage } from "./DocsPage";

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

afterEach(() => {
  cleanup();
});

describe("DocsPage", () => {
  it("renders the page title and the four group headings", () => {
    renderWithProviders(
      <MemoryRouter future={routerFuture}>
        <DocsPage />
      </MemoryRouter>,
    );
    expect(screen.getByText(strings.docs.title)).toBeInTheDocument();
    expect(screen.getByText(strings.docs.groupPRD)).toBeInTheDocument();
    expect(screen.getByText(strings.docs.groupConcepts)).toBeInTheDocument();
    expect(screen.getByText(strings.docs.groupReference)).toBeInTheDocument();
    expect(screen.getByText(strings.docs.groupInternal)).toBeInTheDocument();
  });

  it("renders nav links to /docs/<path> for each doc entry", () => {
    renderWithProviders(
      <MemoryRouter future={routerFuture}>
        <DocsPage />
      </MemoryRouter>,
    );
    const prd = screen.getByRole("link", { name: /PRD/ });
    expect(prd).toHaveAttribute("href", "/docs/PRD.md");
    const arch = screen.getByRole("link", { name: /Architecture/ });
    expect(arch).toHaveAttribute("href", "/docs/docs/concepts/ARCHITECTURE.md");
  });
});
