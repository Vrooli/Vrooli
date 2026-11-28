import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { i18n, setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import type { DesignDocument } from "../design-types";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { sha256 as capturePatchEvidenceSha256 } from "./capturePatchEvidence";
import { CaptureOutlinePatchDialog } from "./CaptureOutlinePatchDialog";

const sourceBytes = new Blob(["captured-before"]);
const previewBytes = new Blob(["component-after"]);

async function makePatch() {
  return {
    schema: "rcl.capture-outline-patch/v1",
    patch_id: "patch-demo-1",
    source: {
      id: "capture-demo-1", scenario: "sample-scenario", build_id: "sha256:source", route: "/settings",
      state_id: "navigation-open", data_id: "synthetic-account-1", viewport: { width: 390, height: 844 },
      image: { path: "evidence/before.png", sha256: await capturePatchEvidenceSha256(sourceBytes) },
      outline: { path: "evidence/outline.json", sha256: "a".repeat(64) },
    },
    target: {
      region_id: "settings-primary-action", outline_node_id: "outline-24", design_frame_id: "frame-phone",
      design_node_ids: ["action-button"], bounds: { x: 0.1, y: 0.7, width: 0.8, height: 0.15 },
      component_identity: "Button/2.2.14",
    },
    preview: {
      id: "render-demo-1", scenario: "sample-scenario", build_id: "sha256:preview", route: "/settings",
      state_id: "navigation-open", data_id: "synthetic-account-1", viewport: { width: 390, height: 844 },
      fidelity: "rendered", image: { path: "evidence/after.png", sha256: await capturePatchEvidenceSha256(previewBytes) },
    },
    composition: {
      design_id: "design-demo-1", base_revision: 4, operation_id: "op-demo-1", summary: "Keep the primary action on one row.",
      operations: [{ kind: "setText", nodeId: "action-button", text: "Continue" }],
    },
  };
}

const design: DesignDocument = {
  id: "design-demo-1", name: "Settings", updatedAt: "2026-11-22T00:00:00Z", revision: 4,
  frames: [{ id: "frame-phone", name: "Phone", device: "phone", spec: {
    id: "page-root", type: "page", props: {}, children: [{ id: "action-button", type: "button", props: { text: "Next" } }],
  } }],
  theme: { ...DEFAULT_TEMPLATE.palette },
};

async function setup() {
  const patch = await makePatch();
  const onApply = vi.fn();
  renderWithProviders(
    <CaptureOutlinePatchDialog design={design} activeFrameId="frame-phone" readOnly={false} onClose={vi.fn()} onApply={onApply} />,
  );
  const patchFile = new File([JSON.stringify(patch)], "region-patch.json", { type: "application/json" });
  fireEvent.change(screen.getByTestId(selectors.workspaceStudio.capturePatchFile), { target: { files: [patchFile] } });
  await screen.findByText(patch.target.region_id);
  return { patch, onApply };
}

describe("CaptureOutlinePatchDialog", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:verified-evidence"),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("reviews a patch and applies it only after both image hashes are verified", async () => {
    const { onApply } = await setup();
    const apply = screen.getByRole("button", { name: i18n.t(strings.workbench.studio.patchApply) });
    expect(apply).toBeDisabled();

    const before = new File([sourceBytes], "before.png", { type: "image/png" });
    const after = new File([previewBytes], "after.png", { type: "image/png" });
    fireEvent.change(screen.getByTestId(selectors.workspaceStudio.capturePatchEvidence), { target: { files: [before, after] } });
    await screen.findByText(i18n.t(strings.workbench.studio.patchHashesMatch));
    const beforeLabel = i18n.t(strings.workbench.studio.patchCapturedBefore);
    const afterLabel = i18n.t(strings.workbench.studio.patchComponentAfter);
    expect(screen.getByRole("img", {
      name: i18n.t(strings.workbench.studio.patchEvidenceAlt, { label: beforeLabel, region: "settings-primary-action" }),
    })).toBeInTheDocument();
    expect(screen.getByRole("img", {
      name: i18n.t(strings.workbench.studio.patchEvidenceAlt, { label: afterLabel, region: "settings-primary-action" }),
    })).toBeInTheDocument();

    fireEvent.click(apply);
    expect(onApply).toHaveBeenCalledOnce();
    expect(onApply.mock.calls[0]?.[0].doc.frames[0]?.spec.children?.[0]?.props.text).toBe("Continue");
    expect(onApply).toHaveBeenCalledWith(expect.any(Object), "patch-demo-1", "Keep the primary action on one row.");
  });

  it("keeps stale patches visible but refuses their application", async () => {
    const patch = await makePatch();
    patch.composition.base_revision = 3;
    renderWithProviders(
      <CaptureOutlinePatchDialog design={design} activeFrameId="frame-phone" readOnly={false} onClose={vi.fn()} onApply={vi.fn()} />,
    );
    fireEvent.change(screen.getByTestId(selectors.workspaceStudio.capturePatchFile), {
      target: { files: [new File([JSON.stringify(patch)], "stale.json", { type: "application/json" })] },
    });
    expect(await screen.findByText(i18n.t(strings.workbench.studio.patchStaleTarget))).toBeInTheDocument();
    expect(screen.getByRole("button", { name: i18n.t(strings.workbench.studio.patchApply) })).toBeDisabled();
  });

  it("refuses image evidence whose digest does not match the patch", async () => {
    await setup();
    const before = new File(["changed"], "before.png", { type: "image/png" });
    const after = new File([previewBytes], "after.png", { type: "image/png" });
    fireEvent.change(screen.getByTestId(selectors.workspaceStudio.capturePatchEvidence), { target: { files: [before, after] } });
    expect(await screen.findByRole("alert")).toHaveTextContent(
      i18n.t(strings.workbench.studio.patchImageHashMismatch, { name: "before.png" }),
    );
    expect(screen.getByRole("button", { name: i18n.t(strings.workbench.studio.patchApply) })).toBeDisabled();
  });
});
