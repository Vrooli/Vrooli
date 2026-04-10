import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ThoughtEdge } from "./types";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Generate a random canvas position within a bounded area.
 * Used by TextCapture and GraphView to place new items without overlapping the origin.
 */
export function randomCanvasPosition(width: number, height: number): { x: number; y: number } {
  return { x: Math.random() * width, y: Math.random() * height };
}

/**
 * Deduplicate edges collected from multiple thoughts.
 * When fetching edges per-thought, the same edge appears for both source and target.
 */
export function deduplicateEdges(edgeSets: ThoughtEdge[][]): ThoughtEdge[] {
  const seen = new Set<string>();
  const unique: ThoughtEdge[] = [];
  for (const edges of edgeSets) {
    for (const e of edges) {
      if (!seen.has(e.id)) {
        seen.add(e.id);
        unique.push(e);
      }
    }
  }
  return unique;
}
