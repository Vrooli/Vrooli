import { cleanup, fireEvent, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders, expectNoA11yViolations } from "../../test-utils";
import { TabNav } from "./TabNav";

describe("TabNav accessibility", () => {
  afterEach(cleanup);

  it("renders keyboard-accessible navigation without axe violations", async () => {
    const { container } = renderWithProviders(
      <TabNav
        tabs={[{ id: "dashboard", label: "Dashboard" }, { id: "resources", label: "Resources" }]}
        activeTab="dashboard"
        onChange={vi.fn()}
      />
    );

    await expectNoA11yViolations(container);
    expect(container.querySelector("button")).toBeInTheDocument();
  });

  it("routes button and link navigation while showing only positive badges", () => {
    const onChange = vi.fn();
    const { container, rerender } = renderWithProviders(
      <TabNav
        tabs={[{ id: "dashboard", label: "Dashboard", badgeCount: 0 }, { id: "resources", label: "Resources", badgeCount: 2 }]}
        activeTab="dashboard"
        onChange={onChange}
      />
    );

    expect(screen.getByText("2")).toBeInTheDocument();
    expect(container.querySelector("button")?.className).toContain("border-emerald-400");
    fireEvent.click(screen.getByRole("button", { name: /Resources/ }));
    expect(onChange).toHaveBeenCalledWith("resources");

    rerender(
      <MemoryRouter>
        <TabNav
          tabs={[{ id: "dashboard", label: "Dashboard" }, { id: "resources", label: "Resources" }]}
          activeTab="resources"
          basePath="secrets-manager///"
          onChange={onChange}
        />
      </MemoryRouter>
    );
    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("link", { name: "Resources" })).toHaveAttribute("href", "/secrets-manager/resources");
    fireEvent.click(screen.getByRole("link", { name: "Resources" }));
    expect(onChange).toHaveBeenLastCalledWith("resources");
  });
});
