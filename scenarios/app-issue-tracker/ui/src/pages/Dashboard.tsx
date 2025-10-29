import { KeyboardEvent, useMemo } from 'react';
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  CircleSlash,
  Clock3,
  Flame,
  TrendingUp,
  XCircle,
  AlertOctagon,
} from 'lucide-react';
import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  Title,
  Tooltip,
} from 'chart.js';
import { Bar } from 'react-chartjs-2';
import type { Issue } from '../types/issue';
import type { AgentSettings, DashboardStats, ProcessorSettings } from '../data/sampleData';
import { MANUAL_FAILURE_REASON_LABELS, type ManualFailureReason } from '../types/issue';
import { StatsCard } from '../components/StatsCard';

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend);

interface DashboardProps {
  stats: DashboardStats;
  issues: Issue[];
  processor: ProcessorSettings;
  agentSettings: AgentSettings;
  onOpenIssues: () => void;
  onOpenIssue: (issueId: string) => void;
  onOpenAutomationSettings: () => void;
}

export function Dashboard({
  stats,
  issues,
  processor,
  agentSettings,
  onOpenIssues,
  onOpenIssue,
  onOpenAutomationSettings,
}: DashboardProps) {
  const priorityData = useMemo(() => {
    const labels = Object.keys(stats.priorityBreakdown);
    const values = Object.values(stats.priorityBreakdown);
    return {
      labels,
      datasets: [
        {
          label: 'Issues by Priority',
          data: values,
          borderRadius: 8,
          backgroundColor: ['#f87171', '#fb923c', '#60a5fa', '#34d399'],
        },
      ],
    };
  }, [stats.priorityBreakdown]);

  const priorityOptions = useMemo(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { grid: { display: false } },
        y: { beginAtZero: true, ticks: { precision: 0 } },
      },
    }),
    [],
  );

  const trendData = useMemo(
    () => ({
      labels: stats.statusTrend.map((item) => item.label),
      datasets: [
        {
          label: 'Investigated Issues',
          data: stats.statusTrend.map((item) => item.value),
          backgroundColor: '#6366f1',
          borderRadius: 8,
          barThickness: 32,
        },
      ],
    }),
    [stats.statusTrend],
  );

  const trendOptions = useMemo(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { grid: { display: false } },
        y: { beginAtZero: true, ticks: { precision: 0 } },
      },
    }),
    [],
  );

  // Failure breakdown chart
  const failureBreakdownData = useMemo(() => {
    if (!stats.failureReasonsBreakdown || Object.keys(stats.failureReasonsBreakdown).length === 0) {
      return null;
    }

    const formatReasonLabel = (reason: string): string => {
      // Check if it's a manual failure reason
      if (reason in MANUAL_FAILURE_REASON_LABELS) {
        return MANUAL_FAILURE_REASON_LABELS[reason as ManualFailureReason];
      }
      // Format auto-failure reasons
      return reason
        .split('_')
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ');
    };

    const entries = Object.entries(stats.failureReasonsBreakdown).sort(([, a], [, b]) => b - a);
    const labels = entries.map(([reason]) => formatReasonLabel(reason));
    const values = entries.map(([, count]) => count);

    return {
      labels,
      datasets: [
        {
          label: 'Failure Count',
          data: values,
          backgroundColor: '#ef4444',
          borderRadius: 6,
          barThickness: 28,
        },
      ],
    };
  }, [stats.failureReasonsBreakdown]);

  const failureBreakdownOptions = useMemo(
    () => ({
      indexAxis: 'y' as const,
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: (context: any) => `${context.parsed.x} failure${context.parsed.x !== 1 ? 's' : ''}`,
          },
        },
      },
      scales: {
        x: {
          beginAtZero: true,
          ticks: { precision: 0 },
          grid: { display: true },
        },
        y: { grid: { display: false } },
      },
    }),
    [],
  );

  const recentIssues = useMemo(
    () =>
      issues
        .slice()
        .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
        .slice(0, 5),
    [issues],
  );

  const handleAutomationCardKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onOpenAutomationSettings();
    }
  };

  return (
    <div className="dashboard-page">
      <section className="page-header">
        <div>
          <h2>Operational Overview</h2>
          <p>Monitor pipeline throughput, queues, and AI agent utilization.</p>
        </div>
        <div className={`status-pill ${processor.active ? 'online' : 'offline'}`}>
          <span className="status-dot" />
          {processor.active ? 'Automation online' : 'Automation paused'}
        </div>
      </section>

      <section className="stats-grid">
        <StatsCard
          title="Total Issues"
          value={stats.totalIssues}
          subtitle="All tracked issues"
          icon={Activity}
          onClick={onOpenIssues}
        />
        <StatsCard
          title="Open"
          value={stats.openIssues}
          subtitle="Awaiting triage"
          icon={AlertTriangle}
          tone="warning"
          onClick={onOpenIssues}
        />
        <StatsCard
          title="Active"
          value={stats.inProgress}
          subtitle="Agent currently working"
          icon={Clock3}
          onClick={onOpenIssues}
        />
        <StatsCard
          title="Completed Today"
          value={stats.completedToday}
          subtitle="Resolved in last 24h"
          icon={CheckCircle2}
          tone="success"
          onClick={onOpenIssues}
        />
      </section>

      {(stats.manualFailures !== undefined || stats.autoFailures !== undefined) && (
        <section className="stats-grid" style={{ marginTop: '24px' }}>
          {stats.manualFailures !== undefined && (
            <StatsCard
              title="Manual Failures"
              value={stats.manualFailures}
              subtitle="Incorrectly marked complete"
              icon={XCircle}
              tone="danger"
              onClick={onOpenIssues}
            />
          )}
          {stats.autoFailures !== undefined && (
            <StatsCard
              title="Auto Failures"
              value={stats.autoFailures}
              subtitle="Agent execution errors"
              icon={CircleSlash}
              tone="danger"
              onClick={onOpenIssues}
            />
          )}
          {stats.manualFailures !== undefined && stats.autoFailures !== undefined && (
            <StatsCard
              title="Total Failures"
              value={stats.manualFailures + stats.autoFailures}
              subtitle="All failure types"
              icon={AlertOctagon}
              tone="danger"
              onClick={onOpenIssues}
            />
          )}
        </section>
      )}

      {failureBreakdownData && (
        <section className="chart-row" style={{ marginTop: '24px' }}>
          <div className="chart-card" style={{ gridColumn: '1 / -1' }}>
            <header>
              <h3>Failure Breakdown by Type</h3>
              <AlertOctagon size={16} />
            </header>
            <div className="chart-wrapper" style={{ minHeight: '280px' }}>
              <Bar options={failureBreakdownOptions} data={failureBreakdownData} />
            </div>
          </div>
        </section>
      )}

      <section className="chart-row">
        <div className="chart-card">
          <header>
            <h3>Issues by Priority</h3>
            <Flame size={16} />
          </header>
          <div className="chart-wrapper">
            <Bar options={priorityOptions} data={priorityData} />
          </div>
        </div>
        <div className="chart-card">
          <header>
            <h3>Investigations this Week</h3>
            <TrendingUp size={16} />
          </header>
          <div className="chart-wrapper">
            <Bar options={trendOptions} data={trendData} />
          </div>
        </div>
      </section>

      <section className="dashboard-detail-grid">
        <div className="detail-card">
          <header>
            <h3>Recent Issues</h3>
            <span className="detail-meta">Latest 5 issues</span>
          </header>
          <ul className="recent-issue-list">
            {recentIssues.map((issue) => (
              <li key={issue.id}>
                <button
                  type="button"
                  className="recent-issue-button"
                  onClick={() => onOpenIssue(issue.id)}
                  aria-label={`Open issue ${issue.title || issue.id}`}
                >
                  <span className="issue-title-line">
                    <span className="recent-issue-title">{issue.title}</span>
                    <span className={`pill priority-${issue.priority.toLowerCase()}`}>{issue.priority}</span>
                  </span>
                  <span className="recent-issue-summary">{issue.summary}</span>
                </button>
              </li>
            ))}
          </ul>
        </div>
        <div
          className="detail-card detail-card--interactive"
          role="button"
          tabIndex={0}
          onClick={onOpenAutomationSettings}
          onKeyDown={handleAutomationCardKeyDown}
          aria-label="Open automation settings"
        >
          <header>
            <h3>Automation Configuration</h3>
            <span className="detail-meta">Unified agent</span>
          </header>
          <dl className="settings-summary">
            <div>
              <dt>Concurrent slots</dt>
              <dd>{processor.concurrentSlots}</dd>
            </div>
            <div>
              <dt>Refresh cadence</dt>
              <dd>{processor.refreshInterval}s</dd>
            </div>
            <div>
              <dt>Agent</dt>
              <dd>{agentSettings.agentId}</dd>
            </div>
            <div>
              <dt>Maximum turns</dt>
              <dd>{agentSettings.maximumTurns}</dd>
            </div>
            <div>
              <dt>Allowed tools</dt>
              <dd>{agentSettings.allowedTools.join(', ')}</dd>
            </div>
            <div>
              <dt>Skip permissions</dt>
              <dd>{agentSettings.skipPermissionChecks ? 'Enabled' : 'Disabled'}</dd>
            </div>
          </dl>
        </div>
      </section>
    </div>
  );
}
