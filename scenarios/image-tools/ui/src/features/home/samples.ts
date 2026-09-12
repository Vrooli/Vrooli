import blurryUrl from "../../assets/samples/sample-blurry.png";
import portraitUrl from "../../assets/samples/sample-portrait.png";
import productUrl from "../../assets/samples/sample-product.png";
import receiptUrl from "../../assets/samples/sample-receipt.png";
import { strings } from "../../consts/strings";
import type { WorkspaceMode } from "../workspace/useWorkspace";

export type SampleKey = "product" | "portrait" | "blurry" | "receipt";

export interface SampleImage {
  key: SampleKey;
  /** Bundled asset URL (Vite-resolved, proxy/route-safe). */
  url: string;
  fileName: string;
  labelKey: (typeof strings.home.samples)[SampleKey];
  /** Mode the Workspace opens in when this sample is loaded. */
  mode: WorkspaceMode;
}

/**
 * The first-run / default sample: a product on a busy background, so the very
 * first tap can be "Remove background". Declared standalone (not `SAMPLES[0]`)
 * so callers always have a non-undefined default under noUncheckedIndexedAccess.
 */
export const DEFAULT_SAMPLE: SampleImage = {
  key: "product",
  url: productUrl,
  fileName: "sample-product.png",
  labelKey: strings.home.samples.product,
  mode: "enhance",
};

/** Curated demo set — each chosen to make one hero flow sing. */
export const SAMPLES: readonly SampleImage[] = [
  DEFAULT_SAMPLE,
  {
    key: "portrait",
    url: portraitUrl,
    fileName: "sample-portrait.png",
    labelKey: strings.home.samples.portrait,
    mode: "enhance",
  },
  {
    key: "blurry",
    url: blurryUrl,
    fileName: "sample-blurry.png",
    labelKey: strings.home.samples.blurry,
    mode: "enhance",
  },
  {
    key: "receipt",
    url: receiptUrl,
    fileName: "sample-receipt.png",
    labelKey: strings.home.samples.receipt,
    mode: "analyze",
  },
];

/** Fetch a sample's bytes and wrap them as a File the Workspace can adopt. */
export async function loadSampleFile(sample: SampleImage): Promise<File> {
  const res = await fetch(sample.url);
  const blob = await res.blob();
  return new File([blob], sample.fileName, { type: blob.type || "image/png" });
}
