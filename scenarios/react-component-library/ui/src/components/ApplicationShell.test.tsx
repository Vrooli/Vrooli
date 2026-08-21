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

import { ApplicationShell as AppShell } from "./ui/ApplicationShell";

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

  it("keeps the skip link in the viewport geometry and targets the main region", () => {
    const { container } = renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>page</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );

    const skipLink = container.querySelector<HTMLAnchorElement>(".rcl-app-shell-skip");
    expect(skipLink).toHaveAttribute("href", "#app-shell-main");
    expect(screen.getByRole("main")).toHaveAttribute("id", "app-shell-main");
    // The stylesheet is injected once into <head> by useComponentStyles rather
    // than rendered inline per instance; the rule it must declare is unchanged.
    const shellStyles = document.head.querySelector('style[data-rcl-style-id="rcl-app-shell"]');
    expect(shellStyles).not.toBeNull();
    expect(shellStyles?.textContent).toContain("clip-path: inset(50%)");
    expect(skipLink?.className).toContain("rcl-app-shell-skip");
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
        <Route element={<AppShell />}>
          <Route path="/" element={<div>page</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    await user.click(screen.getByTestId("sidebar-collapse"));
    expect(screen.getByTestId("workspace-header-open-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-shell")).toHaveAttribute("data-mode", "overlay");
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

    const shell = screen.getByTestId("app-shell");
    const main = screen.getByRole("main");
    expect(shell).toHaveAttribute("data-main-mode", "flush");
    expect(main.className).toContain("flex-col");
    expect(main.className).toContain("w-full");
  });

  it("keeps catalog content full-width inside the vertical workspace column", () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>catalog</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );

    const main = screen.getByRole("main");
    expect(main.className).toContain("w-full");
    expect(main.classList.contains("w-0")).toBe(false);
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
    expect(shell).toHaveAttribute("data-mode", "responsive");
    expect(shell).toHaveAttribute("data-open", "true");
    expect(screen.getByTestId("sidebar-shell-backdrop")).toBeInTheDocument();
  });

  it("opens the shared main-actions menu and exposes all three guided actions", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>page</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    await user.click(screen.getByRole("button", { name: "launcher.open" }));
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "launcher.extract" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "launcher.adopt" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "launcher.create" })).toBeInTheDocument();
  });

  it("opens the requested action from a PWA deep link", () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>page</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/?action=adopt&assetId=cmp-1&targetScenario=demo"] },
    );
    expect(screen.getByRole("dialog", { name: "launcher.adopt" })).toBeInTheDocument();
    expect(screen.getByDisplayValue("cmp-1")).toBeInTheDocument();
    expect(screen.getByDisplayValue("demo")).toBeInTheDocument();
  });

  it("routes the coverage operator view through the shared workspace header", () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/coverage" element={<div>coverage</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/coverage"] },
    );
    expect(screen.getByRole("heading", { name: "Catalog coverage" })).toBeInTheDocument();
    expect(screen.getByText("Maturity distribution and ranked next work")).toBeInTheDocument();
  });

  it("routes the capabilities operator view through the shared workspace header", () => {
    cleanup();
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/capabilities" element={<div>capabilities</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/capabilities"] },
    );
    expect(screen.getByRole("heading", { name: "Capability readiness" })).toBeInTheDocument();
    expect(screen.getByText("Integration readiness and recovery guidance")).toBeInTheDocument();
  });

  it("submits catalog search and opens create from the workspace header", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>page</div>} />
          <Route path="/catalog" element={<div>catalog</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    const search = document.querySelector("form input") as HTMLInputElement;
    expect(search).toBeInTheDocument();
    await user.type(search, "buttons");
    await user.keyboard("{Enter}");
    expect(screen.getByText("catalog")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "dashboard.create" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});
