// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

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
    render(<App />);

    expect(screen.getByTestId("mock-providers")).toContainElement(screen.getByTestId("mock-router"));
  });
});
