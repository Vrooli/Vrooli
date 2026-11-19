import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DEFAULT_TEMPLATE } from "../templates/library";
import { docSignature, seedInput } from "./model";
import { walk, type TemplateNode } from "../templates/spec";

const base = seedInput(DEFAULT_TEMPLATE, "Sample");
const first = "design-b";
const second = "design-a";

function makeDoc(id: string, name: string, updatedAt: string) {
  return {
    id,
    name,
    updatedAt,
    revision: 1,
    frames: base.frames,
    theme: base.theme,
    options: { ...(base.options ?? {}) },
    scenario: null,
    page: null,
  };
}

const docs = [
  makeDoc(first, "Older", "2026-11-18T10:00:00.000Z"),
  makeDoc(second, "Newer", "2026-11-19T10:00:00.000Z"),
];

vi.mock("../designs", () => ({
  DesignConflictError: class DesignConflictError extends Error {
    current: unknown;
    constructor(current: unknown) {
      super("design was modified elsewhere");
      this.name = "DesignConflictError";
      this.current = current;
    }
  },
  createDesign: vi.fn(() => Promise.resolve(docs[0])),
  deleteDesign: vi.fn(() => Promise.resolve()),
  deleteDesignAsset: vi.fn(() => Promise.resolve()),
  designAssetUrl: vi.fn(() => "/asset"),
  designsUsingFallback: vi.fn(() => false),
  getDesign: vi.fn((id: string) => Promise.resolve(docs.find((doc) => doc.id === id))),
  listDesignAssets: vi.fn(() => Promise.resolve([])),
  listDesigns: vi.fn(() =>
    Promise.resolve(
      docs.map((doc) => ({
        id: doc.id,
        name: doc.name,
        updatedAt: doc.updatedAt,
        frameCount: doc.frames.length,
        scenario: null,
        page: null,
      })),
    ),
  ),
  saveDesign: vi.fn((doc: unknown) => Promise.resolve(doc)),
  putDesign: vi.fn((doc: unknown) => Promise.resolve({ design: doc, local: false })),
  uploadDesignAsset: vi.fn((designId: string) =>
    Promise.resolve({
      id: "asset-1",
      designId,
      name: "capture.png",
      mime: "image/png",
      bytes: 3,
      width: 1,
      height: 1,
      fidelity: "captured",
      addedAt: "2026-11-19T10:00:00.000Z",
      url: `/api/designs/${designId}/assets/asset-1`,
    }),
  ),
}));

// `vi.mock` is hoisted above this import, so the hook sees the fake client.
import { useStudio } from "./useStudio";
import { DesignConflictError, putDesign, uploadDesignAsset } from "../designs";
import type { DesignDocument } from "../designs";

const ACTIVE_KEY = "rcl.studio.active-design";

describe("studio tab restore", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("restores the design the person was last editing", async () => {
    window.localStorage.setItem(ACTIVE_KEY, first);
    const { result } = renderHook(() => useStudio());
    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activeDoc?.id).toBe(first);
    // The provisional draft id must never replace the remembered one.
    expect(window.localStorage.getItem(ACTIVE_KEY)).toBe(first);
  });

  it("falls back to the newest design when nothing is remembered", async () => {
    const { result } = renderHook(() => useStudio());
    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activeDoc?.id).toBe(second);
  });
});


describe("studio proposals and assets", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  const open = async () => {
    const hook = renderHook(() => useStudio());
    await waitFor(() => expect(hook.result.current.ready).toBe(true));
    return hook;
  };

  const firstText = (doc: { frames: Array<{ spec: TemplateNode }> }): TemplateNode => {
    let found: TemplateNode | undefined;
    for (const frame of doc.frames) {
      walk(frame.spec, (node) => {
        if (!found && node.type === "text") found = node;
      });
    }
    if (!found) throw new Error("the sample design has no text layer");
    return found;
  };

  it("applies a proposal as one undoable command and refuses a stale one", async () => {
    const { result } = await open();
    const doc = result.current.activeDoc;
    if (!doc) throw new Error("no active design");
    const target = firstText(doc);
    const original = target.props.text;
    const proposal = {
      summary: "Sharper headline",
      operations: [{ kind: "setText" as const, nodeId: target.id, text: "Proposed copy" }],
      revision: doc.revision,
      operationId: "inference_1",
      state: "completed",
      signature: docSignature(doc),
      designId: doc.id,
    };
    let outcome: ReturnType<typeof result.current.applyProposal> | undefined;
    act(() => {
      outcome = result.current.applyProposal(proposal);
    });
    expect(outcome).toEqual({ ok: true });
    await waitFor(() => {
      expect(firstText(result.current.activeDoc as typeof doc).props.text).toBe("Proposed copy");
    });
    expect(result.current.history.entries).toHaveLength(1);
    expect(result.current.history.entries[0]?.label).toBe("AI · Sharper headline");

    act(() => result.current.undo());
    expect(firstText(result.current.activeDoc as typeof doc).props.text).toBe(original);

    // An autosave between propose and apply bumps the revision but not the
    // document; the patch is still valid.
    const appliedAfterSave = result.current.applyProposal({
      ...proposal,
      revision: doc.revision + 1,
      operations: [{ kind: "setText", nodeId: target.id, text: "Still valid" }],
    });
    expect(appliedAfterSave).toEqual({ ok: true });

    const stale = result.current.applyProposal({ ...proposal, signature: `${proposal.signature}!` });
    expect(stale).toEqual({ ok: false, reason: "stale", detail: expect.any(String) });

    // Same document, different design: a slow model response must never patch
    // the tab the person switched to.
    const wrongDesign = result.current.applyProposal({ ...proposal, designId: "another-design" });
    expect(wrongDesign).toEqual({ ok: false, reason: "stale", detail: expect.any(String) });
  });

  it("refuses a patch that the live document cannot apply", async () => {
    const { result } = await open();
    const doc = result.current.activeDoc;
    if (!doc) throw new Error("no active design");
    const outcome = result.current.applyProposal({
      summary: "Remove everything",
      operations: [{ kind: "removeLayer", nodeId: doc.frames[0]?.spec.id ?? "page" }],
      revision: doc.revision,
      operationId: "inference_2",
      state: "completed",
      signature: docSignature(doc),
      designId: doc.id,
    });
    expect(outcome.ok).toBe(false);
    if (!outcome.ok) expect(outcome.reason).toBe("invalid");
  });

  it("changes design chrome as one undoable step", async () => {
    const { result } = await open();
    const before = result.current.activeDoc?.options?.radius;
    act(() => result.current.updateOptions({ radius: 2 }, "Corner radius"));
    expect(result.current.activeDoc?.options?.radius).toBe(2);
    expect(result.current.history.entries).toHaveLength(1);
    act(() => result.current.undo());
    expect(result.current.activeDoc?.options?.radius).toBe(before);
  });

  it("forwards capture fidelity to the asset upload", async () => {
    const { result } = await open();
    const file = new File([new Uint8Array([1, 2, 3])], "capture.png", { type: "image/png" });
    await act(async () => {
      await result.current.importAsset(file, "captured");
    });
    expect(vi.mocked(uploadDesignAsset)).toHaveBeenCalledWith(second, file, "captured");
  });
});


describe("conflict resolution", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("re-saves the local document against the server revision on Keep mine", async () => {
    const hook = renderHook(() => useStudio());
    await waitFor(() => expect(hook.result.current.ready).toBe(true));
    const doc = hook.result.current.activeDoc;
    if (!doc) throw new Error("no active design");
    const remote: DesignDocument = { ...doc, revision: 7, name: "Remote version" };
    const write = vi.mocked(putDesign);
    write
      .mockRejectedValueOnce(new DesignConflictError(remote))
      .mockImplementation((next: unknown) =>
        Promise.resolve({ design: next as DesignDocument, local: false }),
      );

    act(() => {
      hook.result.current.updateFrame(
        doc.frames[0]?.id ?? "",
        (spec) => ({ ...spec, props: { ...spec.props, pad: 40 } }),
        "Pad the page",
      );
    });
    await waitFor(() => expect(hook.result.current.conflict).not.toBeNull());

    act(() => hook.result.current.resolveConflict("keep-mine"));
    await waitFor(() => expect(write).toHaveBeenCalledTimes(2));
    const saved = write.mock.calls.at(-1)?.[0] as DesignDocument;
    expect(saved.revision).toBe(7);
    expect(saved.name).toBe(doc.name);
    await waitFor(() => expect(hook.result.current.conflict).toBeNull());
  });
});
