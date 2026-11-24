import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { WorkspaceTopBar } from "./WorkspaceTopBar";

describe("WorkspaceTopBar capture patch action", () => {
  afterEach(() => cleanup());

  it("opens the capture-outline patch review surface from workspace chrome", async () => {
    await setLocale("en");
    const onImportPatch = vi.fn();
    renderWithProviders(
      <WorkspaceTopBar
        workspaceName="Settings"
        pages={[{ id: "page-settings", name: "Settings", displays: [] } as never]}
        activePageId="page-settings"
        onSelectPage={vi.fn()}
        onAddPage={vi.fn()}
        onDuplicatePage={vi.fn()}
        onImportHtml={vi.fn()}
        onImportPatch={onImportPatch}
        onRemovePage={vi.fn()}
        canUndo={false}
        canRedo={false}
        onUndo={vi.fn()}
        onRedo={vi.fn()}
        dirty={false}
        saving={false}
        saveFailed={false}
        onSave={vi.fn()}
        device="phone"
        onDevice={vi.fn()}
        onPresent={vi.fn()}
        onShare={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Import capture outline patch" }));
    expect(onImportPatch).toHaveBeenCalledOnce();
  });
});
