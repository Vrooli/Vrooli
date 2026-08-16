import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("./app/routes", () => ({
  AppRouter: () => <div data-testid="mock-app-router">router</div>,
}));

import App from "./App";
import { renderWithProviders } from "./test-utils";

describe("App composition", () => {
  it("mounts the production provider and router composition", () => {
    renderWithProviders(<App />, { withoutRouter: true });
    expect(screen.getByTestId("mock-app-router")).toBeVisible();
  });
});
