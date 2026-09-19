import { describe, it, expect, beforeEach } from "vitest";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

beforeEach(() => {
  useWorkspaceStore.setState({ pendingInputDrafts: {}, panes: [], activePane: null });
});

describe("pending input drafts (Layer 2.2/2.3)", () => {
  it("round-trips unsent input: snapshot then consume", () => {
    const store = useWorkspaceStore.getState();
    store.setPendingInputDraft("s1", "git stat");
    expect(useWorkspaceStore.getState().pendingInputDrafts.s1).toBe("git stat");

    const restored = useWorkspaceStore.getState().consumePendingInputDraft("s1");
    expect(restored).toBe("git stat");
    // Consuming clears it so it isn't re-injected twice.
    expect(useWorkspaceStore.getState().consumePendingInputDraft("s1")).toBeUndefined();
  });

  it("setting an empty draft clears any existing one", () => {
    useWorkspaceStore.getState().setPendingInputDraft("s1", "abc");
    useWorkspaceStore.getState().setPendingInputDraft("s1", "");
    expect(useWorkspaceStore.getState().pendingInputDrafts.s1).toBeUndefined();
  });

  it("drafts are isolated per session", () => {
    useWorkspaceStore.getState().setPendingInputDraft("s1", "one");
    useWorkspaceStore.getState().setPendingInputDraft("s2", "two");
    expect(useWorkspaceStore.getState().consumePendingInputDraft("s1")).toBe("one");
    expect(useWorkspaceStore.getState().consumePendingInputDraft("s2")).toBe("two");
  });

  it("closing a pane discards its stashed draft", () => {
    const store = useWorkspaceStore.getState();
    store.addPane("s1", "terminal", true);
    store.setPendingInputDraft("s1", "leftover");
    store.removePane("s1");
    expect(useWorkspaceStore.getState().pendingInputDrafts.s1).toBeUndefined();
  });
});
