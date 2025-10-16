import { useMemo } from 'react';
import { ChevronDown, Eye, EyeOff, Filter, X } from 'lucide-react';
import type { IssueStatus, Priority } from '../data/sampleData';
import { Modal } from './Modal';
import { ISSUE_BOARD_COLUMNS, ISSUE_BOARD_STATUSES } from '../constants/board';
import type { PriorityFilterValue } from '../types/issueCreation';

export interface IssueBoardToolbarProps {
  open: boolean;
  onRequestClose: () => void;
  priorityFilter: PriorityFilterValue;
  onPriorityFilterChange: (value: PriorityFilterValue) => void;
  appFilter: string;
  onAppFilterChange: (value: string) => void;
  searchFilter: string;
  onSearchFilterChange: (value: string) => void;
  appOptions: string[];
  hiddenColumns: IssueStatus[];
  onToggleColumn: (status: IssueStatus) => void;
  onResetColumns: () => void;
}

export function IssueBoardToolbar({
  open,
  onRequestClose,
  priorityFilter,
  onPriorityFilterChange,
  appFilter,
  onAppFilterChange,
  searchFilter,
  onSearchFilterChange,
  appOptions,
  hiddenColumns,
  onToggleColumn,
  onResetColumns,
}: IssueBoardToolbarProps) {
  const hiddenSet = useMemo(() => new Set(hiddenColumns), [hiddenColumns]);
  const statusOrder = useMemo(() => ISSUE_BOARD_STATUSES, []);

  const handleClearFilters = () => {
    onPriorityFilterChange('all');
    onAppFilterChange('all');
    onSearchFilterChange('');
  };

  if (!open) {
    return null;
  }

  return (
    <Modal
      onClose={onRequestClose}
      labelledBy="issue-filters-heading"
      panelClassName="modal-panel--compact"
    >
      <div className="issues-toolbar-dialog">
        <div className="issues-toolbar-popover-header">
          <div className="issues-toolbar-heading">
            <Filter size={16} />
            <span id="issue-filters-heading">Filters & Columns</span>
          </div>
          <button
            type="button"
            className="icon-button icon-button--ghost"
            onClick={onRequestClose}
            aria-label="Close filters"
          >
            <X size={14} />
          </button>
        </div>

        <div className="issues-toolbar-section">
          <div className="issues-toolbar-heading">
            <Filter size={16} />
            <span>Filters</span>
          </div>
          <div className="issues-toolbar-fields">
            <label className="issues-toolbar-field" style={{ flex: '1 1 100%' }}>
              <span>Search</span>
              <input
                type="text"
                value={searchFilter}
                onChange={(event) => onSearchFilterChange(event.target.value)}
                placeholder="Search by title, description, ID, tags..."
              />
            </label>

            <label className="issues-toolbar-field">
              <span>Priority</span>
              <div className="select-wrapper">
                <select
                  value={priorityFilter}
                  onChange={(event) => onPriorityFilterChange(event.target.value as PriorityFilterValue)}
                >
                  <option value="all">All priorities</option>
                  {(['Critical', 'High', 'Medium', 'Low'] as Priority[]).map((priority) => (
                    <option key={priority} value={priority}>
                      {priority}
                    </option>
                  ))}
                </select>
                <ChevronDown size={14} />
              </div>
            </label>

            <label className="issues-toolbar-field">
              <span>Application</span>
              <div className="select-wrapper">
                <select value={appFilter} onChange={(event) => onAppFilterChange(event.target.value)}>
                  <option value="all">All applications</option>
                  {appOptions.map((app) => (
                    <option key={app} value={app}>
                      {app}
                    </option>
                  ))}
                </select>
              </div>
            </label>
          </div>
          <button type="button" className="ghost-button issues-toolbar-clear" onClick={handleClearFilters}>
            Reset filters
          </button>
        </div>

        <div className="issues-toolbar-section">
          <div className="issues-toolbar-heading">
            <Eye size={16} />
            <span>Columns</span>
          </div>
          <div className="column-toggle-row">
            {statusOrder.map((status) => {
              const meta = ISSUE_BOARD_COLUMNS[status];
              const hidden = hiddenSet.has(status);
              return (
                <button
                  key={status}
                  type="button"
                  className={`column-toggle ${hidden ? 'column-toggle--inactive' : ''}`}
                  onClick={() => onToggleColumn(status)}
                  aria-pressed={!hidden}
                >
                  <span>{meta.title}</span>
                  {hidden ? <EyeOff size={14} /> : <Eye size={14} />}
                </button>
              );
            })}
          </div>
          {hiddenColumns.length > 0 && (
            <button type="button" className="link-button" onClick={onResetColumns}>
              Show all columns
            </button>
          )}
        </div>
      </div>
    </Modal>
  );
}
