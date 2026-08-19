import { createRef } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import SessionSidebar from "../SessionSidebar";
import { buildOriginBucketedNavigation, type OriginBucketNavigation } from "../../lib/workspaceNavigation";
import { useWorkspaceStore, type PaneMetadata } from "../../stores/useWorkspaceStore";
import type { SessionOriginName } from "../../api/sessions";

// The tab strip's visibility and contents are pure functions of the origin
// buckets; the sidebar's sync/group/resize hooks are orthogonal, so we stub
// them to keep the render deterministic and free of API/DOM plumbing.
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("../../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({ syncActivePane: vi.fn(), syncPaneUpdate: vi.fn(), syncPaneOrder: vi.fn() }),
}));
vi.mock("../../hooks/useGroupActions", () => ({
  useGroupActions: () => ({
    assignPaneToGroup: vi.fn(),
    removePaneFromGroup: vi.fn(),
    ungroupAllMembers: vi.fn(),
    deleteGroup: vi.fn(),
    createGroup: vi.fn(),
  }),
}));
vi.mock("../../hooks/useResizablePanel", () => ({
  useResizablePanel: () => ({ size: 300, isResizing: false, resizeHandleProps: {} }),
}));

const pane = (sessionId: string): PaneMetadata => ({
  sessionId,
  name: sessionId,
  headerColor: "transparent",
  themeId: "default",
  fontSize: 14,
  groupId: null,
  supportsMessagesView: false,
  manuallyUnread: false,
});

function buckets(origins: Record<string, SessionOriginName>): OriginBucketNavigation[] {
  const panes = Object.keys(origins).map((id) => pane(id));
  return buildOriginBucketedNavigation({
    panes,
    groups: [],
    activePane: panes[0]?.sessionId ?? null,
    originBySession: origins,
  });
}

function renderSidebar(bucketNav: OriginBucketNavigation[]) {
  return render(
    <SessionSidebar
      buckets={bucketNav}
      containerRef={createRef<HTMLElement>()}
      isMobile={false}
      mobileOpen={false}
      onCloseMobile={vi.fn()}
      onActivatePane={vi.fn()}
      onClosePane={vi.fn()}
      onDeletePanePermanently={vi.fn()}
      onNewTerminal={vi.fn()}
      onOpenLauncher={vi.fn()}
      onNewSessionInGroup={vi.fn()}
      onOpenSettings={vi.fn()}
    />,
  );
}

describe("SessionSidebar origin tabs", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ panes: [], sidebarOriginTab: "ui", sidebarSortMode: "manual", groups: [], tabContextMenu: null });
  });
  afterEach(() => cleanup());

  it("does not mount the tab strip when only UI-origin sessions exist", () => {
    renderSidebar(buckets({ a: "ui", b: "ui" }));
    expect(screen.queryByTestId("sidebar-origin-tabs")).toBeNull();
    // Every session is still shown — the sidebar renders exactly as before.
    expect(screen.getByTestId("sidebar-session-a")).not.toBeNull();
    expect(screen.getByTestId("sidebar-session-b")).not.toBeNull();
  });

  it("mounts the tab strip once a programmatic session appears", () => {
    renderSidebar(buckets({ a: "ui", b: "programmatic" }));
    expect(screen.getByTestId("sidebar-origin-tabs")).not.toBeNull();
    expect(screen.getByTestId("sidebar-origin-tab-ui")).not.toBeNull();
    expect(screen.getByTestId("sidebar-origin-tab-programmatic")).not.toBeNull();
    expect(screen.queryByTestId("sidebar-origin-tab-remote")).toBeNull();
  });

  it("shows only the active bucket's sessions and marks its tab selected", () => {
    // Persisted tab is "ui" (the default), so the UI session shows and the
    // programmatic one is filtered out of the active list.
    renderSidebar(buckets({ uiA: "ui", progB: "programmatic" }));
    expect(screen.getByTestId("sidebar-session-uiA")).not.toBeNull();
    expect(screen.queryByTestId("sidebar-session-progB")).toBeNull();
    expect(screen.getByTestId("sidebar-origin-tab-ui").getAttribute("aria-selected")).toBe("true");
    expect(screen.getByTestId("sidebar-origin-tab-programmatic").getAttribute("aria-selected")).toBe("false");
  });

  it("falls back to the first present bucket when the persisted tab's bucket is empty", () => {
    // Persisted active tab points at a bucket with no sessions; the sidebar
    // shows the first present bucket instead of an empty list, without needing
    // the store value changed.
    useWorkspaceStore.setState({ sidebarOriginTab: "remote" });
    renderSidebar(buckets({ uiA: "ui", progB: "programmatic" }));
    // No remote bucket exists, so the ui bucket (first present) is shown.
    expect(screen.getByTestId("sidebar-session-uiA")).not.toBeNull();
    expect(screen.getByTestId("sidebar-origin-tab-ui").getAttribute("aria-selected")).toBe("true");
    expect(useWorkspaceStore.getState().sidebarOriginTab).toBe("remote");
  });
});
