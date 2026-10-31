/**
 * @vrooliComponentSource react-component-library:ClassMerge
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 45c12833-e435-43d6-87c4-ada73b54fdbf
 * @vrooliComponentAppliedAt 2026-08-25T09:55:45Z
 * @vrooliComponentSourceSha256 14a4aa225807e251efd272be0eefb285b157a484d36dd5005fab2b40e5163512
 * @vrooliComponentDriftHash 45646cbca6c11c407766266e5769f4c0b3ebfcf7a376fabe9132ff7ded766c5f
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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
