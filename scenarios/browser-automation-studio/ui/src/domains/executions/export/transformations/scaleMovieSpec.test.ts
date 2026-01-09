import { describe, expect, it } from "vitest";
import type { ReplayMovieSpec } from "@/types/export";
import {
  isScalingNeeded,
  scaleFrames,
  scaleMovieSpec,
  scalePresentation,
} from "./scaleMovieSpec";

describe("scaleMovieSpec", () => {
  const createMockSpec = (
    canvasWidth = 1280,
    canvasHeight = 720,
  ): ReplayMovieSpec => ({
    version: "1.0",
    generated_at: new Date().toISOString(),
    execution: {
      execution_id: "test-123",
      workflow_id: "workflow-456",
      workflow_name: "Test Workflow",
      started_at: new Date().toISOString(),
    },
    theme: {},
    cursor: {
      theme: "default",
      scale: 1,
    },
    decor: {},
    playback: {
      fps: 25,
      duration_ms: 5000,
      frame_interval_ms: 40,
      total_frames: 125,
    },
    presentation: {
      canvas: { width: canvasWidth, height: canvasHeight },
      viewport: { width: canvasWidth, height: canvasHeight },
      browser_frame: {
        x: 0,
        y: 0,
        width: canvasWidth,
        height: canvasHeight,
        radius: 24,
      },
      device_scale_factor: 1,
    },
    cursor_motion: {
      speed_profile: "natural",
      path_style: "bezier",
      animation_type: "smooth",
    },
    frames: [
      {
        index: 0,
        timestamp_ms: 0,
        duration_ms: 1000,
        step_id: "step-1",
        step_type: "click",
        viewport: { width: canvasWidth, height: canvasHeight },
      },
      {
        index: 1,
        timestamp_ms: 1000,
        duration_ms: 1000,
        step_id: "step-2",
        step_type: "type",
        viewport: { width: canvasWidth, height: canvasHeight },
      },
    ],
    assets: [],
    summary: {
      frame_count: 2,
      total_duration_ms: 5000,
    },
  });

  describe("scalePresentation", () => {
    it("scales presentation to target dimensions", () => {
      const basePresentation = {
        canvas: { width: 1280, height: 720 },
        viewport: { width: 1280, height: 720 },
        browser_frame: {
          x: 0,
          y: 0,
          width: 1280,
          height: 720,
          radius: 24,
        },
        device_scale_factor: 1,
      };

      const result = scalePresentation(basePresentation, {
        width: 1920,
        height: 1080,
      });

      expect(result.canvas).toEqual({ width: 1920, height: 1080 });
      expect(result.viewport).toEqual({ width: 1920, height: 1080 });
      expect(result.browser_frame?.width).toBe(1920);
      expect(result.browser_frame?.height).toBe(1080);
      expect(result.device_scale_factor).toBe(1);
    });

    it("handles downscaling", () => {
      const basePresentation = {
        canvas: { width: 1920, height: 1080 },
        viewport: { width: 1920, height: 1080 },
      };

      const result = scalePresentation(basePresentation, {
        width: 960,
        height: 540,
      });

      expect(result.canvas).toEqual({ width: 960, height: 540 });
      expect(result.viewport).toEqual({ width: 960, height: 540 });
    });

    it("handles undefined presentation", () => {
      const result = scalePresentation(undefined, {
        width: 1920,
        height: 1080,
      });

      expect(result.canvas).toEqual({ width: 1920, height: 1080 });
    });
  });

  describe("scaleFrames", () => {
    it("scales all frame viewports", () => {
      const frames = [
        { index: 0, viewport: { width: 1280, height: 720 } },
        { index: 1, viewport: { width: 1280, height: 720 } },
      ] as any[];

      const result = scaleFrames(
        frames,
        { width: 1280, height: 720 },
        { width: 1920, height: 1080 },
        { width: 1280, height: 720 },
      );

      expect(result).toHaveLength(2);
      expect(result[0].viewport).toEqual({ width: 1920, height: 1080 });
      expect(result[1].viewport).toEqual({ width: 1920, height: 1080 });
    });

    it("returns empty array for undefined frames", () => {
      const result = scaleFrames(
        undefined,
        { width: 1280, height: 720 },
        { width: 1920, height: 1080 },
        { width: 1280, height: 720 },
      );

      expect(result).toEqual([]);
    });
  });

  describe("scaleMovieSpec", () => {
    it("scales entire spec to target dimensions", () => {
      const spec = createMockSpec(1280, 720);

      const result = scaleMovieSpec(spec, {
        targetDimensions: { width: 1920, height: 1080 },
      });

      // Canvas should be target dimensions
      expect(result.presentation.canvas).toEqual({ width: 1920, height: 1080 });

      // Frames should be scaled
      expect(result.frames?.[0]?.viewport).toEqual({ width: 1920, height: 1080 });
      expect(result.frames?.[1]?.viewport).toEqual({ width: 1920, height: 1080 });

      // Original spec should be unchanged
      expect(spec.presentation.canvas).toEqual({ width: 1280, height: 720 });
    });

    it("preserves non-dimension properties", () => {
      const spec = createMockSpec();

      const result = scaleMovieSpec(spec, {
        targetDimensions: { width: 1920, height: 1080 },
      });

      expect(result.version).toBe(spec.version);
      expect(result.execution).toEqual(spec.execution);
      expect(result.playback).toEqual(spec.playback);
      expect(result.summary).toEqual(spec.summary);
    });

    it("handles non-uniform scaling", () => {
      const spec = createMockSpec(800, 600);

      const result = scaleMovieSpec(spec, {
        targetDimensions: { width: 1600, height: 900 },
      });

      expect(result.presentation.canvas).toEqual({ width: 1600, height: 900 });
      expect(result.frames?.[0]?.viewport).toEqual({ width: 1600, height: 900 });
    });
  });

  describe("isScalingNeeded", () => {
    it("returns true when dimensions differ", () => {
      expect(
        isScalingNeeded({ width: 1280, height: 720 }, { width: 1920, height: 1080 }),
      ).toBe(true);
    });

    it("returns true when only width differs", () => {
      expect(
        isScalingNeeded({ width: 1280, height: 720 }, { width: 1920, height: 720 }),
      ).toBe(true);
    });

    it("returns true when only height differs", () => {
      expect(
        isScalingNeeded({ width: 1280, height: 720 }, { width: 1280, height: 1080 }),
      ).toBe(true);
    });

    it("returns false when dimensions are equal", () => {
      expect(
        isScalingNeeded({ width: 1920, height: 1080 }, { width: 1920, height: 1080 }),
      ).toBe(false);
    });
  });
});
