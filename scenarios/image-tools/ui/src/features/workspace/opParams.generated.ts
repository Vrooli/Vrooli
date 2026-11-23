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
  inpaint_feather_px?: number;
  inpaint_grow_px?: number;
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

export interface FaceSwapOpParams {
  all_faces?: boolean;
  color_match?: number;
  extent?: string;
  face_strength?: number;
  match?: string[];
  occlusion?: boolean;
  prompt?: string;
  seed?: number;
  skin_target?: string;
  source_face?: number;
  target_face?: number;
}

export interface GarmentReplaceOpParams {
  inpaint_feather_px?: number;
  inpaint_grow_px?: number;
}

export interface ColorTransferOpParams {
  feather_px?: number;
  flatten?: number;
  luminance?: number;
  strength?: number;
}

export interface MaskCompositeOpParams {
  feather_px?: number;
  grow_px?: number;
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
  text?: string;
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
  "color_transfer": ColorTransferOpParams;
  "face_swap": FaceSwapOpParams;
  "garment_replace": GarmentReplaceOpParams;
  "image_to_object": ImageToObjectOpParams;
  "inpaint": InpaintOpParams;
  "mask_composite": MaskCompositeOpParams;
  "object_removal": ObjectRemovalOpParams;
  "scene_geometry": SceneGeometryOpParams;
  "segment": SegmentOpParams;
}
