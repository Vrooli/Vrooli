import { act, fireEvent, renderWithProviders } from "@/test-utils";
import { useAppShortcuts } from "./useAppShortcuts";
import { useKeyboardShortcutHandler, useRegisterShortcuts, useShortcutContext } from "@hooks/useKeyboardShortcuts";
import { useKeyboardShortcutsStore } from "@stores/keyboardShortcutsStore";

type AppShortcutParams = Parameters<typeof useAppShortcuts>[0];

function ShortcutHost({ params }: { params: AppShortcutParams }) {
  useAppShortcuts(params);
  return null;
}

function ShortcutRegistryHost({ actions }: { actions: Record<string, () => void> }) {
  useKeyboardShortcutHandler();
  useShortcutContext("workflow-builder");
  useRegisterShortcuts(actions);
  return null;
}

describe("useAppShortcuts", () => {
  it("keeps replacement handlers registered after app view changes", async () => {
    const firstOpenDocs = vi.fn<AppShortcutParams["openDocs"]>();
    const replacementOpenDocs = vi.fn<AppShortcutParams["openDocs"]>();
    const noop = vi.fn();
    const initial: AppShortcutParams = {
      currentView: "dashboard",
      showAIModal: false,
      showProjectModal: false,
      showDocs: false,
      showTour: false,
      openDocs: firstOpenDocs,
      closeDocs: noop,
      closeAIModal: noop,
      closeProjectModal: noop,
      navigateToDashboard: noop,
      navigateToSettings: noop,
      openProject: noop,
      openProjectModal: noop,
      openAIModal: noop,
      openTour: noop,
      handleStartRecording: noop,
      currentProject: null,
    };

    const { rerender, unmount } = renderWithProviders(<ShortcutHost params={initial} />);
    rerender(<ShortcutHost params={{ ...initial, currentView: "settings", openDocs: replacementOpenDocs }} />);

    // useRegisterShortcuts defers teardown to a microtask to avoid updating
    // Zustand during React cleanup. Flush it before exercising the user's key.
    await act(async () => {
      await new Promise<void>((resolve) => queueMicrotask(resolve));
    });

    fireEvent.keyDown(document, { key: "?", shiftKey: true });

    expect(replacementOpenDocs).toHaveBeenCalledExactlyOnceWith("shortcuts");
    expect(firstOpenDocs).not.toHaveBeenCalled();

    unmount();
    await act(async () => {
      await new Promise<void>((resolve) => queueMicrotask(resolve));
    });
    expect(useKeyboardShortcutsStore.getState().actions.has("show-shortcuts")).toBe(false);
  });

  it("keeps the same callback registered when a fresh action map replaces it", async () => {
    const saveWorkflow = vi.fn();
    const { rerender, unmount } = renderWithProviders(
      <ShortcutRegistryHost actions={{ "save-workflow": saveWorkflow }} />,
    );
    rerender(<ShortcutRegistryHost actions={{ "save-workflow": saveWorkflow }} />);

    await act(async () => {
      await new Promise<void>((resolve) => queueMicrotask(resolve));
    });

    fireEvent.keyDown(document, { key: "s", metaKey: true });
    expect(saveWorkflow).toHaveBeenCalledExactlyOnceWith();

    unmount();
    await act(async () => {
      await new Promise<void>((resolve) => queueMicrotask(resolve));
    });
    expect(useKeyboardShortcutsStore.getState().actions.has("save-workflow")).toBe(false);
  });
});
