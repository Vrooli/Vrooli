import type { Plugin } from "vite";

export interface AssetStampMetadata {
  asset?: string;
  version?: string;
  componentName?: string;
  exempt?: boolean;
  reason?: string;
  relativePath: string;
}

export function stampSource(
  source: string,
  input: { asset: string; version: string; componentName: string },
): { changed: boolean; code: string; reason: string };
export function assetStampMetadata(
  id: string,
  scenarioRoot: string,
  exemptionFile?: string,
): AssetStampMetadata | undefined;
export default function assetStampPlugin(options?: {
  scenarioRoot?: string;
  exemptionFile?: string;
}): Plugin;
