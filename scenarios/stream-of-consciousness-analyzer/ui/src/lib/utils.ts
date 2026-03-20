import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

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
