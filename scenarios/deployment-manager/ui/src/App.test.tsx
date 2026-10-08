import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import * as api from "./lib/api";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { getProxyInfo } from "@vrooli/api-base";

afterEach(() => cleanup());

vi.mock("@vrooli/api-base", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@vrooli/api-base")>()),
  getProxyInfo: vi.fn(),
}));

vi.mock("./lib/api");

describe("App routing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getProxyInfo).mockReturnValue(null);
    vi.mocked(api.listProfiles).mockResolvedValue([]);
    window.history.pushState({}, "", "/evidence");
  });

  it("routes the evidence surface inside the shared shell", async () => {
    renderWithProviders(<App />, { withoutRouter: true });
    expect(await screen.findByText("Evidence review")).toBeInTheDocument();
    expect(screen.getAllByText("Deployment Manager").length).toBeGreaterThan(0);
  });

  it("normalizes a proxied router basename", async () => {
    vi.mocked(getProxyInfo).mockReturnValue({ primary: { path: "/apps/deployment-manager///" } } as never);
    window.history.pushState({}, "", "/apps/deployment-manager/evidence");
    renderWithProviders(<App />, { withoutRouter: true });
    expect(await screen.findByText("Evidence review")).toBeInTheDocument();
  });

  it("uses the proxy base path when no primary path exists", async () => {
    vi.mocked(getProxyInfo).mockReturnValue({ basePath: "/proxy///" } as never);
    window.history.pushState({}, "", "/proxy/evidence");
    renderWithProviders(<App />, { withoutRouter: true });
    expect(await screen.findByText("Evidence review")).toBeInTheDocument();
  });
});
