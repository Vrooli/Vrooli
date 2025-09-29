import { AlertTriangle, Circle, OctagonX } from 'lucide-react';
import type { IssueRecord } from '../types';

interface IssueCardProps {
  issue: IssueRecord;
}

const SEVERITY_META: Record<IssueRecord['severity'], { label: string; tone: 'critical' | 'high' | 'medium' | 'low'; Icon: typeof AlertTriangle }> = {
  critical: { label: 'Critical', tone: 'critical', Icon: OctagonX },
  high: { label: 'High', tone: 'high', Icon: AlertTriangle },
  medium: { label: 'Medium', tone: 'medium', Icon: Circle },
  low: { label: 'Low', tone: 'low', Icon: Circle }
};

export function IssueCard({ issue }: IssueCardProps) {
  const meta = SEVERITY_META[issue.severity];
  const { Icon } = meta;
  const firstSeen = issue.first_seen ? new Date(issue.first_seen).toLocaleString() : 'Unknown';

  return (
    <article className={`issue-card ${meta.tone}`}>
      <header>
        <span className="issue-severity" aria-label={`Severity ${meta.label}`}>
          <Icon size={16} aria-hidden="true" />
          {meta.label}
        </span>
        <span className="issue-id">#{issue.id}</span>
      </header>
      <div className="issue-body">
        <h3>{issue.component}</h3>
        <p>{issue.description}</p>
      </div>
      <footer>
        <span>First seen {firstSeen}</span>
        {issue.status ? <span className="issue-status">Status: {issue.status}</span> : null}
      </footer>
    </article>
  );
}
