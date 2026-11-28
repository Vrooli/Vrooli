import { renderWithProviders } from "../test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryRouter, RouterProvider } from "react-router-dom";

import { screen } from "@testing-library/react";

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
    renderWithProviders(
      <RouterProvider router={router} future={{ v7_startTransition: true }} />,
      { withoutRouter: true },
    );
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("button")).toBeInTheDocument(); // retry
    expect(screen.getByRole("link")).toBeInTheDocument(); // home
  });
});
