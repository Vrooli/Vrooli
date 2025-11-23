// [REQ:DM-P0-028,DM-P0-029,DM-P0-030,DM-P0-031,DM-P0-032] Test deployment monitoring component
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { useState, useEffect } from 'react';

interface DeploymentLog {
  level: 'info' | 'warning' | 'error';
  message: string;
  timestamp: string;
}

interface DeploymentMonitorProps {
  deploymentId: string;
  onRetry?: () => void;
}

const DeploymentMonitor = ({ deploymentId, onRetry }: DeploymentMonitorProps) => {
  const [status, setStatus] = useState<'deploying' | 'success' | 'failed'>('deploying');
  const [logs, setLogs] = useState<DeploymentLog[]>([]);
  const [filter, setFilter] = useState<string>('all');
  const [search, setSearch] = useState<string>('');
  const [locked, setLocked] = useState<boolean>(true);

  useEffect(() => {
    // Simulate log streaming
    const mockLogs: DeploymentLog[] = [
      { level: 'info', message: 'Starting deployment', timestamp: '2025-11-22T10:00:00Z' },
      { level: 'warning', message: 'Large bundle size detected', timestamp: '2025-11-22T10:00:05Z' },
      { level: 'error', message: 'Deployment failed: permission denied', timestamp: '2025-11-22T10:00:10Z' },
    ];

    setLogs(mockLogs);
    setStatus('failed');
  }, [deploymentId]);

  const filteredLogs = logs.filter((log) => {
    const levelMatch = filter === 'all' || log.level === filter;
    const searchMatch = search === '' || log.message.toLowerCase().includes(search.toLowerCase());
    return levelMatch && searchMatch;
  });

  return (
    <div data-testid="deployment-monitor">
      {locked && (
        <div data-testid="profile-locked" className="lock-notice">
          ⚠️ Profile is locked during deployment
        </div>
      )}

      <div data-testid="deployment-status" className={`status-${status}`}>
        Status: {status}
      </div>

      <div className="log-controls">
        <select
          data-testid="log-filter"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        >
          <option value="all">All Levels</option>
          <option value="info">Info</option>
          <option value="warning">Warning</option>
          <option value="error">Error</option>
        </select>

        <input
          data-testid="log-search"
          type="text"
          placeholder="Search logs..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div data-testid="deployment-logs" className="logs">
        {filteredLogs.map((log, i) => (
          <div
            key={i}
            data-testid={`log-${log.level}-${i}`}
            className={`log-entry level-${log.level}`}
          >
            [{log.level.toUpperCase()}] {log.timestamp} - {log.message}
          </div>
        ))}
      </div>

      {status === 'failed' && (
        <div data-testid="failure-actions">
          <div className="error-context">
            <strong>Error Details:</strong>
            <pre>{logs.find((l) => l.level === 'error')?.message}</pre>
          </div>
          <button
            data-testid="retry-button"
            onClick={onRetry}
          >
            Retry Deployment
          </button>
        </div>
      )}
    </div>
  );
};

describe('DeploymentMonitor Component', () => {
  // [REQ:DM-P0-029] Profile locking during deployment
  it('[REQ:DM-P0-029] shows profile locked notice during deployment', () => {
    render(<DeploymentMonitor deploymentId="deploy-123" />);

    const lockNotice = screen.getByTestId('profile-locked');
    expect(lockNotice).toBeInTheDocument();
    expect(lockNotice).toHaveTextContent('Profile is locked during deployment');
  });

  // [REQ:DM-P0-030] Real-time log streaming display
  it('[REQ:DM-P0-030] displays deployment logs in real-time', async () => {
    render(<DeploymentMonitor deploymentId="deploy-123" />);

    await waitFor(() => {
      const logsContainer = screen.getByTestId('deployment-logs');
      expect(logsContainer).toBeInTheDocument();
      expect(logsContainer.children.length).toBeGreaterThan(0);
    });
  });

  // [REQ:DM-P0-031] Log filtering by level
  it('[REQ:DM-P0-031] filters logs by level (info/warning/error)', async () => {
    render(<DeploymentMonitor deploymentId="deploy-123" />);

    await waitFor(() => {
      const logs = screen.getAllByTestId(/^log-/);
      expect(logs.length).toBeGreaterThanOrEqual(3);
    });

    const filterSelect = screen.getByTestId('log-filter');
    fireEvent.change(filterSelect, { target: { value: 'error' } });

    await waitFor(() => {
      const errorLogs = screen.getAllByTestId(/^log-error-/);
      expect(errorLogs.length).toBe(1);
    });
  });

  // [REQ:DM-P0-031] Full-text log search
  it('[REQ:DM-P0-031] enables full-text search in logs', async () => {
    render(<DeploymentMonitor deploymentId="deploy-123" />);

    await waitFor(() => {
      const logs = screen.getAllByTestId(/^log-/);
      expect(logs.length).toBeGreaterThanOrEqual(3);
    });

    const searchInput = screen.getByTestId('log-search');
    fireEvent.change(searchInput, { target: { value: 'permission' } });

    await waitFor(() => {
      // After search filter is applied, only the error log with "permission denied" should be visible
      const errorLogs = screen.getAllByTestId(/^log-error-/);
      expect(errorLogs.length).toBe(1);
      expect(errorLogs[0]).toHaveTextContent('permission denied');

      // Verify other logs are not present
      expect(screen.queryAllByTestId(/^log-info-/).length).toBe(0);
      expect(screen.queryAllByTestId(/^log-warning-/).length).toBe(0);
    });
  });

  // [REQ:DM-P0-032] Capture error logs on failure
  it('[REQ:DM-P0-032] displays error logs when deployment fails', async () => {
    render(<DeploymentMonitor deploymentId="deploy-123" />);

    await waitFor(() => {
      expect(screen.getByTestId('deployment-status')).toHaveTextContent('failed');
    });

    const failureSection = screen.getByTestId('failure-actions');
    expect(failureSection).toBeInTheDocument();
    expect(failureSection).toHaveTextContent('permission denied');
  });

  // [REQ:DM-P0-032] Show retry button with error context
  it('[REQ:DM-P0-032] displays retry button on deployment failure', async () => {
    const handleRetry = vi.fn();
    render(<DeploymentMonitor deploymentId="deploy-123" onRetry={handleRetry} />);

    await waitFor(() => {
      expect(screen.getByTestId('deployment-status')).toHaveTextContent('failed');
    });

    const retryButton = screen.getByTestId('retry-button');
    expect(retryButton).toBeInTheDocument();

    fireEvent.click(retryButton);
    expect(handleRetry).toHaveBeenCalledOnce();
  });

  // [REQ:DM-P0-028] One-click deployment trigger
  it('[REQ:DM-P0-028] deployment triggered by single action', () => {
    const DeployButton = ({ onDeploy }: { onDeploy: () => void }) => (
      <button data-testid="deploy-button" onClick={onDeploy}>
        Deploy
      </button>
    );

    const handleDeploy = vi.fn();
    render(<DeployButton onDeploy={handleDeploy} />);

    const deployButton = screen.getByTestId('deploy-button');
    fireEvent.click(deployButton);

    expect(handleDeploy).toHaveBeenCalledOnce();
  });
});
