import { beforeEach, describe, expect, it } from "vitest";

import {
  resetWorkspaceIntent,
  setWorkspaceIntent,
  takeWorkspaceIntent,
} from "./workspaceIntent";

describe("workspaceIntent", () => {
  beforeEach(() => {
    resetWorkspaceIntent();
  });

  it("returns null when nothing is staged", () => {
    expect(takeWorkspaceIntent()).toBeNull();
  });

  it("returns a staged intent exactly once (StrictMode re-mount is a no-op)", () => {
    const file = new File(["x"], "a.png", { type: "image/png" });
    setWorkspaceIntent({ file, mode: "enhance", operation: "upscale" });

    const taken = takeWorkspaceIntent();
    expect(taken?.mode).toBe("enhance");
    expect(taken?.operation).toBe("upscale");
    expect(taken?.file).toBe(file);

    // A second take (StrictMode double-mount, or a later manual /workspace
    // visit) must NOT re-apply the same intent.
    expect(takeWorkspaceIntent()).toBeNull();
  });

  it("applies the latest staged intent after a fresh set", () => {
    setWorkspaceIntent({ mode: "edit", operation: "crop" });
    expect(takeWorkspaceIntent()?.operation).toBe("crop");

    setWorkspaceIntent({ mode: "create" });
    const taken = takeWorkspaceIntent();
    expect(taken?.mode).toBe("create");
    expect(taken?.operation).toBeUndefined();
  });
});
