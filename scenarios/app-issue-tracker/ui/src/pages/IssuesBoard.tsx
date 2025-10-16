import { useMemo, useState, useRef, useEffect, useCallback } from 'react';
import type { DragEvent, TouchEvent } from 'react';
import { EyeOff } from 'lucide-react';
import { Issue, IssueStatus } from '../data/sampleData';
import { IssueCard } from '../components/IssueCard';
import { ISSUE_BOARD_COLUMNS, ISSUE_BOARD_STATUSES } from '../constants/board';

interface IssuesBoardProps {
  issues: Issue[];
  focusedIssueId?: string | null;
  runningProcesses?: Map<string, { agent_id: string; start_time: string; duration?: string }>;
  onIssueSelect?: (issueId: string) => void;
  onIssueDelete?: (issue: Issue) => void;
  onIssueDrop?: (issueId: string, fromStatus: IssueStatus, toStatus: IssueStatus) => void | Promise<void>;
  onIssueArchive?: (issue: Issue) => void;
  onStopAgent?: (issueId: string) => void;
  hiddenColumns?: IssueStatus[];
  onHideColumn?: (status: IssueStatus) => void;
}

export function IssuesBoard({
  issues,
  focusedIssueId,
  runningProcesses,
  onIssueSelect,
  onIssueDelete,
  onIssueDrop,
  onIssueArchive,
  onStopAgent,
  hiddenColumns,
  onHideColumn,
}: IssuesBoardProps) {
  const [dragState, setDragState] = useState<{ issueId: string; from: IssueStatus } | null>(null);
  const [dragOverStatus, setDragOverStatus] = useState<IssueStatus | null>(null);
  const [pointerDragState, setPointerDragState] = useState<{
    pointerId: number;
    issueId: string;
    fromStatus: IssueStatus;
    startX: number;
    startY: number;
    offsetX: number;
    offsetY: number;
    ghost: HTMLElement | null;
    dragging: boolean;
  } | null>(null);
  const kanbanGridRef = useRef<HTMLDivElement>(null);
  const scrollLockRef = useRef<{ timeout: number | null }>({
    timeout: null,
  });
  const autoScrollRef = useRef<{ animationId: number | null; speed: number }>({
    animationId: null,
    speed: 0,
  });
  const POINTER_DRAG_THRESHOLD = 12;
  const AUTO_SCROLL_THRESHOLD = 80; // pixels from edge to trigger auto-scroll
  const AUTO_SCROLL_SPEED_MAX = 15; // max pixels per frame

  // Attach native wheel listener with { passive: false }
  useEffect(() => {
    const kanbanGrid = kanbanGridRef.current;
    if (!kanbanGrid) return;

    const handleWheelNative = (event: globalThis.WheelEvent) => {
      if (event.defaultPrevented) {
        return;
      }

      const scrollLock = scrollLockRef.current;
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

    kanbanGrid.addEventListener('wheel', handleWheelNative, { passive: false });
    return () => kanbanGrid.removeEventListener('wheel', handleWheelNative);
  }, []);

  const grouped = useMemo(() => {
    const base = ISSUE_BOARD_STATUSES.reduce((acc, status) => {
      acc[status] = [];
      return acc;
    }, {} as Record<IssueStatus, Issue[]>);

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
    () => ISSUE_BOARD_STATUSES.filter((status) => !hiddenSet.has(status)),
    [hiddenSet],
  );

  // Start auto-scroll (like ecosystem-manager)
  const startAutoScroll = useCallback((scrollDirection: number) => {
    const autoScroll = autoScrollRef.current;
    autoScroll.speed = scrollDirection;

    if (autoScroll.animationId !== null) {
      return; // Already running
    }

    const step = () => {
      const { speed } = autoScrollRef.current;
      if (speed === 0) {
        autoScrollRef.current.animationId = null;
        return;
      }

      const kanbanGrid = kanbanGridRef.current;
      if (kanbanGrid) {
        kanbanGrid.scrollLeft += speed;
      }

      autoScrollRef.current.animationId = requestAnimationFrame(step);
    };

    autoScroll.animationId = requestAnimationFrame(step);
  }, []);

  // Stop auto-scroll
  const stopAutoScroll = useCallback(() => {
    const autoScroll = autoScrollRef.current;
    autoScroll.speed = 0;

    if (autoScroll.animationId !== null) {
      cancelAnimationFrame(autoScroll.animationId);
      autoScroll.animationId = null;
    }
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (scrollLockRef.current.timeout !== null) {
        clearTimeout(scrollLockRef.current.timeout);
      }
      if (autoScrollRef.current.animationId !== null) {
        cancelAnimationFrame(autoScrollRef.current.animationId);
      }
    };
  }, []);

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

  const handlePointerCardDown = (issue: Issue, event: React.PointerEvent<HTMLDivElement>, card: HTMLElement) => {
    const rect = card.getBoundingClientRect();
    const offsetX = event.clientX - rect.left;
    const offsetY = event.clientY - rect.top;

    const state = {
      pointerId: event.pointerId,
      issueId: issue.id,
      fromStatus: issue.status,
      startX: event.clientX,
      startY: event.clientY,
      offsetX,
      offsetY,
      ghost: null,
      dragging: false,
      card,
    };

    setPointerDragState(state as any);

    card.setPointerCapture?.(event.pointerId);

    // Use document listeners like ecosystem-manager
    const handleDocumentPointerMove = (e: PointerEvent) => handlePointerMoveDocument(e, state as any);
    const handleDocumentPointerUp = (e: PointerEvent) => handlePointerUpDocument(e, state as any);

    document.addEventListener('pointermove', handleDocumentPointerMove, { passive: false });
    document.addEventListener('pointerup', handleDocumentPointerUp, { passive: false });
    document.addEventListener('pointercancel', handleDocumentPointerUp, { passive: false });

    // Store cleanup function
    (state as any).cleanup = () => {
      document.removeEventListener('pointermove', handleDocumentPointerMove);
      document.removeEventListener('pointerup', handleDocumentPointerUp);
      document.removeEventListener('pointercancel', handleDocumentPointerUp);
    };
  };

  const handlePointerMoveDocument = (e: PointerEvent, state: any) => {
    if (!state || e.pointerId !== state.pointerId) {
      return;
    }

    const deltaX = Math.abs(e.clientX - state.startX);
    const deltaY = Math.abs(e.clientY - state.startY);

    if (!state.dragging) {
      if (deltaX <= POINTER_DRAG_THRESHOLD && deltaY <= POINTER_DRAG_THRESHOLD) {
        return;
      }

      // Activate drag
      e.preventDefault();
      const { card, offsetX, offsetY } = state;
      const rect = card.getBoundingClientRect();
      const ghost = card.cloneNode(true) as HTMLElement;
      ghost.removeAttribute('id');
      ghost.style.width = `${rect.width}px`;
      ghost.style.height = `${rect.height}px`;
      ghost.style.opacity = '0.9';
      ghost.style.position = 'fixed';
      ghost.style.zIndex = '1000';
      ghost.style.pointerEvents = 'none';
      ghost.classList.add('touch-drag-ghost');
      ghost.style.transition = 'none';

      const x = e.clientX - offsetX;
      const y = e.clientY - offsetY;
      ghost.style.transform = `translate3d(${x}px, ${y}px, 0)`;

      document.body.appendChild(ghost);
      card.classList.add('dragging');

      state.ghost = ghost;
      state.dragging = true;
      setPointerDragState({ ...state });
      return;
    }

    if (!state.dragging) {
      return;
    }

    e.preventDefault();

    // Update ghost position
    if (state.ghost) {
      const x = e.clientX - state.offsetX;
      const y = e.clientY - state.offsetY;
      state.ghost.style.transform = `translate3d(${x}px, ${y}px, 0)`;
    }

    // Auto-scroll near VIEWPORT edges (like ecosystem-manager)
    const viewportWidth = window.innerWidth || document.documentElement.clientWidth;

    let horizontalScroll = 0;
    if (e.clientX < AUTO_SCROLL_THRESHOLD) {
      horizontalScroll = -1;
    } else if (e.clientX > viewportWidth - AUTO_SCROLL_THRESHOLD) {
      horizontalScroll = 1;
    }

    if (horizontalScroll === 0) {
      stopAutoScroll();
    } else {
      startAutoScroll(horizontalScroll * 18); // Direct speed like ecosystem-manager
    }

    // Highlight column
    const element = document.elementFromPoint(e.clientX, e.clientY);
    const column = element?.closest('.kanban-column');

    document.querySelectorAll('.kanban-column').forEach((col) => {
      col.classList.remove('kanban-column--drag-over');
    });

    if (column) {
      const status = column.getAttribute('data-status') as IssueStatus | null;
      column.classList.add('kanban-column--drag-over');
      setDragOverStatus(status);
    } else {
      setDragOverStatus(null);
    }
  };

  const handlePointerUpDocument = async (e: PointerEvent, state: any) => {
    if (!state || e.pointerId !== state.pointerId) {
      return;
    }

    const wasDragging = state.dragging;

    if (wasDragging) {
      e.preventDefault();
      // Prevent click event after drag
      setTimeout(() => {
        state.card.dataset.preventClick = 'true';
        setTimeout(() => {
          delete state.card.dataset.preventClick;
        }, 100);
      }, 0);
    }

    const { card, issueId, fromStatus, ghost } = state;

    // Cleanup
    card.classList.remove('dragging');
    card.releasePointerCapture?.(e.pointerId);

    if (ghost && ghost.parentElement) {
      ghost.parentElement.removeChild(ghost);
    }

    // Clear highlights and stop scroll (like ecosystem-manager)
    document.querySelectorAll('.kanban-column').forEach((col) => {
      col.classList.remove('kanban-column--drag-over');
    });
    stopAutoScroll();

    if (wasDragging) {
      const dropTarget = document.elementFromPoint(e.clientX, e.clientY);
      const column = dropTarget?.closest('.kanban-column');

      if (column) {
        const targetStatus = column.getAttribute('data-status') as IssueStatus | null;
        if (targetStatus && targetStatus !== fromStatus && onIssueDrop) {
          await onIssueDrop(issueId, fromStatus, targetStatus);
        }
      }
    }

    // Remove document listeners
    if (state.cleanup) {
      state.cleanup();
    }

    setPointerDragState(null);
    setDragOverStatus(null);
  };

  return (
    <div className="issues-board">
      <div className="kanban-grid" data-columns={visibleStatuses.length} ref={kanbanGridRef}>
        {visibleStatuses.length === 0 ? (
          <div className="kanban-empty-state">
            <p>All columns are hidden. Use the column controls to show them again.</p>
          </div>
        ) : (
          visibleStatuses.map((status) => {
            const column = ISSUE_BOARD_COLUMNS[status];
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
                data-status={status}
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
                  {columnIssues.map((issue) => {
                    const runningInfo = runningProcesses?.get(issue.id);
                    return (
                      <IssueCard
                        key={issue.id}
                        issue={issue}
                        isFocused={issue.id === focusedIssueId}
                        runningProcess={runningInfo}
                        onSelect={onIssueSelect}
                        onDelete={confirmDelete}
                        onArchive={confirmArchive}
                        onStopAgent={onStopAgent}
                        onDragStart={handleCardDragStart}
                        onDragEnd={handleCardDragEnd}
                        onPointerDown={handlePointerCardDown}
                        draggable
                      />
                    );
                  })}
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
