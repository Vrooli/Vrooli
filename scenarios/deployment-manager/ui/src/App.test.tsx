import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import * as api from "./lib/api";
import { renderWithProviders } from "@vrooli/api-base/testing";

vi.mock("./lib/api");

describe("App routing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.listProfiles).mockResolvedValue([]);
    window.history.pushState({}, "", "/evidence");
  });

  it("routes the evidence surface inside the shared shell", async () => {
    renderWithProviders(<App />, { withoutRouter: true });
    expect(await screen.findByText("Evidence review")).toBeInTheDocument();
    expect(screen.getByText("Deployment Manager")).toBeInTheDocument();
  });
});
