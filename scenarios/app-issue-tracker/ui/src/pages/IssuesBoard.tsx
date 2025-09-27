import { useMemo } from 'react';
import { AlertTriangle, Archive, CheckCircle2, CircleSlash, Construction, Search } from 'lucide-react';
import { Issue, IssueStatus } from '../data/sampleData';
import { IssueCard } from '../components/IssueCard';

interface IssuesBoardProps {
  issues: Issue[];
}

const columnMeta: Record<IssueStatus, { title: string; icon: React.ComponentType<{ size?: number }> }> = {
  open: { title: 'Open', icon: AlertTriangle },
  investigating: { title: 'Investigating', icon: Search },
  'in-progress': { title: 'In Progress', icon: Construction },
  fixed: { title: 'Fixed', icon: CheckCircle2 },
  closed: { title: 'Closed', icon: Archive },
  failed: { title: 'Failed', icon: CircleSlash },
};

export function IssuesBoard({ issues }: IssuesBoardProps) {
  const grouped = useMemo(() => {
    const base: Record<IssueStatus, Issue[]> = {
      open: [],
      investigating: [],
      'in-progress': [],
      fixed: [],
      closed: [],
      failed: [],
    };

    issues.forEach((issue) => {
      base[issue.status] = [...base[issue.status], issue];
    });

    (Object.keys(base) as IssueStatus[]).forEach((status) => {
      base[status].sort(
        (first, second) => new Date(second.createdAt).getTime() - new Date(first.createdAt).getTime(),
      );
    });

    return base;
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
