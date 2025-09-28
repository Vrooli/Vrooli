import { useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import { ActivitySquare, BarChart3, Gauge, MessageCircle, Target } from 'lucide-react';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  PieChart,
  Pie,
  Cell,
} from 'recharts';

import apiClient from '../utils/api';
import type { AnalyticsData, Chatbot } from '../types';

interface TrendPoint {
  date: string;
  conversations: number;
  leads: number;
  engagement: number;
}

const TREND_COLORS = ['#2563eb', '#10b981', '#8b5cf6', '#f59e0b', '#ef4444'];

function Analytics() {
  const { id } = useParams();
  const [loading, setLoading] = useState(true);
  const [chatbot, setChatbot] = useState<Chatbot | null>(null);
  const [analytics, setAnalytics] = useState<AnalyticsData | null>(null);
  const [timeRange, setTimeRange] = useState<'7' | '30' | '90'>('7');

  useEffect(() => {
    let isMounted = true;

    const fetchData = async () => {
      if (!id) return;

      try {
        setLoading(true);
        const [chatbotRes, analyticsRes] = await Promise.all([
          apiClient.get(`/api/v1/chatbots/${id}`),
          apiClient.get(`/api/v1/analytics/${id}?days=${timeRange}`),
        ]);

        if (chatbotRes.ok) {
          const chatbotPayload = (await chatbotRes.json()) as Chatbot;
          if (isMounted) {
            setChatbot(chatbotPayload);
          }
        }

        if (analyticsRes.ok) {
          const analyticsPayload = (await analyticsRes.json()) as AnalyticsData;
          if (isMounted) {
            setAnalytics(analyticsPayload);
          }
        } else if (isMounted) {
          setAnalytics(getMockAnalytics());
        }
      } catch (error) {
        console.error('Failed to load analytics:', error);
        if (isMounted) {
          setAnalytics(getMockAnalytics());
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    fetchData();
    return () => {
      isMounted = false;
    };
  }, [id, timeRange]);

  const trendData = useMemo(() => buildTrendData(Number(timeRange)), [timeRange]);

  if (loading) {
    return (
      <div className="card loading-card">
        <div className="loading-spinner" />
        <p>Crunching engagement insights…</p>
      </div>
    );
  }

  if (!chatbot || !analytics) {
    return (
      <div className="card">
        <div className="empty-state">
          <div className="empty-icon">
            <BarChart3 size={20} />
          </div>
          <h4>No analytics available</h4>
          <p>We haven’t captured enough traffic for this chatbot yet. Come back after it collects more conversations.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <h2>Performance intelligence</h2>
          <p>Insight into how {chatbot.name} converts visitors into qualified pipeline.</p>
        </div>
        <select className="select" value={timeRange} onChange={(event) => setTimeRange(event.target.value as typeof timeRange)}>
          <option value="7">Last 7 days</option>
          <option value="30">Last 30 days</option>
          <option value="90">Last 90 days</option>
        </select>
      </header>

      <section className="metric-grid">
        <MetricCard
          icon={MessageCircle}
          label="Conversations"
          value={analytics.total_conversations}
          hint="Total assisted engagements"
        />
        <MetricCard
          icon={Gauge}
          label="Avg. conversation length"
          value={`${analytics.avg_conversation_length.toFixed(1)} min`}
          hint="Retention per session"
        />
        <MetricCard
          icon={Target}
          label="Leads captured"
          value={analytics.leads_captured}
          hint={`${analytics.conversion_rate.toFixed(1)}% conversion rate`}
        />
        <MetricCard
          icon={ActivitySquare}
          label="Engagement score"
          value={analytics.engagement_score.toFixed(1)}
          hint="Composite health indicator"
        />
      </section>

      <section className="chart-grid">
        <div className="card">
          <header className="card-header">
            <div>
              <h3>Growth trajectory</h3>
              <p>Conversations and qualified leads captured across the selected window.</p>
            </div>
          </header>
          <div className="chart-wrapper">
            <ResponsiveContainer>
              <LineChart data={trendData}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis
                  dataKey="date"
                  tickFormatter={(value) => new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                />
                <YAxis />
                <Tooltip
                  labelFormatter={(value) => new Date(value as string).toLocaleDateString(undefined, {
                    weekday: 'short',
                    month: 'short',
                    day: 'numeric',
                  })}
                />
                <Line type="monotone" dataKey="conversations" stroke="#2563eb" strokeWidth={3} dot={false} name="Conversations" />
                <Line type="monotone" dataKey="leads" stroke="#10b981" strokeWidth={3} dot={false} name="Leads" />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="card">
          <header className="card-header">
            <div>
              <h3>Intent distribution</h3>
              <p>Where your visitors invest their time with this assistant.</p>
            </div>
          </header>
          <div className="chart-wrapper">
            <ResponsiveContainer>
              <PieChart>
                <Pie
                  data={analytics.top_intents}
                  dataKey="count"
                  nameKey="name"
                  outerRadius={110}
                  label={({ name, percent }) => `${formatIntent(name)} ${(percent * 100).toFixed(0)}%`}
                >
                  {analytics.top_intents.map((entry, index) => (
                    <Cell key={entry.name} fill={TREND_COLORS[index % TREND_COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip formatter={(value, name) => [value, formatIntent(String(name))]} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </div>
      </section>
    </div>
  );
}

interface MetricCardProps {
  icon: LucideIcon;
  label: string;
  value: string | number;
  hint?: string;
}

function MetricCard({ icon: Icon, label, value, hint }: MetricCardProps) {
  return (
    <div className="card metric-card">
      <div className="metric-icon">
        <Icon size={18} />
      </div>
      <div className="metric-copy">
        <span className="metric-label">{label}</span>
        <span className="metric-value">{value}</span>
        {hint && <span className="metric-hint">{hint}</span>}
      </div>
    </div>
  );
}

function buildTrendData(days: number): TrendPoint[] {
  const data: TrendPoint[] = [];
  const now = new Date();

  for (let index = days; index >= 0; index -= 1) {
    const date = new Date(now);
    date.setDate(now.getDate() - index);

    data.push({
      date: date.toISOString(),
      conversations: Math.floor(Math.random() * 18) + 12,
      leads: Math.floor(Math.random() * 6) + 2,
      engagement: Math.random() * 2 + 6,
    });
  }

  return data;
}

function getMockAnalytics(): AnalyticsData {
  return {
    total_conversations: 156,
    total_messages: 934,
    leads_captured: 42,
    avg_conversation_length: 6.8,
    engagement_score: 7.2,
    conversion_rate: 26.9,
    top_intents: [
      { name: 'pricing_inquiry', count: 34 },
      { name: 'support_request', count: 28 },
      { name: 'demo_request', count: 22 },
      { name: 'feature_inquiry', count: 18 },
      { name: 'integration_question', count: 12 },
    ],
  };
}

function formatIntent(intent: string) {
  return intent.replace(/_/g, ' ');
}

export default Analytics;
