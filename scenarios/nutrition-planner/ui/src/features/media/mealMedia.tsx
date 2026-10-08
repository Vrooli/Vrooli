import { useEffect, useState } from "react";
import { Leaf } from "lucide-react";

export type MealMediaRecord = {
  id: string;
  src: string;
  contentHash: string;
  source: { kind: "licensed" | "owned"; reference: string; rights: string };
  approval: { status: "approved"; reviewer: string; reviewedAt: string };
  compatibility: { recipeId: string; recipeRevision: number; appearance: "light" | "evening" | "any" };
  crop: { focalPoint: { x: number; y: number }; bounds: { x: number; y: number; width: number; height: number } };
};

// The admitted inventory is intentionally empty until owner-approved media is supplied.
export const mealMediaInventory: readonly MealMediaRecord[] = [];

function complete(record: MealMediaRecord, recipeId: string, recipeRevision: number, appearance: "light" | "evening") {
  const source = record?.source;
  const approval = record?.approval;
  const compatibility = record?.compatibility;
  const crop = record?.crop;
  if (!source || !approval || !compatibility || !crop || !crop.focalPoint || !crop.bounds) return false;
  const cropValues = [crop.focalPoint.x, crop.focalPoint.y, crop.bounds.x, crop.bounds.y, crop.bounds.width, crop.bounds.height];
  return Boolean(
    record.id?.trim() && /^sha256:[a-f0-9]{64}$/i.test(record.contentHash ?? "") &&
    (source.kind === "licensed" || source.kind === "owned") && source.reference?.trim() && source.rights?.trim() &&
    approval.reviewer?.trim() && Number.isFinite(Date.parse(approval.reviewedAt ?? "")) && approval.status === "approved" &&
    compatibility.recipeId === recipeId && compatibility.recipeRevision === recipeRevision &&
    (compatibility.appearance === appearance || compatibility.appearance === "any") &&
    cropValues.every(Number.isFinite) &&
    crop.focalPoint.x >= 0 && crop.focalPoint.x <= 1 && crop.focalPoint.y >= 0 && crop.focalPoint.y <= 1 &&
    crop.bounds.x >= 0 && crop.bounds.y >= 0 && crop.bounds.width > 0 && crop.bounds.height > 0 &&
    crop.bounds.x + crop.bounds.width <= 1 && crop.bounds.y + crop.bounds.height <= 1 &&
    typeof record.src === "string" && record.src.startsWith("/assets/meal-media/") && !/[\\%]/.test(record.src) && !record.src.includes(".."),
  );
}

export function selectMealMedia(records: readonly MealMediaRecord[], recipeId: string, recipeRevision: number, appearance: "light" | "evening") {
  return records.filter((record) => complete(record, recipeId, recipeRevision, appearance)).slice().sort((a, b) => a.id.localeCompare(b.id))[0];
}

type MealMediaFrameProps = {
  recipeId?: string;
  recipeRevision?: number;
  label: string;
  appearance?: "light" | "evening";
  className: string;
  fallbackText: string;
  detail?: string;
  records?: readonly MealMediaRecord[];
};

export function MealMediaFrame({ recipeId, recipeRevision, label, appearance = "light", className, fallbackText, detail, records = mealMediaInventory }: MealMediaFrameProps) {
  const record = recipeId && recipeRevision ? selectMealMedia(records, recipeId, recipeRevision, appearance) : undefined;
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [record?.id]);
  const showImage = Boolean(record && !failed);
  return <div className={`${className}${showImage ? " has-eligible-media" : ""}`} role="img" aria-label={showImage ? label : fallbackText}>
    {showImage && record ? <img src={record.src} alt="" style={{ objectPosition: `${record.crop.focalPoint.x * 100}% ${record.crop.focalPoint.y * 100}%` }} onError={() => setFailed(true)} /> : <><Leaf aria-hidden="true" /><span>{fallbackText}</span></>}
    {detail && <span className="meal-media-detail">{detail}</span>}
  </div>;
}
