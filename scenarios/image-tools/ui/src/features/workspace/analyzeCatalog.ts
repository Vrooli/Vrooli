import {
  Copy,
  FileSearch,
  Gauge,
  ScanSearch,
  ScanText,
  Image,
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
 * Maps discovered analysis operations plus the durable AI `caption` operation
 * to human labels and icons. Unknown operations still render through the
 * fallback, provided the API result parser supports them.
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
  caption: {
    labelKey: strings.workspace.analyzeOp.caption.label,
    descKey: strings.workspace.analyzeOp.caption.desc,
    Icon: Image,
  },
};

/** Fallback icon for an analysis op the catalog doesn't yet name. */
export const ANALYZE_FALLBACK_ICON: LucideIcon = ScanSearch;

/** Presentation for a known analysis op, or `undefined` if it isn't catalogued. */
export const analyzePresentation = (operation: string): AnalyzeOpPresentation | undefined =>
  ANALYZE_CATALOG[operation];
