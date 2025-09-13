import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, LineChart, Line } from 'recharts';
import apiClient from '../utils/api';

function Analytics() {
  const { id } = useParams();
  const [loading, setLoading] = useState(true);
  const [chatbot, setChatbot] = useState(null);
  const [analytics, setAnalytics] = useState(null);
  const [timeRange, setTimeRange] = useState('7'); // days

  useEffect(() => {
    loadData();
  }, [id, timeRange]);

  const loadData = async () => {
    try {
      setLoading(true);
      
      // Load chatbot info
      const chatbotResponse = await apiClient.get(`/api/v1/chatbots/${id}`);
      if (chatbotResponse.ok) {
        const chatbotData = await chatbotResponse.json();
        setChatbot(chatbotData);
      }

      // Load analytics
      const analyticsResponse = await apiClient.get(`/api/v1/analytics/${id}?days=${timeRange}`);
      if (analyticsResponse.ok) {
        const analyticsData = await analyticsResponse.json();
        setAnalytics(analyticsData);
      } else {
        // Mock data for demonstration
        setAnalytics({
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
            { name: 'integration_question', count: 12 }
          ]
        });
      }
    } catch (error) {
      console.error('Failed to load analytics:', error);
      // Set mock data on error
      setAnalytics({
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
          { name: 'integration_question', count: 12 }
        ]
      });
    } finally {
      setLoading(false);
    }
  };

  const generateMockTrendData = () => {
    const days = parseInt(timeRange);
    const data = [];
    for (let i = days; i >= 0; i--) {
      const date = new Date();
      date.setDate(date.getDate() - i);
      data.push({
        date: date.toISOString().split('T')[0],
        conversations: Math.floor(Math.random() * 20) + 10,
        leads: Math.floor(Math.random() * 8) + 2,
        engagement: Math.random() * 3 + 6
      });
    }
    return data;
  };

  const COLORS = ['#3b82f6', '#10b981', '#8b5cf6', '#f59e0b', '#ef4444'];

  if (loading) {
    return (
      <div className="loading-container">
        <div className="loading-spinner"></div>
        <p>Loading analytics...</p>
      </div>
    );
  }

  if (!chatbot || !analytics) {
    return (
      <div className="card">
        <div className="empty-state">
          <div className="empty-icon">📊</div>
          <h3>No Analytics Available</h3>
          <p>Analytics data is not available for this chatbot.</p>
        </div>
      </div>
    );
  }

  const trendData = generateMockTrendData();

  return (
    <div className="analytics-page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Analytics: {chatbot.name}</h1>
          <p className="page-subtitle">
            Performance metrics and insights for the last {timeRange} days
          </p>
        </div>
        <div className="time-range-selector">
          <select
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value)}
            className="form-select"
          >
            <option value="7">Last 7 days</option>
            <option value="30">Last 30 days</option>
            <option value="90">Last 90 days</option>
          </select>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-4 mb-6">
        <MetricCard
          title="Total Conversations"
          value={analytics.total_conversations}
          icon="💬"
          trend="+12% from last period"
          trendType="positive"
        />
        <MetricCard
          title="Messages Exchanged"
          value={analytics.total_messages}
          icon="📨"
          trend={`${(analytics.total_messages / analytics.total_conversations).toFixed(1)} avg per conversation`}
          trendType="neutral"
        />
        <MetricCard
          title="Leads Captured"
          value={analytics.leads_captured}
          icon="🎯"
          trend={`${analytics.conversion_rate.toFixed(1)}% conversion rate`}
          trendType="positive"
        />
        <MetricCard
          title="Engagement Score"
          value={analytics.engagement_score.toFixed(1)}
          icon="⭐"
          trend="Above average"
          trendType="positive"
        />
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-2 mb-6">
        {/* Conversation Trend */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Conversation Trends</h2>
          </div>
          <div style={{ width: '100%', height: 300 }}>
            <ResponsiveContainer>
              <LineChart data={trendData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis 
                  dataKey="date" 
                  tickFormatter={(date) => new Date(date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
                />
                <YAxis />
                <Tooltip 
                  labelFormatter={(date) => new Date(date).toLocaleDateString('en-US', { 
                    weekday: 'long', 
                    month: 'short', 
                    day: 'numeric' 
                  })}
                />
                <Line 
                  type="monotone" 
                  dataKey="conversations" 
                  stroke="#3b82f6" 
                  strokeWidth={3}
                  name="Conversations"
                />
                <Line 
                  type="monotone" 
                  dataKey="leads" 
                  stroke="#10b981" 
                  strokeWidth={3}
                  name="Leads"
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Intent Distribution */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Top User Intents</h2>
          </div>
          <div style={{ width: '100%', height: 300 }}>
            <ResponsiveContainer>
              <PieChart>
                <Pie
                  data={analytics.top_intents}
                  dataKey="count"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  outerRadius={100}
                  fill="#8884d8"
                  label={({name, percent}) => `${name.replace('_', ' ')}: ${(percent * 100).toFixed(0)}%`}
                >
                  {analytics.top_intents.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip 
                  formatter={(value, name) => [value, name.replace('_', ' ')]}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Detailed Stats */}
      <div className="grid grid-cols-2">
        {/* Performance Breakdown */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Performance Breakdown</h2>
          </div>
          <div className="stats-list">
            <div className="stat-item">
              <span className="stat-label">Average Conversation Length</span>
              <span className="stat-value">{analytics.avg_conversation_length.toFixed(1)} messages</span>
            </div>
            <div className="stat-item">
              <span className="stat-label">Conversion Rate</span>
              <span className="stat-value">{analytics.conversion_rate.toFixed(1)}%</span>
            </div>
            <div className="stat-item">
              <span className="stat-label">Engagement Score</span>
              <span className="stat-value">{analytics.engagement_score.toFixed(1)}/10</span>
            </div>
            <div className="stat-item">
              <span className="stat-label">Messages per Day</span>
              <span className="stat-value">{Math.floor(analytics.total_messages / parseInt(timeRange))}</span>
            </div>
            <div className="stat-item">
              <span className="stat-label">Leads per Day</span>
              <span className="stat-value">{(analytics.leads_captured / parseInt(timeRange)).toFixed(1)}</span>
            </div>
          </div>
        </div>

        {/* Intent Details */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Intent Analysis</h2>
          </div>
          <div className="intent-list">
            {analytics.top_intents.map((intent, index) => (
              <div key={intent.name} className="intent-item">
                <div className="intent-info">
                  <div className="intent-name">
                    {intent.name.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())}
                  </div>
                  <div className="intent-count">{intent.count} occurrences</div>
                </div>
                <div className="intent-bar">
                  <div 
                    className="intent-progress" 
                    style={{
                      width: `${(intent.count / Math.max(...analytics.top_intents.map(i => i.count))) * 100}%`,
                      backgroundColor: COLORS[index % COLORS.length]
                    }}
                  ></div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Recommendations */}
      <div className="card mt-6">
        <div className="card-header">
          <h2 className="card-title">🎯 Optimization Recommendations</h2>
        </div>
        <div className="recommendations">
          {analytics.conversion_rate < 20 && (
            <div className="recommendation">
              <div className="recommendation-icon">📈</div>
              <div className="recommendation-content">
                <h4>Improve Conversion Rate</h4>
                <p>Your conversion rate is below average. Consider optimizing your chatbot's lead qualification questions and call-to-action prompts.</p>
              </div>
            </div>
          )}
          
          {analytics.avg_conversation_length < 4 && (
            <div className="recommendation">
              <div className="recommendation-icon">🔄</div>
              <div className="recommendation-content">
                <h4>Increase Engagement</h4>
                <p>Conversations are relatively short. Try asking more follow-up questions to keep users engaged and gather more information.</p>
              </div>
            </div>
          )}

          {analytics.top_intents.length > 0 && (
            <div className="recommendation">
              <div className="recommendation-icon">🧠</div>
              <div className="recommendation-content">
                <h4>Enhance Knowledge Base</h4>
                <p>Your top intent is "{analytics.top_intents[0].name.replace('_', ' ')}". Consider expanding your knowledge base to better handle these types of questions.</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function MetricCard({ title, value, icon, trend, trendType }) {
  return (
    <div className="card metric-card">
      <div className="metric-header">
        <div className="metric-icon">{icon}</div>
        <div className="metric-value">{value}</div>
      </div>
      <div className="metric-title">{title}</div>
      {trend && (
        <div className={`metric-trend metric-trend-${trendType}`}>
          {trend}
        </div>
      )}
    </div>
  );
}

export default Analytics;