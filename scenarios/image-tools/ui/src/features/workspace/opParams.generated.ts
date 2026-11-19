// AUTO-GENERATED — do not edit by hand.
//
// Source : api/internal/capability/manifests.seed.json (params_schema)
// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)
// Drift  : api/internal/capability/render_paramtypes_test.go fails the build if this file is stale.
//
// The TypeScript mirror of the Go param types. An `opParams` literal typed with
// one of these interfaces cannot use a key the operation's schema does not
// declare.

export interface InpaintOpParams {
  inpaint_context_pixels?: number;
  inpaint_crop_to_mask?: boolean;
  inpaint_max_edge_px?: number;
  inpaint_neutralize_masked_pixels?: boolean;
}

export interface ObjectRemovalOpParamsBackgroundRepairRemovalRegionsItem {
  height: number;
  width: number;
  x: number;
  y: number;
}

export interface ObjectRemovalOpParamsBackgroundRepair {
  maskExpansionPixels?: number;
  removalRegions?: ObjectRemovalOpParamsBackgroundRepairRemovalRegionsItem[];
}

export interface ObjectRemovalOpParams {
  background_repair?: ObjectRemovalOpParamsBackgroundRepair;
  grow_px?: number;
  halo_threshold?: number;
}

export interface SegmentOpParamsSegmentationBox {
  height: number;
  width: number;
  x: number;
  y: number;
}

export interface SegmentOpParamsSegmentationPointsItem {
  negative?: boolean;
  x: number;
  y: number;
}

export interface SegmentOpParamsSegmentation {
  box?: SegmentOpParamsSegmentationBox;
  points?: SegmentOpParamsSegmentationPointsItem[];
}

export interface SegmentOpParams {
  segmentation?: SegmentOpParamsSegmentation;
}

export interface SceneGeometryOpParamsSupportRoi {
  height: number;
  width: number;
  x: number;
  y: number;
}

export interface SceneGeometryOpParams {
  support_roi?: SceneGeometryOpParamsSupportRoi;
}

export interface ImageToObjectOpParams {
  mesh_resolution?: number;
}

export interface AiOpParamsByOperation {
  "image_to_object": ImageToObjectOpParams;
  "inpaint": InpaintOpParams;
  "object_removal": ObjectRemovalOpParams;
  "scene_geometry": SceneGeometryOpParams;
  "segment": SegmentOpParams;
}
