import {
  Copy,
  FileSearch,
  Gauge,
  ScanSearch,
  ScanText,
  ShieldAlert,
  type LucideIcon,
} from "lucide-react";

import { strings } from "../../consts/strings";

type AnalyzeOpStrings = typeof strings.workspace.analyzeOp;
type AnalyzeOpEntry = AnalyzeOpStrings[keyof AnalyzeOpStrings];

/** Per-analysis-op presentation: friendly label/description keys and an icon. */
export interface AnalyzeOpPresentation {
  labelKey: AnalyzeOpEntry["label"];
  descKey: AnalyzeOpEntry["desc"];
  Icon: LucideIcon;
}

/**
 * Maps each analysis op (discovered from `AnalysisService.ListAnalysisOperations`)
 * to its humanized label/description keys and a lucide icon. Driven by the live
 * catalog, so a backend that adds an analysis op still renders (via the
 * fallback) without a code change here. The three shipped ops are the pure-Go
 * `probe`, the model-backed `ocr`, and the model-backed `nsfw_classify`.
 */
export const ANALYZE_CATALOG: Readonly<Record<string, AnalyzeOpPresentation>> = {
  probe: {
    labelKey: strings.workspace.analyzeOp.probe.label,
    descKey: strings.workspace.analyzeOp.probe.desc,
    Icon: FileSearch,
  },
  ocr: {
    labelKey: strings.workspace.analyzeOp.ocr.label,
    descKey: strings.workspace.analyzeOp.ocr.desc,
    Icon: ScanText,
  },
  nsfw_classify: {
    labelKey: strings.workspace.analyzeOp.nsfw_classify.label,
    descKey: strings.workspace.analyzeOp.nsfw_classify.desc,
    Icon: ShieldAlert,
  },
  duplicate_detect: {
    labelKey: strings.workspace.analyzeOp.duplicate_detect.label,
    descKey: strings.workspace.analyzeOp.duplicate_detect.desc,
    Icon: Copy,
  },
  quality_assessment: {
    labelKey: strings.workspace.analyzeOp.quality_assessment.label,
    descKey: strings.workspace.analyzeOp.quality_assessment.desc,
    Icon: Gauge,
  },
};

/** Fallback icon for an analysis op the catalog doesn't yet name. */
export const ANALYZE_FALLBACK_ICON: LucideIcon = ScanSearch;

/** Presentation for a known analysis op, or `undefined` if it isn't catalogued. */
export const analyzePresentation = (operation: string): AnalyzeOpPresentation | undefined =>
  ANALYZE_CATALOG[operation];
