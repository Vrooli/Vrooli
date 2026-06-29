import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { TopBar } from "./TopBar";

vi.mock("../../api/health", () => ({
  fetchHealth: vi.fn(),
}));

vi.mock("../../api/healthStatus", () => ({
  getProviderHealth: vi.fn(),
}));

import { fetchHealth } from "../../api/health";
import { getProviderHealth } from "../../api/healthStatus";

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

beforeEach(() => {
  vi.mocked(fetchHealth).mockImplementation(() => new Promise(() => {}));
  vi.mocked(getProviderHealth).mockImplementation(() => new Promise(() => {}));
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderTopBar(props: Partial<Parameters<typeof TopBar>[0]> = {}) {
  return renderWithProviders(
    <MemoryRouter future={routerFuture}>
      <TopBar onOpenSettings={vi.fn()} {...props} />
    </MemoryRouter>,
  );
}

describe("TopBar", () => {
  it("renders the app title", () => {
    renderTopBar();
    expect(screen.getByText(strings.app.title)).toBeInTheDocument();
  });

  it("renders the app eyebrow", () => {
    renderTopBar();
    expect(screen.getByText(strings.app.eyebrow)).toBeInTheDocument();
  });

  it("renders the settings button", () => {
    renderTopBar();
    expect(screen.getByRole("button", { name: strings.shell.openSettings })).toBeInTheDocument();
  });

  it("calls onOpenSettings when settings button is clicked", async () => {
    const user = userEvent.setup();
    const onOpenSettings = vi.fn();
    renderTopBar({ onOpenSettings });
    await user.click(screen.getByRole("button", { name: strings.shell.openSettings }));
    expect(onOpenSettings).toHaveBeenCalledTimes(1);
  });

  it("does not render mobile menu button by default", () => {
    renderTopBar();
    expect(screen.queryByRole("button", { name: strings.shell.openMenu })).not.toBeInTheDocument();
  });

  it("renders mobile menu button when showMobileMenuButton=true", () => {
    renderTopBar({ showMobileMenuButton: true, onToggleMobileMenu: vi.fn() });
    expect(screen.getByRole("button", { name: strings.shell.openMenu })).toBeInTheDocument();
  });

  it("calls onToggleMobileMenu when mobile menu button is clicked", async () => {
    const user = userEvent.setup();
    const onToggleMobileMenu = vi.fn();
    renderTopBar({ showMobileMenuButton: true, onToggleMobileMenu });
    await user.click(screen.getByRole("button", { name: strings.shell.openMenu }));
    expect(onToggleMobileMenu).toHaveBeenCalledTimes(1);
  });

  it("renders the header element", () => {
    renderTopBar();
    expect(screen.getByRole("banner")).toBeInTheDocument();
  });
});
