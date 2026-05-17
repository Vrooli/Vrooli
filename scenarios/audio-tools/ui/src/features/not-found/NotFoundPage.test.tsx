import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { NotFoundPage } from "./NotFoundPage";

afterEach(() => {
  cleanup();
});

describe("NotFoundPage", () => {
  it("renders the i18n title and description", () => {
    renderWithProviders(
      <MemoryRouter future={routerFuture}>
        <NotFoundPage />
      </MemoryRouter>,
    );
    expect(screen.getByText(strings.notFound.title)).toBeInTheDocument();
    expect(screen.getByText(strings.notFound.description)).toBeInTheDocument();
  });

  it("renders the back-to-home link pointing at /", () => {
    renderWithProviders(
      <MemoryRouter future={routerFuture}>
        <NotFoundPage />
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: strings.notFound.backToOverview });
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute("href", "/");
  });
});
