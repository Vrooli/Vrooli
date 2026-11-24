import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import { TestAppRouter } from "../app/routes";
import { strings } from "../consts/strings";

vi.mock("../features/health/HealthCard", () => ({ HealthCard: () => <div data-testid="health-card" /> }));
vi.mock("../features/health/ProviderMaturityCard", () => ({ ProviderMaturityCard: () => <div data-testid="provider-maturity-card" /> }));

describe("DashboardPage journey links", () => {
  afterEach(() => cleanup());

  it("offers direct entry to define a brand, generate assets, and apply them", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });

    expect(screen.getByRole("link", { name: strings.pages.brands.title })).toHaveAttribute("href", "/brands");
    expect(screen.getByRole("link", { name: strings.pages.generation.title })).toHaveAttribute("href", "/generation");
    expect(screen.getByRole("link", { name: strings.pages.apply.title })).toHaveAttribute("href", "/apply");
    expect(screen.getByTestId("health-card")).toBeInTheDocument();
    expect(screen.getByTestId("provider-maturity-card")).toBeInTheDocument();
  });
});
