import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import RunCommandDialog from "./RunCommandDialog";

describe("RunCommandDialog", () => {
  it("runs an edited command in the chosen working directory", async () => {
    const onRun = vi.fn().mockResolvedValue(undefined);
    render(<RunCommandDialog command="echo one" onClose={vi.fn()} onRun={onRun} />);
    fireEvent.change(screen.getByTestId("run-command-input"), { target: { value: "echo two" } });
    fireEvent.change(screen.getByTestId("run-command-working-dir"), { target: { value: "/tmp" } });
    await act(async () => { fireEvent.click(screen.getByTestId("run-command-submit")); await Promise.resolve(); });
    expect(onRun).toHaveBeenCalledWith("echo two", "/tmp");
  });

  it("requires confirmation before a risky command can run", async () => {
    const onRun = vi.fn().mockResolvedValue(undefined);
    render(<RunCommandDialog command="sudo echo one" onClose={vi.fn()} onRun={onRun} />);
    expect(screen.getByTestId("run-command-submit")).toBeDisabled();
    fireEvent.click(screen.getByTestId("run-command-risk-confirm"));
    await act(async () => { fireEvent.click(screen.getByTestId("run-command-submit")); await Promise.resolve(); });
    expect(onRun).toHaveBeenCalledWith("sudo echo one", ".");
  });

  it("does not show the risk acknowledgement for a safe command", () => {
    render(<RunCommandDialog command="echo one" onClose={vi.fn()} onRun={vi.fn()} />);
    expect(screen.queryByTestId("run-command-risk-confirm")).toBeNull();
  });
});
