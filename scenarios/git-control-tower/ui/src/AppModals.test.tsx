import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithQueryClient } from "./test-utils";
import { AppModals, type AppModalsProps } from "./AppModals";

vi.mock("./components/SettingsModal", () => ({
  SettingsModal: ({
    isOpen,
    groupingEnabled,
    repoId,
    onClose,
    onToggleGrouping,
  }: {
    isOpen: boolean;
    groupingEnabled: boolean;
    repoId: string | null;
    onClose: () => void;
    onToggleGrouping: () => void;
  }) =>
    isOpen ? (
      <section data-testid="settings-modal" data-grouping={String(groupingEnabled)} data-repo-id={repoId ?? ""}>
        <button type="button" onClick={onToggleGrouping}>toggle grouping</button>
        <button type="button" onClick={onClose}>close settings</button>
      </section>
    ) : null,
}));

vi.mock("./components/UpstreamInfoModal", () => ({
  UpstreamInfoModal: ({
    isOpen,
    localBranch,
    upstreamRef,
    ahead,
    behind,
    onClose,
  }: {
    isOpen: boolean;
    localBranch?: string;
    upstreamRef?: string;
    ahead: number;
    behind: number;
    onClose: () => void;
  }) =>
    isOpen ? (
      <section data-testid="upstream-modal">
        <span>{localBranch}</span>
        <span>{upstreamRef}</span>
        <span>{ahead}/{behind}</span>
        <button type="button" onClick={onClose}>close upstream</button>
      </section>
    ) : null,
}));

vi.mock("./components/DiscardConfirmationModal", () => ({
  DiscardConfirmationModal: ({
    isOpen,
    files,
    isLoading,
    onConfirm,
    onCancel,
  }: {
    isOpen: boolean;
    files: Array<{ path: string; untracked: boolean }>;
    isLoading: boolean;
    onConfirm: () => void;
    onCancel: () => void;
  }) =>
    isOpen ? (
      <section data-testid="discard-modal" data-loading={String(isLoading)}>
        <span>{files.map((file) => file.path).join(",")}</span>
        <button type="button" onClick={onConfirm}>confirm discard</button>
        <button type="button" onClick={onCancel}>cancel discard</button>
      </section>
    ) : null,
}));

vi.mock("./components/DeleteConfirmationModal", () => ({
  DeleteConfirmationModal: ({
    isOpen,
    path,
    isDirectory,
    isLoading,
    onConfirm,
    onCancel,
  }: {
    isOpen: boolean;
    path: string;
    isDirectory: boolean;
    isLoading: boolean;
    onConfirm: () => void;
    onCancel: () => void;
  }) =>
    isOpen ? (
      <section data-testid="delete-modal" data-directory={String(isDirectory)} data-loading={String(isLoading)}>
        <span>{path}</span>
        <button type="button" onClick={onConfirm}>confirm delete</button>
        <button type="button" onClick={onCancel}>cancel delete</button>
      </section>
    ) : null,
}));

vi.mock("./components/FileSearchModal", () => ({
  FileSearchModal: ({
    isOpen,
    repoId,
    onClose,
    onSelectFile,
  }: {
    isOpen: boolean;
    repoId: string | null;
    onClose: () => void;
    onSelectFile: (path: string, lineNumber?: number) => void;
  }) =>
    isOpen ? (
      <section data-testid="desktop-file-search" data-repo-id={repoId ?? ""}>
        <button type="button" onClick={() => onSelectFile("src/App.tsx", 42)}>select desktop file</button>
        <button type="button" onClick={onClose}>close file search</button>
      </section>
    ) : null,
}));

vi.mock("./components/MobileFileSearch", () => ({
  MobileFileSearch: ({
    isOpen,
    repoId,
    onClose,
    onSelectFile,
  }: {
    isOpen: boolean;
    repoId: string | null;
    onClose: () => void;
    onSelectFile: (path: string, lineNumber?: number) => void;
  }) =>
    isOpen ? (
      <section data-testid="mobile-file-search" data-repo-id={repoId ?? ""}>
        <button type="button" onClick={() => onSelectFile("src/Mobile.tsx")}>select mobile file</button>
        <button type="button" onClick={onClose}>close mobile search</button>
      </section>
    ) : null,
}));

function buildProps(overrides: Partial<AppModalsProps> = {}): AppModalsProps {
  return {
    isSettingsOpen: false,
    onCloseSettings: vi.fn(),
    repoDir: "/repo",
    repoId: "repo-1",
    syncStatus: undefined,
    layoutPreset: "classic",
    primaryPanel: "diff",
    onChangePreset: vi.fn(),
    onChangePrimary: vi.fn(),
    onResetLayout: vi.fn(),
    fileViewMode: "flat",
    onToggleGrouping: vi.fn(),
    groupingRules: [],
    onChangeGroupingRules: vi.fn(),
    isUpstreamInfoOpen: false,
    onCloseUpstreamInfo: vi.fn(),
    localBranch: "main",
    upstreamRef: "origin/main",
    upstreamAhead: 2,
    upstreamBehind: 1,
    pendingDiscardFiles: null,
    isDiscardLoading: false,
    onConfirmDiscard: vi.fn(),
    onCancelDiscard: vi.fn(),
    pendingDeletePath: null,
    isDeleteLoading: false,
    onConfirmDelete: vi.fn(),
    onCancelDelete: vi.fn(),
    isFileSearchOpen: false,
    onCloseFileSearch: vi.fn(),
    onSelectAnyFile: vi.fn(),
    isMobile: false,
    ...overrides,
  };
}

describe("AppModals", () => {
  it("routes open modal state and callbacks to desktop modal surfaces", async () => {
    const user = userEvent.setup();
    const props = buildProps({
      isSettingsOpen: true,
      fileViewMode: "grouped",
      isUpstreamInfoOpen: true,
      pendingDiscardFiles: [
        { path: "src/changed.ts", untracked: false },
        { path: "src/new.ts", untracked: true },
      ],
      pendingDeletePath: { path: "src/old.ts", isDir: false },
      isFileSearchOpen: true,
    });

    renderWithQueryClient(<AppModals {...props} />);

    expect(screen.getByTestId("settings-modal")).toHaveAttribute("data-grouping", "true");
    expect(screen.getByTestId("upstream-modal")).toHaveTextContent("main");
    expect(screen.getByTestId("upstream-modal")).toHaveTextContent("origin/main");
    expect(screen.getByTestId("discard-modal")).toHaveTextContent("src/changed.ts,src/new.ts");
    expect(screen.getByTestId("delete-modal")).toHaveAttribute("data-directory", "false");
    expect(screen.getByTestId("desktop-file-search")).toHaveAttribute("data-repo-id", "repo-1");
    expect(screen.queryByTestId("mobile-file-search")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "toggle grouping" }));
    expect(props.onToggleGrouping).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "confirm discard" }));
    expect(props.onConfirmDiscard).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "confirm delete" }));
    expect(props.onConfirmDelete).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "select desktop file" }));
    expect(props.onSelectAnyFile).toHaveBeenCalledWith("src/App.tsx", 42);
  });

  it("uses the mobile file search surface when the app is in mobile layout", async () => {
    const user = userEvent.setup();
    const props = buildProps({
      isMobile: true,
      isFileSearchOpen: true,
    });

    renderWithQueryClient(<AppModals {...props} />);

    expect(screen.getByTestId("mobile-file-search")).toHaveAttribute("data-repo-id", "repo-1");
    expect(screen.queryByTestId("desktop-file-search")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "select mobile file" }));
    expect(props.onSelectAnyFile).toHaveBeenCalledWith("src/Mobile.tsx");

    await user.click(screen.getByRole("button", { name: "close mobile search" }));
    expect(props.onCloseFileSearch).toHaveBeenCalledTimes(1);
  });
});
