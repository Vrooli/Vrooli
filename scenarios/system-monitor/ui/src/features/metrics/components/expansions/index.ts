import type { ComponentType } from 'react';
import type {
  CardType,
  CPUMetrics,
  MemoryMetrics,
  NetworkMetrics,
  DiskCardDetails,
  GPUCardDetails
} from '../../../../types';
import { CpuExpansion } from './CpuExpansion';
import { MemoryExpansion } from './MemoryExpansion';
import { DiskExpansion } from './DiskExpansion';
import { GpuExpansion } from './GpuExpansion';
import { NetworkExpansion } from './NetworkExpansion';

export { CpuExpansion } from './CpuExpansion';
export { MemoryExpansion } from './MemoryExpansion';
export { DiskExpansion } from './DiskExpansion';
export { GpuExpansion } from './GpuExpansion';
export { NetworkExpansion } from './NetworkExpansion';

type ExpansionDetails = CPUMetrics | MemoryMetrics | NetworkMetrics | DiskCardDetails | GPUCardDetails;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const expansionMap: Partial<Record<CardType, ComponentType<{ details: any }>>> = {
  cpu: CpuExpansion as ComponentType<{ details: ExpansionDetails }>,
  memory: MemoryExpansion as ComponentType<{ details: ExpansionDetails }>,
  disk: DiskExpansion as ComponentType<{ details: ExpansionDetails }>,
  gpu: GpuExpansion as ComponentType<{ details: ExpansionDetails }>,
  network: NetworkExpansion as ComponentType<{ details: ExpansionDetails }>
};
