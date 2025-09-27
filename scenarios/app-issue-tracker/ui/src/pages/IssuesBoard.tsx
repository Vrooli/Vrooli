import { useMemo } from 'react';
import { AlertTriangle, CheckCircle2, CircleSlash, Construction } from 'lucide-react';
import { Issue, IssueStatus } from '../data/sampleData';
import { IssueCard } from '../components/IssueCard';

interface IssuesBoardProps {
  issues: Issue[];
}

const columnMeta: Record<IssueStatus, { title: string; icon: React.ComponentType<{ size?: number }> }> = {
  pending: { title: 'Pending', icon: AlertTriangle },
  active: { title: 'Active', icon: Construction },
  complete: { title: 'Complete', icon: CheckCircle2 },
  failed: { title: 'Failed', icon: CircleSlash },
};

export function IssuesBoard({ issues }: IssuesBoardProps) {
  const grouped = useMemo(() => {
    return issues.reduce<Record<IssueStatus, Issue[]>>(
      (acc, issue) => {
        acc[issue.status] = [...acc[issue.status], issue];
        return acc;
      },
      { pending: [], active: [], complete: [], failed: [] },
    );
  }, [issues]);

  return (
    <div className="issues-board">
      <header className="page-header">
        <div>
          <h2>Kanban</h2>
          <p>Track issues as they move through investigation and resolution.</p>
        </div>
      </header>
      <div className="kanban-grid">
        {(Object.keys(columnMeta) as IssueStatus[]).map((status) => {
          const column = columnMeta[status];
          const ColumnIcon = column.icon;
          return (
            <section key={status} className="kanban-column">
              <header>
                <ColumnIcon size={16} />
                <h3>{column.title}</h3>
                <span className="column-count">{grouped[status].length}</span>
              </header>
              <div className="kanban-column-body">
                {grouped[status].map((issue) => (
                  <IssueCard key={issue.id} issue={issue} />
                ))}
                {grouped[status].length === 0 && (
                  <div className="empty-column">
                    <p>No issues in this state.</p>
                  </div>
                )}
              </div>
            </section>
          );
        })}
      </div>
    </div>
  );
}
