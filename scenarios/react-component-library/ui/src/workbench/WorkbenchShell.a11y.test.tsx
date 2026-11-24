import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, screen } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { i18n, setLocale } from "../i18n";
import { strings } from "../consts/strings";
import { WorkbenchShell } from "./WorkbenchShell";

vi.mock("./useBrand", () => ({
  useBrand: () => ({ data: null }),
}));

vi.mock("../api/review", () => ({
  getReviewSummary: vi.fn().mockResolvedValue({ open: 0 }),
}));

describe("WorkbenchShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => cleanup());

  it("composes named navigation, a skip target, header utilities, and routed content accessibly", async () => {
    const { container } = renderWithProviders(
      <WorkbenchShell>
        <div data-testid="workbench-content">Workbench content</div>
      </WorkbenchShell>,
      { routerEntries: ["/"] },
    );
    await act(async () => {
      await new Promise((resolve) => globalThis.setTimeout(resolve, 0));
    });

    expect(screen.getByTestId(selectors.app.shell)).toBeInTheDocument();
    expect(screen.getByTestId(`${selectors.app.shell}-skip`)).toHaveAttribute(
      "href",
      `#${selectors.app.main}`,
    );
    expect(
      screen.getByRole("navigation", { name: i18n.t(strings.workbench.nav.label) }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("workbench-shell-tabs")).toHaveAttribute(
      "aria-label",
      i18n.t(strings.workbench.nav.mobile),
    );
    expect(screen.getByRole("button", { name: "Search the workbench" })).toBeInTheDocument();
    expect(screen.getByTestId("workbench-content")).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });

  it("gives the design workspace the library shell's full content area", async () => {
    renderWithProviders(
      <WorkbenchShell>
        <div data-testid="workbench-content">Design workspace</div>
      </WorkbenchShell>,
      { routerEntries: ["/design?workspace=ws_harbor"] },
    );
    await act(async () => {
      await new Promise((resolve) => globalThis.setTimeout(resolve, 0));
    });

    expect(screen.getByTestId(selectors.app.shell)).toHaveAttribute("data-main-mode", "fill");
    expect(screen.getByTestId(selectors.app.main)).toHaveClass("is-workspace-studio");
  });
});
