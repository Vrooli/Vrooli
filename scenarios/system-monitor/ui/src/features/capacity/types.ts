import type { MessageShape } from '@bufbuild/protobuf';
import type {
  GetCapacityOverviewResponseSchema,
  ReconcileCapacityResponseSchema,
} from '../../shared/api/proto-contracts';
import type {
  CapacityClaimSchema,
  GpuCapacitySchema,
  CapacityFindingSchema,
  PolicyLeverSchema,
} from '@vrooli/proto-types/system-monitor/v1/capacity/capacity_pb';

export type CapacityOverview = MessageShape<typeof GetCapacityOverviewResponseSchema>;
export type CapacityReconciliation = MessageShape<typeof ReconcileCapacityResponseSchema>;

export type CapacityClaim = MessageShape<typeof CapacityClaimSchema>;
export type GpuCapacity = MessageShape<typeof GpuCapacitySchema>;
export type CapacityFinding = MessageShape<typeof CapacityFindingSchema>;
export type PolicyLever = MessageShape<typeof PolicyLeverSchema>;
