import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders, makeHealthResponse } from "../test-utils";

vi.mock("../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/components")>();
  return {
    ...actual,
    componentsClient: {
      listComponents: vi.fn().mockResolvedValue({ components: [] }),
      getComponent: vi.fn(),
      getComponentByLibraryId: vi.fn(),
      indexComponents: vi.fn(),
      getComponentContent: vi.fn(),
      updateComponentContent: vi.fn(),
    },
  };
});
vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn() };
});

import { AppShell } from "./AppShell";

describe("AppShell", () => {
  beforeEach(async () => {
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders shell, catalog drawer access, and the child route content", () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div data-testid="child">hello</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    expect(screen.getByTestId("app-shell")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-shell")).toBeInTheDocument();
    expect(screen.getByTestId("app-sidebar-content")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-header")).toBeInTheDocument();
    expect(screen.queryByTestId("mobile-nav")).not.toBeInTheDocument();
    expect(screen.getByTestId("child")).toBeInTheDocument();
    expect(screen.queryByTestId("active-work-menu")).not.toBeInTheDocument();
  });

  it("does not wrap content in a centered card or eyebrow text", () => {
    const { container } = renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>page</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    expect(container.querySelector(".max-w-xl")).toBeNull();
    const shell = screen.getByTestId("app-shell");
    expect(shell.className).toContain("w-full");
  });

  it("collapses desktop navigation and exposes the header hamburger to restore it", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}><Route path="/" element={<div>page</div>} /></Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    await user.click(screen.getByTestId("sidebar-collapse"));
    expect(screen.getByTestId("workspace-header-open-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-shell").className).toContain("md:!hidden");
    await user.click(screen.getByTestId("workspace-header-open-sidebar"));
    expect(screen.queryByTestId("workspace-header-open-sidebar")).not.toBeInTheDocument();
  });

  it("uses the full-bleed main layout for component detail routes", () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/assets/:id" element={<div>detail</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/assets/cmp-1"] },
    );

    const main = screen.getByTestId("app-main");
    expect(main.className).toContain("p-0");
    expect(main.className).toContain("flex-col");
    expect(main.className).toContain("w-full");
  });

  it("renders the mobile sidebar branch when the media query matches", () => {
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: true,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>mobile</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );

    expect(screen.getByTestId("sidebar-shell")).toBeInTheDocument();
  });

  it("opens a full-width safe-area sidebar shell from the workspace header", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div data-testid="child">hello</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );

    await user.click(screen.getByTestId("workspace-header-open-sidebar"));

    const shell = screen.getByTestId("sidebar-shell");
    expect(shell).toHaveAttribute("role", "dialog");
    expect(shell.className).toContain("w-full");
    expect(shell.className).toContain("pt-safe");
    expect(shell.className).toContain("pb-safe");
    expect(screen.getByTestId("sidebar-shell-backdrop")).toBeInTheDocument();
  });

  it("opens the shared main-actions menu and exposes all three guided actions", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Routes><Route element={<AppShell />}><Route path="/" element={<div>page</div>} /></Route></Routes>, { routerEntries: ["/"] });
    await user.click(screen.getByRole("button", { name: "launcher.open" }));
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "launcher.extract" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "launcher.adopt" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "launcher.create" })).toBeInTheDocument();
  });

  it("opens the requested action from a PWA deep link", () => {
    renderWithProviders(<Routes><Route element={<AppShell />}><Route path="/" element={<div>page</div>} /></Route></Routes>, { routerEntries: ["/?action=adopt&assetId=cmp-1&targetScenario=demo"] });
    expect(screen.getByRole("dialog", { name: "launcher.adopt" })).toBeInTheDocument();
    expect(screen.getByDisplayValue("cmp-1")).toBeInTheDocument();
    expect(screen.getByDisplayValue("demo")).toBeInTheDocument();
  });
});
