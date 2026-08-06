import axes from "../../../../experience-manager/capabilities/axes.json";

type ViewportValue = {
  id: string;
  params: { width: number; height: number };
};

type ViewportAxis = {
  id: "viewport";
  values: ViewportValue[];
};

function isViewportAxis(value: unknown): value is ViewportAxis {
  if (!value || typeof value !== "object") return false;
  const candidate = value as { id?: unknown; values?: unknown };
  if (candidate.id !== "viewport" || !Array.isArray(candidate.values)) return false;
  return candidate.values.every((entry) => {
    if (!entry || typeof entry !== "object") return false;
    const viewport = entry as { id?: unknown; params?: unknown };
    if (typeof viewport.id !== "string" || !viewport.params || typeof viewport.params !== "object") return false;
    const params = viewport.params as { width?: unknown; height?: unknown };
    return typeof params.width === "number" && typeof params.height === "number";
  });
}

const rawAxes: unknown = axes;
const registryAxes = rawAxes && typeof rawAxes === "object" && "axes" in rawAxes ? rawAxes.axes : [];
const viewportAxis = Array.isArray(registryAxes)
  ? registryAxes.find((value: unknown): value is ViewportAxis => isViewportAxis(value))
  : undefined;

if (!viewportAxis) {
  throw new Error("experience-manager viewport axis is missing or malformed");
}

export const VIEWPORT_AXIS_PRESETS = viewportAxis.values.map((value) => ({
  id: value.id,
  width: value.params.width,
  height: value.params.height,
})) as readonly { id: string; width: number; height: number }[];

export type ViewportAxisId = (typeof VIEWPORT_AXIS_PRESETS)[number]["id"];
