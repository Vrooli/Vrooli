import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import { LaunchFooter } from "./LaunchFooter";

describe("LaunchFooter", () => {
  it("uses the shared overlay action bar and preserves launch behavior", () => {
    const onLaunch = vi.fn();
    render(
      <LaunchFooter
        summary="Codex · This machine"
        label="Launch selected Codex"
        buttonLabel="Launch"
        disabled={false}
        busy={false}
        busyLabel="Creating…"
        onLaunch={onLaunch}
        testId="launch-action"
      />,
    );

    const button = screen.getByTestId("launch-action");
    expect(button).toHaveAttribute("aria-label", "Launch");
    expect(button).toHaveAttribute("data-primary", "true");
    expect(screen.getByRole("group", { name: "Launch selected Codex" })).toBeInTheDocument();
    expect(screen.getByTestId("launcher-summary")).toHaveTextContent("Codex · This machine");
    expect(screen.getByTestId("launcher-footer")).toContainElement(
      screen.getByRole("group", { name: "Launch selected Codex" }),
    );

    fireEvent.click(button);
    expect(onLaunch).toHaveBeenCalledTimes(1);
  });

  it("keeps the disabled reason and busy state visible while disabling the action", () => {
    const onLaunch = vi.fn();
    const { rerender } = render(
      <LaunchFooter
        summary="Codex"
        label="Launch"
        buttonLabel="Launch"
        disabled
        busy={false}
        busyLabel="Creating…"
        reason="The selected model is unavailable."
        onLaunch={onLaunch}
        testId="launch-action"
      />,
    );

    expect(screen.getByTestId("launcher-disabled-reason")).toHaveTextContent(
      "The selected model is unavailable.",
    );
    expect(screen.getByTestId("launcher-disabled-reason")).toHaveAttribute("role", "status");
    expect(screen.getByTestId("launch-action")).toBeDisabled();

    rerender(
      <LaunchFooter
        summary="Codex"
        label="Launch"
        buttonLabel="Launch"
        disabled={false}
        busy
        busyLabel="Creating…"
        onLaunch={onLaunch}
        testId="launch-action"
      />,
    );
    expect(screen.getByTestId("launch-action")).toHaveTextContent("Creating…");
    expect(screen.getByTestId("launch-action")).toBeDisabled();
    expect(screen.queryByTestId("launcher-disabled-reason")).toBeNull();
    expect(onLaunch).not.toHaveBeenCalled();
  });
});
