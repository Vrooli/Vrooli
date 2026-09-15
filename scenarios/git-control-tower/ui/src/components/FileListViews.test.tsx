import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithQueryClient } from "../test-utils";
import { FileListMobileActions } from "./FileListMobileActions";

describe("FileList view components", () => {
  it("routes mobile action sheet commands for grouped and ungrouped files", async () => {
    const user = userEvent.setup();
    const groupedProps = {
      mobileActionFile: "src/changed.ts",
      mobileActionFileInfo: {
        path: "src/changed.ts",
        isStaged: false,
        isUnstaged: true,
        isUntracked: false,
        isConflict: false,
      },
      onClose: vi.fn(),
      onStageFile: vi.fn(),
      onUnstageFile: vi.fn(),
      onDiscardFile: vi.fn(),
      onIgnoreFile: vi.fn(),
      openFileMetrics: vi.fn(),
      resolvedGroups: [{
        key: "manual:src",
        label: "Source",
        root: "src/",
        source: "manual",
        files: ["src/changed.ts"],
      }],
    };

    renderWithQueryClient(<FileListMobileActions {...groupedProps} />);

    await user.click(screen.getByRole("button", { name: /View Metrics/i }));
    expect(groupedProps.openFileMetrics).toHaveBeenCalledWith("src/changed.ts", "unstaged");
    expect(groupedProps.onClose).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: /^Stage/i }));
    expect(groupedProps.onStageFile).toHaveBeenCalledWith("src/changed.ts");

    await user.click(screen.getByRole("button", { name: /Ignore \(Source\)/i }));
    expect(groupedProps.onIgnoreFile).toHaveBeenCalledWith("src/changed.ts", "group", "src/");

    await user.click(screen.getByRole("button", { name: /Discard Changes/i }));
    expect(groupedProps.onDiscardFile).toHaveBeenCalledWith("src/changed.ts", false);
  });
});
