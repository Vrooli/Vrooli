// Grid geometry is already on the terminal wire; navigator.userAgent is not a
// trustworthy device identity and deliberately never participates here.
export type DeviceArchetype = "phone" | "tablet" | "laptop" | "monitor" | "ultrawide";

export function aspectForGrid(cols: number, rows: number, cellAspect: number): number { return (cols * cellAspect) / rows; }
export function archetypeForGrid(cols: number, rows: number, cellAspect: number): DeviceArchetype {
  const aspect = aspectForGrid(cols, rows, cellAspect);
  if (aspect < 1.1) return "phone";
  if (aspect < 1.6 && cols <= 110) return "tablet";
  if (aspect < 2.1) return "laptop";
  if (aspect < 3) return "monitor";
  return "ultrawide";
}
export function orientationForGrid(cols: number, rows: number, cellAspect: number): "portrait" | "landscape" { return aspectForGrid(cols, rows, cellAspect) < 1 ? "portrait" : "landscape"; }
