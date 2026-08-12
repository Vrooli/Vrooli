import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { WorkbenchPage } from "./WorkbenchPage";

describe("WorkbenchPage", () => {
  afterEach(() => cleanup());

  it("renders the catalog, safe-state announcement, and empty release state", () => {
    renderWithProviders(<WorkbenchPage />, { routerEntries: ["/catalog"] });
    expect(screen.getByTestId(selectors.pages.workbench)).toBeInTheDocument();
    expect(screen.getByTestId("backdrop-style-catalog")).toBeInTheDocument();
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.getByRole("link")).toBeInTheDocument();
    expect(screen.getAllByRole("img")).toHaveLength(8);
    expect(screen.getByTestId("backdrop-contact-sheet")).toBeInTheDocument();
    expect(screen.getByTestId("backdrop-placement-matrix")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: /pages\.workbench\.releaseAction/i })).toHaveLength(2);
    expect(screen.getAllByRole("button", { name: /pages\.workbench\.releaseAction/i }).every((button) => button instanceof HTMLButtonElement && button.disabled)).toBe(true);
    const copyButton = screen.getAllByRole("button", { name: /pages\.workbench\.copyPlan/i })[0];
    if (copyButton) fireEvent.click(copyButton);
    expect(screen.getAllByRole("button", { name: /pages\.workbench\.planCopied/i })).toHaveLength(2);
  });

  it.each(["loading", "error", "empty"])("renders the %s route state", (state) => {
    renderWithProviders(<WorkbenchPage />, { routerEntries: [`/catalog?state=${state}`] });
    expect(screen.getByTestId(selectors.pages.workbench)).toHaveAttribute("data-workbench-state", state);
    if (state === "error") expect(screen.getByRole("alert")).toBeInTheDocument();
    if (state === "empty") expect(screen.getByText("pages.workbench.emptyState")).toBeInTheDocument();
    if (state === "loading") expect(screen.getAllByRole("status")).toHaveLength(2);
  });
});
