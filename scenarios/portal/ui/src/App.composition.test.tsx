import { screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "./test-utils";

vi.mock("./app/providers", () => ({
  Providers: ({ children }: { children: ReactNode }) => (
    <div data-testid="mock-providers">{children}</div>
  ),
}));

vi.mock("./app/routes", () => ({
  AppRouter: () => <div data-testid="mock-router" />,
}));

import App from "./App";

describe("App", () => {
  it("composes providers around the production router", () => {
    renderWithProviders(<App />);

    expect(screen.getByTestId("mock-providers")).toContainElement(screen.getByTestId("mock-router"));
  });
});
