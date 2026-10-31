/**
 * Shared section components barrel exports.
 */

export { SectionCard } from "./SectionCard";
export { SectionHeader } from "./SectionHeader";
export { StageStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
export {
  type StatusDisplayConfig,
  STAGE_STATUS_CONFIG as STATUS_CONFIG,
  getStageStatusDisplay as getStatusDisplay,
} from "../../../lib/status-display";
export {
  StageStatusOverview,
  StagePlaceholder,
  StageError,
  StageAbout,
  StageDetailCard,
  StageWarning,
} from "./StageComponents";
