import { describe, expect, it } from "vitest";
import { boundImage, fitImage, rotatedDimensions, zoomImage } from "../imageGeometry";

describe("image inspection geometry", () => {
  const image = { width: 1000, height: 800 };
  const stage = { width: 400, height: 300 };
  it("fits both axes without upscaling small images", () => {
    expect(fitImage(image, stage)).toBe(0.375);
    expect(fitImage({ width: 100, height: 100 }, stage)).toBe(1);
    expect(fitImage(image, { width: 0, height: 0 })).toBe(1);
  });
  it("keeps the pointed-at image coordinate stable during magnification", () => {
    const before = { scale: 1, x: 20, y: -10 };
    const at = { x: 100, y: 50 };
    const after = zoomImage(before, 2, at, image, stage);
    expect((at.x - after.x) / after.scale).toBe((at.x - before.x) / before.scale);
    expect((at.y - after.y) / after.scale).toBe((at.y - before.y) / before.scale);
  });
  it("clamps panning at every image edge and centers an image smaller than the stage", () => {
    expect(boundImage({ scale: 1, x: 900, y: -900 }, image, stage)).toEqual({ scale: 1, x: 300, y: -250 });
    expect(boundImage({ scale: 0.2, x: 100, y: -100 }, image, stage)).toEqual({ scale: 0.2, x: 0, y: 0 });
    expect(zoomImage({ scale: 1, x: 0, y: 0 }, 50, { x: 0, y: 0 }, image, stage).scale).toBe(8);
  });
  it("fits rotated portrait/landscape bounds independently", () => {
    expect(rotatedDimensions(image, 90)).toEqual({ width: 800, height: 1000 });
    expect(fitImage(rotatedDimensions(image, 90), stage)).toBe(0.3);
    expect(rotatedDimensions(image, 180)).toEqual(image);
  });
});
