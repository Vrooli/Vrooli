import { AlertTriangle, ArchiveRestore, CheckCircle2, CircleSlash, Construction } from 'lucide-react';
import type { IssueStatus } from '../data/sampleData';

export interface ColumnDefinition {
  title: string;
  icon: React.ComponentType<{ size?: number }>;
}

export const ISSUE_BOARD_COLUMNS: Record<IssueStatus, ColumnDefinition> = {
  open: { title: 'Open', icon: AlertTriangle },
  active: { title: 'Active', icon: Construction },
  completed: { title: 'Completed', icon: CheckCircle2 },
  failed: { title: 'Failed', icon: CircleSlash },
  archived: { title: 'Archived', icon: ArchiveRestore },
};

export const ISSUE_BOARD_STATUSES = Object.keys(ISSUE_BOARD_COLUMNS) as IssueStatus[];

