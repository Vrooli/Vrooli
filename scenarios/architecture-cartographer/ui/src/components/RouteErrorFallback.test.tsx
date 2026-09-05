import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryRouter, RouterProvider } from "react-router-dom";

import { render, screen } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";

import { i18n } from "../i18n";
import { RouteErrorFallback } from "./RouteErrorFallback";

const buildRouter = () =>
  createMemoryRouter(
    [
      {
        path: "/",
        errorElement: <RouteErrorFallback />,
        element: <Thrower />,
      },
    ],
    { initialEntries: ["/"] },
  );

function Thrower(): React.ReactElement {
  throw new Error("boom");
}

describe("RouteErrorFallback", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("renders the error UI with retry + home actions", () => {
    const router = buildRouter();
    render(
      <I18nextProvider i18n={i18n}>
        <RouterProvider router={router} future={{ v7_startTransition: true }} />
      </I18nextProvider>,
    );
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("button")).toBeInTheDocument(); // retry
    expect(screen.getByRole("link")).toBeInTheDocument(); // home
  });
});
