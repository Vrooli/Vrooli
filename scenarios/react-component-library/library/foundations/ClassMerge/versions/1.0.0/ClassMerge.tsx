/**
 * @vrooliComponentSource foundations.class-merge
 * @version 1.0.0
 * @status released
 * @deps {"clsx":"^2.1.0","tailwind-merge":"^2.2.0"}
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Combines conditional class values and resolves Tailwind conflicts so the
 * consumer's explicit override wins over a component default.
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

export type { ClassValue };
