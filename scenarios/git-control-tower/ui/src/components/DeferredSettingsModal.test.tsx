import { useState } from "react";
import { fireEvent, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { DeferredSettingsModal } from "./DeferredSettingsModal";

const loaded = vi.hoisted(() => vi.fn());
vi.mock("./SettingsModal", () => {
  loaded();
  return { SettingsModal: ({ isOpen }: { isOpen: boolean }) => {
    const [draft, setDraft] = useState("");
    return isOpen ? <input aria-label="Settings draft" value={draft} onChange={event => setDraft(event.target.value)} /> : null;
  } };
});

it("defers settings until first open and preserves form state across reopen", async () => {
  const props = {
    isOpen: false, preset: "classic" as const, primaryPanel: "diff" as const,
    onChangePreset: vi.fn(), onChangePrimary: vi.fn(), onResetLayout: vi.fn(),
    groupingEnabled: false, onToggleGrouping: vi.fn(), groupingRules: [],
    onChangeGroupingRules: vi.fn(), onClose: vi.fn(),
  };
  const { rerender } = renderWithProviders(<DeferredSettingsModal {...props} />);
  expect(loaded).not.toHaveBeenCalled();
  rerender(<DeferredSettingsModal {...props} isOpen />);
  const input = await screen.findByRole("textbox", { name: "Settings draft" });
  fireEvent.change(input, { target: { value: "Retain my draft" } });
  rerender(<DeferredSettingsModal {...props} />);
  expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  rerender(<DeferredSettingsModal {...props} isOpen />);
  expect(screen.getByRole("textbox")).toHaveValue("Retain my draft");
  expect(loaded).toHaveBeenCalledOnce();
});
