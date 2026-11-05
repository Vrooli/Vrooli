import { describe, expect, it } from "vitest";
import { shouldFocusArtifactViewer } from "./artifactViewerLayout";

describe("artifact viewer layout", () => {
  it("keeps a usable companion split when there is enough room", () => {
    expect(shouldFocusArtifactViewer(1100, 440)).toBe(false);
  });

  it("snaps to focus mode before the workspace becomes cramped", () => {
    expect(shouldFocusArtifactViewer(900, 440)).toBe(true);
  });

  it("uses the minimum viewer width for an undersized saved width", () => {
    expect(shouldFocusArtifactViewer(800, 200)).toBe(true);
  });
});
