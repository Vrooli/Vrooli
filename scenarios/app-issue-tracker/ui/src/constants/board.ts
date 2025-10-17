import { AlertTriangle, ArchiveRestore, CheckCircle2, CircleSlash, Clock, Construction } from 'lucide-react';
import type { IssueStatus } from '../data/sampleData';
import { formatStatusLabel } from '../utils/issues';

export interface ColumnDefinition {
  title: string;
  icon: React.ComponentType<{ size?: number }>;
}

const BASE_COLUMNS: Record<string, ColumnDefinition> = {
  open: { title: 'Open', icon: AlertTriangle },
  active: { title: 'Active', icon: Construction },
  waiting: { title: 'Waiting', icon: Clock },
  completed: { title: 'Completed', icon: CheckCircle2 },
  failed: { title: 'Failed', icon: CircleSlash },
  archived: { title: 'Archived', icon: ArchiveRestore },
};

export function getIssueStatusColumn(status: IssueStatus): ColumnDefinition {
  const key = status.toLowerCase();
  const meta = BASE_COLUMNS[key];
  if (meta) {
    return meta;
  }
  return {
    title: formatStatusLabel(status),
    icon: AlertTriangle,
  } satisfies ColumnDefinition;
}

export const DEFAULT_BOARD_STATUS_ORDER: IssueStatus[] = ['open', 'active', 'waiting', 'completed', 'failed', 'archived'];
