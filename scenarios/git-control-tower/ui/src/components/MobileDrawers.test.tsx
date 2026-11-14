import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { setViewport } from "../test-utils";
import { DiscardConfirmationModal } from "./DiscardConfirmationModal";
import { DeleteConfirmationModal } from "./DeleteConfirmationModal";
import { ScenarioPickerModal } from "./ScenarioPickerModal";
import { StageSameNameConfirmationModal } from "./StageSameNameConfirmationModal";

vi.mock("../lib/hooks", () => ({
  useScenarios: () => ({ data: [{ name: "demo", display_name: "Demo", description: "Test scenario", status: "running", health_status: "healthy", runtime: "registry" }], isLoading: false, isError: false }),
}));

const cases = [
  { name: "discard", render: (props: Actions) => <DiscardConfirmationModal {...props} files={[{ path: "src/file.ts", untracked: false }]} /> },
  { name: "delete", render: (props: Actions) => <DeleteConfirmationModal {...props} path="src/file.ts" isDirectory={false} /> },
  { name: "stage", render: (props: Actions) => <StageSameNameConfirmationModal {...props} files={["src/file.ts"]} /> },
];
type Actions = { isOpen: boolean; isLoading: boolean; onCancel: () => void; onConfirm: () => void };

describe("mobile confirmation drawers", () => {
  it.each(cases)("$name keeps actions in the library footer and refuses dismissal during work", ({ render }) => {
    setViewport(390, 844);
    const actions = { isOpen: true, isLoading: true, onCancel: vi.fn(), onConfirm: vi.fn() };
    const { rerender } = renderWithProviders(render(actions));
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveClass("rcl-full-page-drawer__panel");
    expect(within(dialog).getByRole("button", { name: "Cancel" }).closest("footer")).not.toBeNull();
    expect(within(dialog).getByRole("button", { name: "Cancel" })).toBeDisabled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Close" }));
    expect(actions.onCancel).not.toHaveBeenCalled();
    expect(actions.onConfirm).not.toHaveBeenCalled();
    rerender(render({ ...actions, isLoading: false }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(actions.onCancel).toHaveBeenCalledOnce();
  });
});

it("keeps the mobile scenario picker searchable and preserves selection callbacks", () => {
  setViewport(390, 844);
  const onClose = vi.fn();
  const onSelect = vi.fn();
  renderWithProviders(<ScenarioPickerModal isOpen onClose={onClose} onSelect={onSelect} currentScenario="demo" />);
  const dialog = screen.getByRole("dialog", { name: "Select Scenario" });
  expect(dialog).toHaveClass("rcl-full-page-drawer__panel");
  const search = within(dialog).getByPlaceholderText("Search scenarios...");
  fireEvent.change(search, { target: { value: "missing" } });
  expect(within(dialog).getByText("No scenarios match your search.")).toBeInTheDocument();
  fireEvent.change(search, { target: { value: "Demo" } });
  fireEvent.click(within(dialog).getByRole("button", { name: /Demo/ }));
  expect(onSelect).toHaveBeenCalledWith("demo");
  expect(onClose).toHaveBeenCalledOnce();
});
