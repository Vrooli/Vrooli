import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { AppHeader } from "../../src/components/layout/AppHeader.js";
import { SideNav } from "../../src/components/layout/SideNav.js";
import { HealthStatus } from "../../src/types.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("AppHeader exposes health, quick run, settings, and keyboard status controls", async () => {
  const user = userEvent.setup();
  const onSectionChange = vi.fn();
  const onStatusClick = vi.fn();
  const onSettingsClick = vi.fn();
  const onQuickRunClick = vi.fn();
  const onNavigationClick = vi.fn();
  renderWithProviders(createElement(AppHeader, {
    health: { status: HealthStatus.HEALTHY } as any,
    wsStatus: "connected", activeSection: "runs", isMobile: false,
    onSectionChange, onStatusClick, onSettingsClick, onQuickRunClick, onNavigationClick,
  }));
  assert.ok(screen.getByText("Agent Manager"));
  await user.click(screen.getByRole("button", { name: "Quick Run" }));
  await user.click(screen.getByRole("button", { name: "Open navigation menu" }));
  await user.click(screen.getByRole("button", { name: "Settings" }));
  await user.keyboard("{Enter}");
  await user.click(screen.getByRole("button", { name: "Open status details" }));
  assert.deepEqual(onSectionChange.mock.calls, []);
  assert.equal(onQuickRunClick.mock.calls.length, 1);
  assert.equal(onNavigationClick.mock.calls.length, 1);
  assert.equal(onSettingsClick.mock.calls.length, 2);
  assert.equal(onStatusClick.mock.calls.length, 1);
});

test("AppHeader communicates an unavailable connection state", () => {
  renderWithProviders(createElement(AppHeader, {
    health: { status: HealthStatus.UNHEALTHY } as any,
    wsStatus: "error", activeSection: "runs", isMobile: true,
    onSectionChange: vi.fn(), onStatusClick: vi.fn(), onSettingsClick: vi.fn(), onQuickRunClick: vi.fn(), onNavigationClick: vi.fn(),
  }));
  assert.ok(screen.getByLabelText("Open status details").textContent !== null);
});

test("SideNav changes sections and marks the active section", async () => {
	const user = userEvent.setup();
	const onSectionChange = vi.fn();
	renderWithProviders(createElement(SideNav, { activeSection: "health", onSectionChange, onSettingsClick: vi.fn(), mobileOpen: false, onMobileOpenChange: vi.fn() }));
	assert.equal(screen.getByTestId("sidenav-health").getAttribute("aria-current"), "page");
	await user.click(screen.getByTestId("sidenav-workflows"));
  assert.deepEqual(onSectionChange.mock.calls, [["workflows"]]);
});

test("SideNav exposes the mobile drawer labels and settings action", async () => {
	const settings = vi.fn();
	renderWithProviders(createElement(SideNav, { activeSection: "dashboard", onSectionChange: vi.fn(), onSettingsClick: settings, mobileOpen: true, onMobileOpenChange: vi.fn() }));
	assert.equal(screen.getByTestId("mobile-nav-dashboard").getAttribute("aria-current"), "page");
	await userEvent.setup().click(screen.getByRole("button", { name: "Mobile navigation settings" }));
	assert.equal(settings.mock.calls.length, 1);
});
