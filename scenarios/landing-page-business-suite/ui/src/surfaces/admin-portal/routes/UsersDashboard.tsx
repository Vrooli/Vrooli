import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { Button } from '../../../shared/ui/button';
import {
  Users,
  MessageSquare,
  ClipboardList,
  RefreshCw,
  AlertTriangle,
  AlertCircle,
  CheckCircle,
} from 'lucide-react';
import { LAYOUT } from '../config/layout.constants';
import { fetchFeedbackList, type FeedbackRequest } from '../../../shared/api/feedback';
import { getWaitlistEmails } from '../../../shared/api/waitlist';
import type { WaitlistEmail } from '../../../shared/api/types';

interface UsersStats {
  feedbackPending: number;
  feedbackTotal: number;
  waitlistCount: number;
}

/**
 * Users Dashboard - Entry point for customer management
 *
 * Provides quick flows, stats, and navigation for:
 * - User accounts management
 * - Feedback triage
 * - Waitlist management
 */
export function UsersDashboard() {
  const navigate = useNavigate();

  const [stats, setStats] = useState<UsersStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadStats = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [feedbackList, waitlistEmails] = await Promise.all([
        fetchFeedbackList().catch(() => [] as FeedbackRequest[]),
        getWaitlistEmails().catch(() => [] as WaitlistEmail[]),
      ]);

      const feedbackPending = feedbackList.filter(
        (f) => f.status === 'pending' || f.status === 'in_progress'
      ).length;

      setStats({
        feedbackPending,
        feedbackTotal: feedbackList.length,
        waitlistCount: waitlistEmails.length,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load user stats');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.sectionSpacing}>
        <PageHeader
          variant="icon-title"
          title="Users Dashboard"
          icon={Users}
          iconBgClass="bg-blue-500/10"
          iconColorClass="text-blue-400"
          testId="users-dashboard-header"
        />

        {/* Quick Flows */}
        <div className="mb-8" data-testid="users-quick-flows">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500 mb-4">
            Quick flows
          </p>
          <div className="grid gap-4 md:grid-cols-3">
            <QuickFlowCard
              title="View accounts"
              description="Manage user accounts, subscriptions, and sessions"
              icon={Users}
              iconBg="bg-blue-500/20"
              iconColor="text-blue-300"
              onClick={() => navigate('/admin/accounts')}
              testId="flow-accounts"
              badge="Soon"
            />
            <QuickFlowCard
              title="Triage feedback"
              description="Review and respond to user feedback"
              icon={MessageSquare}
              iconBg="bg-purple-500/20"
              iconColor="text-purple-300"
              onClick={() => navigate('/admin/feedback')}
              testId="flow-feedback"
              badge={stats?.feedbackPending ? `${stats.feedbackPending} pending` : undefined}
              badgeVariant={stats?.feedbackPending ? 'warning' : undefined}
            />
            <QuickFlowCard
              title="Manage waitlist"
              description="View and export coming soon signups"
              icon={ClipboardList}
              iconBg="bg-emerald-500/20"
              iconColor="text-emerald-300"
              onClick={() => navigate('/admin/waitlist')}
              testId="flow-waitlist"
            />
          </div>
        </div>

        {/* Stats Overview */}
        <div
          className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6"
          data-testid="users-stats"
        >
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-6">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                User activity
              </p>
              <h2 className="text-xl font-semibold text-white mt-1">
                Feedback and waitlist overview
              </h2>
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={loadStats}
              className="gap-2"
              data-testid="users-stats-refresh"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </Button>
          </div>

          {loading ? (
            <div className="grid gap-4 md:grid-cols-3">
              {[0, 1, 2].map((i) => (
                <div key={i} className="h-24 rounded-xl bg-white/5 animate-pulse" />
              ))}
            </div>
          ) : error ? (
            <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100 flex items-center gap-3">
              <AlertTriangle className="h-4 w-4" />
              <span>{error}</span>
              <Button size="sm" variant="ghost" onClick={loadStats}>
                Retry
              </Button>
            </div>
          ) : stats ? (
            <div className="grid gap-4 md:grid-cols-3">
              <StatCard
                label="Pending feedback"
                value={stats.feedbackPending}
                icon={stats.feedbackPending > 0 ? AlertCircle : CheckCircle}
                iconColor={stats.feedbackPending > 0 ? 'text-amber-300' : 'text-emerald-300'}
                iconBg={stats.feedbackPending > 0 ? 'bg-amber-500/20' : 'bg-emerald-500/20'}
                description={
                  stats.feedbackPending > 0
                    ? `${stats.feedbackPending} item${stats.feedbackPending !== 1 ? 's' : ''} need${stats.feedbackPending === 1 ? 's' : ''} attention`
                    : 'All feedback addressed'
                }
                onClick={() => navigate('/admin/feedback')}
              />
              <StatCard
                label="Total feedback"
                value={stats.feedbackTotal}
                icon={MessageSquare}
                iconColor="text-purple-300"
                iconBg="bg-purple-500/20"
                description={`${stats.feedbackTotal} submission${stats.feedbackTotal !== 1 ? 's' : ''} received`}
                onClick={() => navigate('/admin/feedback')}
              />
              <StatCard
                label="Waitlist signups"
                value={stats.waitlistCount}
                icon={ClipboardList}
                iconColor="text-blue-300"
                iconBg="bg-blue-500/20"
                description={
                  stats.waitlistCount > 0
                    ? `${stats.waitlistCount} user${stats.waitlistCount !== 1 ? 's' : ''} waiting`
                    : 'No signups yet'
                }
                onClick={() => navigate('/admin/waitlist')}
              />
            </div>
          ) : null}
        </div>

        {/* Feedback Attention Banner */}
        {stats && stats.feedbackPending > 0 && (
          <div
            className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-6"
            data-testid="users-feedback-attention"
          >
            <div className="flex items-start gap-4">
              <div className="p-3 rounded-xl bg-amber-500/20">
                <MessageSquare className="h-6 w-6 text-amber-300" />
              </div>
              <div className="flex-1">
                <h3 className="text-lg font-semibold text-white mb-2">
                  Feedback needs attention
                </h3>
                <p className="text-sm text-amber-100/80 mb-4">
                  You have {stats.feedbackPending} pending feedback{' '}
                  {stats.feedbackPending !== 1 ? 'items' : 'item'} waiting for review.
                  Responding promptly helps build trust with your users.
                </p>
                <Button
                  size="sm"
                  onClick={() => navigate('/admin/feedback')}
                  data-testid="users-feedback-triage"
                >
                  Triage feedback
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}

interface QuickFlowCardProps {
  title: string;
  description: string;
  icon: React.ElementType;
  iconBg: string;
  iconColor: string;
  onClick: () => void;
  testId: string;
  badge?: string;
  badgeVariant?: 'warning' | 'default';
}

function QuickFlowCard({
  title,
  description,
  icon: Icon,
  iconBg,
  iconColor,
  onClick,
  testId,
  badge,
  badgeVariant = 'default',
}: QuickFlowCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group rounded-xl border border-white/10 bg-white/5 p-4 text-left hover:bg-white/10 transition-all"
      data-testid={testId}
    >
      <div className="flex items-center gap-3 mb-2">
        <div className={`p-2 rounded-lg ${iconBg}`}>
          <Icon className={`h-4 w-4 ${iconColor}`} />
        </div>
        <p className="font-semibold text-white">{title}</p>
        {badge && (
          <span
            className={`text-[10px] uppercase px-2 py-0.5 rounded-full ${
              badgeVariant === 'warning'
                ? 'bg-amber-500/20 text-amber-300'
                : 'text-slate-500'
            }`}
          >
            {badge}
          </span>
        )}
      </div>
      <p className="text-sm text-slate-400">{description}</p>
      <span className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-slate-300 group-hover:translate-x-1 transition-transform">
        Go →
      </span>
    </button>
  );
}

interface StatCardProps {
  label: string;
  value: number;
  icon: React.ElementType;
  iconColor: string;
  iconBg: string;
  description: string;
  onClick: () => void;
}

function StatCard({
  label,
  value,
  icon: Icon,
  iconColor,
  iconBg,
  description,
  onClick,
}: StatCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-xl border border-white/10 bg-slate-900/40 p-4 text-left hover:bg-slate-900/60 transition-all"
    >
      <div className="flex items-center gap-3 mb-3">
        <div className={`p-2 rounded-lg ${iconBg}`}>
          <Icon className={`h-5 w-5 ${iconColor}`} />
        </div>
        <p className="text-xs uppercase tracking-[0.3em] text-slate-500">{label}</p>
      </div>
      <p className="text-3xl font-semibold text-white mb-1">{value}</p>
      <p className="text-xs text-slate-400">{description}</p>
    </button>
  );
}
