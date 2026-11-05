export const MAX_MODEL_BYTES = 20 * 1024 * 1024;
export const MAX_MODEL_VERTICES = 600000;
export const MAX_MODEL_TRIANGLES = 200000;
export type ModelFormat = "glb" | "gltf" | "obj" | "stl";
export type ModelError = "fetch" | "invalid" | "limit" | "external" | "unsupported" | "unavailable" | "context";
export type ModelView = "perspective" | "front" | "right" | "top";
export interface ModelOptions { wireframe: boolean; grid: boolean; background: "studio" | "light" | "dark"; lighting: "studio" | "soft"; rotate: boolean; }
export const DEFAULT_MODEL_OPTIONS: ModelOptions = { wireframe: false, grid: true, background: "studio", lighting: "studio", rotate: false };
export interface ModelStats { meshes: number; triangles: number; dimensions: [number, number, number]; }
export type PreparedModel = { format: "glb" | "gltf"; bytes: ArrayBuffer; uris: string[] } | { format: "obj" | "stl"; positions: Float32Array; normals?: Float32Array; partial: boolean };
export function modelFormat(name: string): ModelFormat | null {
  const extension = name.split(".").pop()?.toLowerCase();
  return extension === "glb" || extension === "gltf" || extension === "obj" || extension === "stl" ? extension : null;
}
export function modelError(error: unknown): ModelError {
  if (error instanceof RangeError) return "limit";
  return error instanceof Error && ["fetch", "invalid", "external", "unsupported", "context", "unavailable"].includes(error.message) ? error.message as ModelError : "invalid";
}
// A bounding sphere must fit both axes, including a portrait drawer. The near
// plane stays proportional to avoid clipping small models or losing depth.
export function modelCameraDistance(radius: number, aspect: number, fieldOfView = 40): number {
  const vertical = fieldOfView * Math.PI / 360;
  const horizontal = Math.atan(Math.tan(vertical) * Math.max(0.01, aspect));
  return radius / Math.sin(Math.min(vertical, horizontal)) * 1.16;
}
