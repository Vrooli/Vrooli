import { describe, expect, it, vi } from "vitest";
import { AppShell } from "./shared/ui/composites/app-shell";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { renderWithProviders } from "./test-utils/renderWithProviders";

describe("AppShell accessibility", () => {
  it("has no axe-core violations", async () => {
    const { container } = renderWithProviders(
      <AppShell
        activeTab="dashboard"
        isTickRunning={false}
        onOpenSettings={vi.fn()}
        onRunTick={vi.fn()}
        onTabChange={vi.fn()}
      >
        <p>Dashboard content</p>
      </AppShell>,
    );

    expect(container.querySelector("main")).toBeTruthy();
    await expectNoA11yViolations(container);
  });
});
