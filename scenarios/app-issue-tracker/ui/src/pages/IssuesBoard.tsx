import { useMemo, useState, useRef, useEffect } from 'react';
import type { DragEvent, WheelEvent } from 'react';
import { AlertTriangle, ArchiveRestore, CheckCircle2, CircleSlash, Clock, Construction, EyeOff } from 'lucide-react';
import { Issue, IssueStatus } from '../data/sampleData';
import { IssueCard } from '../components/IssueCard';

interface IssuesBoardProps {
  issues: Issue[];
  focusedIssueId?: string | null;
  onIssueSelect?: (issueId: string) => void;
  onIssueDelete?: (issue: Issue) => void;
  onIssueDrop?: (issueId: string, fromStatus: IssueStatus, toStatus: IssueStatus) => void | Promise<void>;
  onIssueArchive?: (issue: Issue) => void;
  hiddenColumns?: IssueStatus[];
  onHideColumn?: (status: IssueStatus) => void;
}

export const columnMeta: Record<IssueStatus, { title: string; icon: React.ComponentType<{ size?: number }> }> = {
  open: { title: 'Open', icon: AlertTriangle },
  active: { title: 'Active', icon: Construction },
  waiting: { title: 'Waiting', icon: Clock },
  completed: { title: 'Completed', icon: CheckCircle2 },
  failed: { title: 'Failed', icon: CircleSlash },
  archived: { title: 'Archived', icon: ArchiveRestore },
};

export function IssuesBoard({
  issues,
  focusedIssueId,
  onIssueSelect,
  onIssueDelete,
  onIssueDrop,
  onIssueArchive,
  hiddenColumns,
  onHideColumn,
}: IssuesBoardProps) {
  const [dragState, setDragState] = useState<{ issueId: string; from: IssueStatus } | null>(null);
  const [dragOverStatus, setDragOverStatus] = useState<IssueStatus | null>(null);
  const kanbanGridRef = useRef<HTMLDivElement>(null);
  const scrollLockRef = useRef<{ timeout: number | null }>({
    timeout: null,
  });

  const grouped = useMemo(() => {
    const base: Record<IssueStatus, Issue[]> = {
      open: [],
      active: [],
      waiting: [],
      completed: [],
      failed: [],
      archived: [],
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

  const hiddenSet = useMemo(() => new Set(hiddenColumns ?? []), [hiddenColumns]);
  const visibleStatuses = useMemo(
    () => (Object.keys(columnMeta) as IssueStatus[]).filter((status) => !hiddenSet.has(status)),
    [hiddenSet],
  );

  // Cleanup scroll lock timeout on unmount
  useEffect(() => {
    return () => {
      if (scrollLockRef.current.timeout !== null) {
        clearTimeout(scrollLockRef.current.timeout);
      }
    };
  }, []);

  const handleWheel = (event: WheelEvent<HTMLDivElement>) => {
    if (event.defaultPrevented) {
      return;
    }

    const scrollLock = scrollLockRef.current;
    const now = Date.now();
    const horizontalLockActive = scrollLock.timeout !== null;

    // Check if we're scrolling within a column
    const columnBody = (event.target as HTMLElement).closest('.kanban-column-body');
    if (columnBody && !horizontalLockActive) {
      const canScrollVertically = columnBody.scrollHeight > columnBody.clientHeight;
      if (canScrollVertically) {
        const scrollTop = columnBody.scrollTop;
        const atTop = scrollTop <= 0;
        const atBottom = (columnBody.scrollHeight - columnBody.clientHeight - scrollTop) <= 1;

        // Allow vertical scrolling within column if not at boundaries
        if ((event.deltaY < 0 && !atTop) || (event.deltaY > 0 && !atBottom)) {
          return;
        }
      }
    }

    // Convert vertical scroll to horizontal scroll
    if (Math.abs(event.deltaY) > Math.abs(event.deltaX)) {
      event.preventDefault();
      const kanbanGrid = kanbanGridRef.current;
      if (kanbanGrid) {
        kanbanGrid.scrollLeft += event.deltaY;
      }

      // Set horizontal scroll lock for 250ms
      if (scrollLock.timeout !== null) {
        clearTimeout(scrollLock.timeout);
      }
      scrollLock.timeout = window.setTimeout(() => {
        scrollLock.timeout = null;
      }, 250);
    }
  };

  const confirmDelete = (issue: Issue) => {
    if (!onIssueDelete) {
      return;
    }
    if (typeof window !== 'undefined' && typeof window.confirm === 'function') {
      const confirmed = window.confirm(
        `Delete ${issue.id}${issue.title ? ` — ${issue.title}` : ''}? This cannot be undone.`,
      );
      if (!confirmed) {
        return;
      }
    }
    onIssueDelete(issue);
  };

  const confirmArchive = (issue: Issue) => {
    if (!onIssueArchive) {
      return;
    }
    let shouldArchive = true;
    if (typeof window !== 'undefined' && typeof window.confirm === 'function') {
      shouldArchive = window.confirm(`Archive ${issue.id}? The issue will move to the Archived column.`);
    }
    if (!shouldArchive) {
      return;
    }
    onIssueArchive(issue);
  };

  const handleCardDragStart = (issue: Issue, event: DragEvent<HTMLDivElement>) => {
    event.dataTransfer.effectAllowed = 'move';
    try {
      event.dataTransfer.setData('text/plain', issue.id);
    } catch {
      // Ignore browsers that disallow custom data on drag start
    }
    setDragState({ issueId: issue.id, from: issue.status });
    setDragOverStatus(null);
  };

  const handleCardDragEnd = () => {
    setDragState(null);
    setDragOverStatus(null);
  };

  const handleColumnDragOver = (event: DragEvent<HTMLElement>, status: IssueStatus) => {
    if (!dragState) {
      return;
    }
    event.preventDefault();
    if (dragState.from === status) {
      event.dataTransfer.dropEffect = 'none';
      setDragOverStatus(null);
      return;
    }
    event.dataTransfer.dropEffect = 'move';
    if (dragOverStatus !== status) {
      setDragOverStatus(status);
    }
  };

  const handleColumnDragLeave = (event: DragEvent<HTMLElement>, status: IssueStatus) => {
    const nextTarget = event.relatedTarget as Node | null;
    if (nextTarget && (event.currentTarget as HTMLElement).contains(nextTarget)) {
      return;
    }
    if (dragOverStatus === status) {
      setDragOverStatus(null);
    }
  };

  const handleColumnDrop = async (event: DragEvent<HTMLElement>, status: IssueStatus) => {
    event.preventDefault();
    const currentDrag = dragState;
    setDragState(null);
    setDragOverStatus(null);
    if (!currentDrag || currentDrag.from === status) {
      return;
    }
    await onIssueDrop?.(currentDrag.issueId, currentDrag.from, status);
  };

  return (
    <div className="issues-board">
      <div className="kanban-grid" data-columns={visibleStatuses.length} ref={kanbanGridRef} onWheel={handleWheel}>
        {visibleStatuses.length === 0 ? (
          <div className="kanban-empty-state">
            <p>All columns are hidden. Use the column controls to show them again.</p>
          </div>
        ) : (
          visibleStatuses.map((status) => {
            const column = columnMeta[status];
            const ColumnIcon = column.icon;
            const columnIssues = grouped[status] ?? [];
            const columnClasses = ['kanban-column'];
            if (dragOverStatus === status) {
              columnClasses.push('kanban-column--drag-over');
            }

            return (
              <section
                key={status}
                className={columnClasses.join(' ')}
                onDragOver={(event) => handleColumnDragOver(event, status)}
                onDragEnter={(event) => handleColumnDragOver(event, status)}
                onDragLeave={(event) => handleColumnDragLeave(event, status)}
                onDrop={(event) => void handleColumnDrop(event, status)}
              >
                <header>
                  <div className="kanban-column-heading">
                    <ColumnIcon size={14} />
                    <h3>{column.title}</h3>
                    <span className="column-count">{columnIssues.length}</span>
                  </div>
                  {onHideColumn && (
                    <button
                      type="button"
                      className="column-hide-btn"
                      onClick={() => onHideColumn(status)}
                      aria-label={`Hide ${column.title} column`}
                    >
                      <EyeOff size={13} />
                    </button>
                  )}
                </header>
                <div className="kanban-column-body">
                  {columnIssues.map((issue) => (
                    <IssueCard
                      key={issue.id}
                      issue={issue}
                      isFocused={issue.id === focusedIssueId}
                      onSelect={onIssueSelect}
                      onDelete={confirmDelete}
                      onArchive={confirmArchive}
                      onDragStart={handleCardDragStart}
                      onDragEnd={handleCardDragEnd}
                      draggable
                    />
                  ))}
                  {columnIssues.length === 0 && (
                    <div className="empty-column">
                      <p>No issues in this state.</p>
                    </div>
                  )}
                </div>
              </section>
            );
          })
        )}
      </div>
    </div>
  );
}
