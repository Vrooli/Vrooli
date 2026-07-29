import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { AppHeader } from "../../src/components/layout/AppHeader.js";
import { MobileNav } from "../../src/components/layout/MobileNav.js";
import { HealthStatus } from "../../src/types.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("AppHeader exposes health, navigation, quick run, settings, and keyboard status controls", async () => {
  const user = userEvent.setup();
  const onSectionChange = vi.fn();
  const onStatusClick = vi.fn();
  const onSettingsClick = vi.fn();
  const onQuickRunClick = vi.fn();
  renderWithProviders(createElement(AppHeader, {
    health: { status: HealthStatus.HEALTHY } as any,
    wsStatus: "connected", activeSection: "runs", isMobile: false,
    onSectionChange, onStatusClick, onSettingsClick, onQuickRunClick,
  }));
  assert.ok(screen.getByText("Agent Manager"));
  assert.equal(screen.getByRole("button", { name: "Runs" }).getAttribute("aria-current"), "page");
  await user.click(screen.getByRole("button", { name: "Profiles" }));
  await user.click(screen.getByRole("button", { name: "Quick Run" }));
  await user.click(screen.getByRole("button", { name: "Settings" }));
  await user.keyboard("{Enter}");
  await user.click(screen.getByRole("button", { name: "Open status details" }));
  assert.deepEqual(onSectionChange.mock.calls, [["profiles"]]);
  assert.equal(onQuickRunClick.mock.calls.length, 1);
  assert.equal(onSettingsClick.mock.calls.length, 2);
  assert.equal(onStatusClick.mock.calls.length, 1);
});

test("MobileNav changes sections and marks the active section", async () => {
  const user = userEvent.setup();
  const onSectionChange = vi.fn();
  renderWithProviders(createElement(MobileNav, { activeSection: "health", onSectionChange }));
  assert.equal(screen.getByTestId("mobile-nav-health").getAttribute("aria-current"), "page");
  await user.click(screen.getByRole("button", { name: "Flows" }));
  assert.deepEqual(onSectionChange.mock.calls, [["workflows"]]);
});
