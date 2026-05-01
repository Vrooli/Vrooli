import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { GraphActionLauncher } from "./GraphActionLauncher";

describe("GraphActionLauncher", () => {
  it("opens a compact menu and dispatches launcher actions", () => {
    const onQuickCapture = vi.fn();
    const onPlanWork = vi.fn();
    const onAuthorOperatingMode = vi.fn();

    render(
      <GraphActionLauncher
        onQuickCapture={onQuickCapture}
        onPlanWork={onPlanWork}
        onAuthorOperatingMode={onAuthorOperatingMode}
      />,
    );

    fireEvent.click(screen.getByTestId("graph-action-fab"));

    expect(screen.getByRole("menu")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("menuitem", { name: "Quick Capture" }));
    expect(onQuickCapture).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("menu")).toBeNull();

    fireEvent.click(screen.getByTestId("graph-action-fab"));
    fireEvent.click(screen.getByRole("menuitem", { name: "Plan Work With Agent" }));
    expect(onPlanWork).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("graph-action-fab"));
    fireEvent.click(screen.getByRole("menuitem", { name: "Author Operating Mode" }));
    expect(onAuthorOperatingMode).toHaveBeenCalledTimes(1);
  });

  it("disables session actions while busy and shows launcher errors", () => {
    render(
      <GraphActionLauncher
        isBusy
        error="Unable to start session."
        onQuickCapture={vi.fn()}
        onPlanWork={vi.fn()}
        onAuthorOperatingMode={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByTestId("graph-action-fab"));

    expect(screen.getByRole("menuitem", { name: "Quick Capture" })).toBeEnabled();
    expect(screen.getByRole("menuitem", { name: "Plan Work With Agent" })).toBeDisabled();
    expect(screen.getByRole("menuitem", { name: "Author Operating Mode" })).toBeDisabled();
    expect(screen.getByRole("alert")).toHaveTextContent("Unable to start session.");
  });

  it("closes on Escape", () => {
    render(
      <GraphActionLauncher
        onQuickCapture={vi.fn()}
        onPlanWork={vi.fn()}
        onAuthorOperatingMode={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByTestId("graph-action-fab"));
    expect(screen.getByRole("menu")).toBeInTheDocument();

    fireEvent.keyDown(window, { key: "Escape" });
    expect(screen.queryByRole("menu")).toBeNull();
  });
});
