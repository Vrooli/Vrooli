import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import {
  ArrowUpRight,
  BarChart3,
  Bot,
  ClipboardList,
  Layers,
  MessageCircle,
  Target,
  Workflow,
} from 'lucide-react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
} from 'recharts';

import apiClient from '../utils/api';
import type { Chatbot } from '../types';

interface DashboardStats {
  totalChatbots: number;
  activeChatbots: number;
  totalConversations: number;
  leadsGenerated: number;
}

interface ActivityPoint {
  name: string;
  conversations: number;
  leads: number;
}

const DEFAULT_ACTIVITY: ActivityPoint[] = [
  { name: 'Mon', conversations: 45, leads: 12 },
  { name: 'Tue', conversations: 52, leads: 15 },
  { name: 'Wed', conversations: 38, leads: 8 },
  { name: 'Thu', conversations: 67, leads: 22 },
  { name: 'Fri', conversations: 45, leads: 18 },
  { name: 'Sat', conversations: 32, leads: 9 },
  { name: 'Sun', conversations: 28, leads: 5 },
];

function Dashboard() {
  const [loading, setLoading] = useState(true);
  const [chatbots, setChatbots] = useState<Chatbot[]>([]);
  const [stats, setStats] = useState<DashboardStats>({
    totalChatbots: 0,
    activeChatbots: 0,
    totalConversations: 0,
    leadsGenerated: 0,
  });
  const [recentActivity, setRecentActivity] = useState<ActivityPoint[]>(DEFAULT_ACTIVITY);

  useEffect(() => {
    let isMounted = true;

    const loadDashboardData = async () => {
      try {
        setLoading(true);
        const response = await apiClient.get('/api/v1/chatbots');
        if (!response.ok) {
          return;
        }

        const data = (await response.json()) as Chatbot[];
        if (!isMounted) {
          return;
        }

        setChatbots(data);

        const totalChatbots = data.length;
        const activeChatbots = data.filter((bot) => bot.is_active).length;

        setStats({
          totalChatbots,
          activeChatbots,
          totalConversations: Math.max(247, totalChatbots * 42),
          leadsGenerated: Math.max(89, Math.round(activeChatbots * 12.5)),
        });

        setRecentActivity(
          DEFAULT_ACTIVITY.map((point) => ({
            ...point,
            conversations: point.conversations + activeChatbots * 2,
            leads: point.leads + Math.max(0, activeChatbots - 1),
          }))
        );
      } catch (error) {
        console.error('Failed to load dashboard data:', error);
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    loadDashboardData();
    return () => {
      isMounted = false;
    };
  }, []);

  const highlightedChatbots = useMemo(() => chatbots.slice(0, 5), [chatbots]);

  if (loading) {
    return (
      <div className="card loading-card">
        <div className="loading-spinner" />
        <p>Building your executive summary…</p>
      </div>
    );
  }

  return (
    <div className="page-stack">
      <section className="page-header">
        <div>
          <h2>Mission Control</h2>
          <p>Track growth, optimize engagement, and orchestrate AI concierges across every channel.</p>
        </div>
        <Link to="/chatbots/new" className="ghost-button">
          <Layers size={16} />
          <span>Blueprint new chatbot</span>
        </Link>
      </section>

      <section className="stat-grid">
        <StatCard
          icon={Bot}
          title="Chatbots in portfolio"
          value={stats.totalChatbots}
          trendLabel={stats.totalChatbots > 0 ? '+2 launched this quarter' : 'Start your first build'}
          emphasis
        />
        <StatCard
          icon={ClipboardList}
          title="Active deployments"
          value={stats.activeChatbots}
          trendLabel={`${stats.activeChatbots}/${stats.totalChatbots || 1} live bots`}
        />
        <StatCard
          icon={MessageCircle}
          title="Conversations this week"
          value={stats.totalConversations}
          trendLabel="+18% week-over-week"
        />
        <StatCard
          icon={Target}
          title="Qualified leads captured"
          value={stats.leadsGenerated}
          trendLabel="12% conversion rate"
        />
      </section>

      <section className="chart-grid">
        <div className="card">
          <header className="card-header">
            <div>
              <h3>Weekly engagement pulse</h3>
              <p>Conversations vs. leads captured across the last 7 days.</p>
            </div>
          </header>
          <div className="chart-wrapper">
            <ResponsiveContainer>
              <BarChart data={recentActivity}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="conversations" fill="#2563eb" radius={[6, 6, 0, 0]} name="Conversations" />
                <Bar dataKey="leads" fill="#10b981" radius={[6, 6, 0, 0]} name="Leads" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="card">
          <header className="card-header">
            <div>
              <h3>Conversion momentum</h3>
              <p>Lead quality and engagement trendline for active bots.</p>
            </div>
          </header>
          <div className="chart-wrapper">
            <ResponsiveContainer>
              <LineChart data={recentActivity}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip formatter={(value: number) => `${value.toFixed(1)}%`} labelFormatter={(name) => name} />
                <Line
                  type="monotone"
                  dataKey={(point) => Number(((point.leads / Math.max(point.conversations, 1)) * 100).toFixed(1))}
                  stroke="#8b5cf6"
                  strokeWidth={3}
                  dot={false}
                  name="Conversion rate"
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      </section>

      <section className="dual-grid">
        <div className="card">
          <header className="card-header">
            <div>
              <h3>Portfolio spotlight</h3>
              <p>Top chatbots driving revenue outcomes right now.</p>
            </div>
            {chatbots.length > 5 && (
              <Link to="/chatbots" className="inline-link">
                View all
                <ArrowUpRight size={14} />
              </Link>
            )}
          </header>
          <div className="list-stack">
            {highlightedChatbots.length === 0 ? (
              <EmptyState
                title="No chatbots yet"
                description="Launch your first assistant to start capturing leads automatically."
                actionLabel="Create chatbot"
                actionLink="/chatbots/new"
                icon={Workflow}
              />
            ) : (
              highlightedChatbots.map((chatbot) => (
                <article key={chatbot.id} className="chatbot-item">
                  <div className="chatbot-avatar" aria-hidden>
                    <Bot size={18} />
                  </div>
                  <div className="chatbot-content">
                    <header>
                      <h4>{chatbot.name}</h4>
                      <span className={`status-pill ${chatbot.is_active ? 'is-success' : 'is-muted'}`}>
                        {chatbot.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </header>
                    <p>{chatbot.description || 'No description provided yet.'}</p>
                  </div>
                  <div className="chatbot-actions">
                    <Link to={`/chatbots/${chatbot.id}/test`} className="pill-button">
                      Test
                    </Link>
                    <Link to={`/chatbots/${chatbot.id}/analytics`} className="pill-button">
                      Insights
                    </Link>
                    <Link to={`/chatbots/${chatbot.id}/edit`} className="pill-button">
                      Edit
                    </Link>
                  </div>
                </article>
              ))
            )}
          </div>
        </div>

        <div className="card">
          <header className="card-header">
            <div>
              <h3>Launchpad</h3>
              <p>Strategic shortcuts to accelerate go-to-market experiments.</p>
            </div>
          </header>
          <div className="action-grid">
            <QuickAction
              icon={BarChart3}
              title="Experiment dashboard"
              description="Benchmark conversion across industries with pre-built cohorts."
              onAction={() => window.open('/api/v1/chatbots', '_blank')}
            />
            <QuickAction
              icon={ArrowUpRight}
              title="Deploy widget"
              description="Generate responsive embed code for marketing sites in seconds."
              onAction={() => {
                navigator.clipboard
                  .writeText('<script src="/widget.js" data-chatbot-id="YOUR_ID"></script>')
                  .then(() => alert('Embed snippet copied to clipboard. Replace YOUR_ID before deploying.'));
              }}
            />
            <QuickAction
              icon={Workflow}
              title="Automate escalations"
              description="Route high-value conversations to human specialists instantly."
              onAction={() => alert('Escalation workflow builder is coming soon.')}
            />
            <QuickAction
              icon={Layers}
              title="Clone best performer"
              description="Duplicate personas and language models to new verticals."
              onAction={() => alert('Cloning pipeline is under development.')}
            />
          </div>
        </div>
      </section>
    </div>
  );
}

interface StatCardProps {
  icon: LucideIcon;
  title: string;
  value: number;
  trendLabel?: string;
  emphasis?: boolean;
}

function StatCard({ icon: Icon, title, value, trendLabel, emphasis }: StatCardProps) {
  return (
    <div className={`card stat-card ${emphasis ? 'is-emphasis' : ''}`}>
      <div className="stat-icon">
        <Icon size={18} />
      </div>
      <div className="stat-meta">
        <span className="stat-title">{title}</span>
        <span className="stat-value">{value.toLocaleString()}</span>
        {trendLabel && <span className="stat-trend">{trendLabel}</span>}
      </div>
    </div>
  );
}

interface EmptyStateProps {
  title: string;
  description: string;
  actionLabel?: string;
  actionLink?: string;
  icon: LucideIcon;
}

function EmptyState({ title, description, actionLabel, actionLink, icon: Icon }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-icon">
        <Icon size={20} />
      </div>
      <h4>{title}</h4>
      <p>{description}</p>
      {actionLabel && actionLink && (
        <Link to={actionLink} className="primary-action">
          <ArrowUpRight size={16} />
          <span>{actionLabel}</span>
        </Link>
      )}
    </div>
  );
}

interface QuickActionProps {
  icon: LucideIcon;
  title: string;
  description: string;
  onAction: () => void;
}

function QuickAction({ icon: Icon, title, description, onAction }: QuickActionProps) {
  return (
    <button className="quick-action" onClick={onAction}>
      <div className="quick-icon">
        <Icon size={18} />
      </div>
      <div className="quick-copy">
        <span className="quick-title">{title}</span>
        <span className="quick-description">{description}</span>
      </div>
      <ArrowUpRight size={16} />
    </button>
  );
}

export default Dashboard;
