import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsTabPrecommit } from "./SettingsTabPrecommit";
import { renderWithQueryClient } from "../test-utils";
import type { PrecommitConfig } from "../lib/api";

const mocks = vi.hoisted(() => ({
  config: undefined as PrecommitConfig | undefined,
  saveMutate: vi.fn(),
  streamRun: vi.fn(),
  streamCancel: vi.fn(),
  streamReset: vi.fn(),
  streamState: { running: false, elapsedMs: 0, tail: [] as string[] },
}));

vi.mock("../lib/hooks-core", () => ({
  usePrecommitConfig: () => ({ data: mocks.config, isLoading: false }),
  useSavePrecommitConfig: () => ({
    mutate: (cfg: PrecommitConfig) => mocks.saveMutate(cfg),
    isPending: false,
  }),
  useStreamPrecommit: () => ({
    state: mocks.streamState,
    run: mocks.streamRun,
    cancel: mocks.streamCancel,
    reset: mocks.streamReset,
  }),
}));

function baseConfig(overrides: Partial<PrecommitConfig> = {}): PrecommitConfig {
  return {
    enabled: false,
    command: "make hygiene",
    working_directory: "/repo",
    timeout_seconds: 300,
    run_before_commit: true,
    allow_override: true,
    ...overrides,
  };
}

describe("SettingsTabPrecommit", () => {
  beforeEach(() => {
    mocks.config = baseConfig();
    mocks.saveMutate = vi.fn();
    mocks.streamRun = vi.fn();
    mocks.streamCancel = vi.fn();
    mocks.streamReset = vi.fn();
    mocks.streamState = { running: false, elapsedMs: 0, tail: [] };
  });

  it("auto-saves immediately when Enabled is toggled", () => {
    renderWithQueryClient(<SettingsTabPrecommit repoId="r1" />);
    const enabledCheckbox = screen.getAllByRole("checkbox")[0];
    if (!enabledCheckbox) throw new Error("expected an Enabled checkbox");
    fireEvent.click(enabledCheckbox);
    expect(mocks.saveMutate).toHaveBeenCalledTimes(1);
    expect((mocks.saveMutate.mock.calls[0] ?? [])[0]).toMatchObject({ enabled: true });
  });

  it("auto-saves immediately when Run before commit is toggled", () => {
    mocks.config = baseConfig({ enabled: true, run_before_commit: true });
    renderWithQueryClient(<SettingsTabPrecommit repoId="r1" />);
    const checkboxes = screen.getAllByRole("checkbox");
    const runBeforeCommitCheckbox = checkboxes[1];
    if (!runBeforeCommitCheckbox) throw new Error("expected a Run-before-commit checkbox");
    fireEvent.click(runBeforeCommitCheckbox);
    expect(mocks.saveMutate).toHaveBeenCalledTimes(1);
    expect((mocks.saveMutate.mock.calls[0] ?? [])[0]).toMatchObject({ run_before_commit: false });
  });

  it("does not auto-save when a text field (command) is edited", () => {
    renderWithQueryClient(<SettingsTabPrecommit repoId="r1" />);
    fireEvent.change(screen.getByLabelText(/Command/i), { target: { value: "pnpm lint" } });
    expect(mocks.saveMutate).not.toHaveBeenCalled();
    expect(screen.getByTestId("precommit-unsaved-indicator")).toBeInTheDocument();
  });

  it("Save button is disabled when text fields match the server", () => {
    renderWithQueryClient(<SettingsTabPrecommit repoId="r1" />);
    expect(screen.getByRole("button", { name: /Save/i })).toBeDisabled();
    expect(screen.queryByTestId("precommit-unsaved-indicator")).not.toBeInTheDocument();
  });

  it("Save button persists text-field edits and clears the unsaved indicator on refetch", () => {
    const { rerender } = renderWithQueryClient(<SettingsTabPrecommit repoId="r1" />);
    fireEvent.change(screen.getByLabelText(/Command/i), { target: { value: "pnpm lint" } });
    const saveBtn = screen.getByRole("button", { name: /Save/i });
    expect(saveBtn).not.toBeDisabled();
    fireEvent.click(saveBtn);
    expect(mocks.saveMutate).toHaveBeenCalledTimes(1);
    expect((mocks.saveMutate.mock.calls[0] ?? [])[0]).toMatchObject({ command: "pnpm lint" });
    // Simulate server confirming the save by returning the new config.
    mocks.config = baseConfig({ command: "pnpm lint" });
    rerender(<SettingsTabPrecommit repoId="r1" />);
    expect(screen.queryByTestId("precommit-unsaved-indicator")).not.toBeInTheDocument();
  });
});
