import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line } from 'recharts';
import apiClient from '../utils/api';

function Dashboard() {
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    totalChatbots: 0,
    activeChatbots: 0,
    totalConversations: 0,
    leadsGenerated: 0
  });
  const [chatbots, setChatbots] = useState([]);
  const [recentActivity, setRecentActivity] = useState([]);

  useEffect(() => {
    loadDashboardData();
  }, []);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      
      // Load chatbots
      const chatbotsResponse = await apiClient.get('/api/v1/chatbots');
      const chatbotsData = await chatbotsResponse.json();
      setChatbots(chatbotsData);

      // Calculate stats
      const totalChatbots = chatbotsData.length;
      const activeChatbots = chatbotsData.filter(bot => bot.is_active).length;

      // Mock additional stats (in a real implementation, these would come from analytics endpoints)
      setStats({
        totalChatbots,
        activeChatbots,
        totalConversations: 247,
        leadsGenerated: 89
      });

      // Mock recent activity data
      setRecentActivity([
        { name: 'Mon', conversations: 45, leads: 12 },
        { name: 'Tue', conversations: 52, leads: 15 },
        { name: 'Wed', conversations: 38, leads: 8 },
        { name: 'Thu', conversations: 67, leads: 22 },
        { name: 'Fri', conversations: 45, leads: 18 },
        { name: 'Sat', conversations: 32, leads: 9 },
        { name: 'Sun', conversations: 28, leads: 5 }
      ]);

    } catch (error) {
      console.error('Failed to load dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="dashboard-loading">
        <div className="loading-spinner"></div>
        <p>Loading dashboard...</p>
      </div>
    );
  }

  return (
    <div className="dashboard">
      <div className="page-header">
        <h1 className="page-title">Dashboard</h1>
        <p className="page-subtitle">
          Monitor your AI chatbots performance and manage conversations
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-4 mb-6">
        <StatCard
          title="Total Chatbots"
          value={stats.totalChatbots}
          icon="🤖"
          change="+2 this month"
          changeType="positive"
        />
        <StatCard
          title="Active Chatbots"
          value={stats.activeChatbots}
          icon="✅"
          change={`${stats.activeChatbots}/${stats.totalChatbots} active`}
          changeType="neutral"
        />
        <StatCard
          title="Total Conversations"
          value={stats.totalConversations}
          icon="💬"
          change="+18% from last week"
          changeType="positive"
        />
        <StatCard
          title="Leads Generated"
          value={stats.leadsGenerated}
          icon="🎯"
          change="+12% conversion rate"
          changeType="positive"
        />
      </div>

      <div className="grid grid-cols-2 mb-6">
        {/* Activity Chart */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Weekly Activity</h2>
          </div>
          <div style={{ width: '100%', height: 300 }}>
            <ResponsiveContainer>
              <BarChart data={recentActivity}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="conversations" fill="#3b82f6" name="Conversations" />
                <Bar dataKey="leads" fill="#10b981" name="Leads" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Conversion Trend */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Conversion Trend</h2>
          </div>
          <div style={{ width: '100%', height: 300 }}>
            <ResponsiveContainer>
              <LineChart data={recentActivity}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip 
                  formatter={(value, name, props) => {
                    if (name === 'conversion') {
                      return [`${value.toFixed(1)}%`, 'Conversion Rate'];
                    }
                    return [value, name];
                  }}
                />
                <Line 
                  type="monotone" 
                  dataKey={(data) => ((data.leads / data.conversations) * 100).toFixed(1)} 
                  stroke="#8b5cf6" 
                  strokeWidth={3}
                  name="conversion"
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2">
        {/* Recent Chatbots */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Your Chatbots</h2>
            <Link to="/chatbots/new" className="btn btn-primary btn-sm">
              Create New
            </Link>
          </div>
          <div className="chatbot-list">
            {chatbots.length === 0 ? (
              <div className="empty-state">
                <div className="empty-icon">🤖</div>
                <h3>No chatbots yet</h3>
                <p>Create your first AI chatbot to get started</p>
                <Link to="/chatbots/new" className="btn btn-primary">
                  Create Chatbot
                </Link>
              </div>
            ) : (
              chatbots.slice(0, 5).map(chatbot => (
                <div key={chatbot.id} className="chatbot-item">
                  <div className="chatbot-info">
                    <div className="chatbot-name">{chatbot.name}</div>
                    <div className="chatbot-description">
                      {chatbot.description || 'No description'}
                    </div>
                  </div>
                  <div className="chatbot-status">
                    <span className={`badge ${chatbot.is_active ? 'badge-success' : 'badge-gray'}`}>
                      {chatbot.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </div>
                  <div className="chatbot-actions">
                    <Link 
                      to={`/chatbots/${chatbot.id}/test`} 
                      className="btn btn-outline btn-sm"
                    >
                      Test
                    </Link>
                    <Link 
                      to={`/chatbots/${chatbot.id}/analytics`} 
                      className="btn btn-primary btn-sm"
                    >
                      Analytics
                    </Link>
                  </div>
                </div>
              ))
            )}
            {chatbots.length > 5 && (
              <div className="view-all">
                <Link to="/chatbots" className="btn btn-outline">
                  View All Chatbots ({chatbots.length})
                </Link>
              </div>
            )}
          </div>
        </div>

        {/* Quick Actions */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Quick Actions</h2>
          </div>
          <div className="quick-actions">
            <Link to="/chatbots/new" className="quick-action">
              <div className="action-icon">🚀</div>
              <div className="action-content">
                <h3>Create New Chatbot</h3>
                <p>Set up a new AI-powered chatbot for your website</p>
              </div>
              <div className="action-arrow">→</div>
            </Link>
            
            <Link to="/chatbots" className="quick-action">
              <div className="action-icon">⚙️</div>
              <div className="action-content">
                <h3>Manage Chatbots</h3>
                <p>Edit configurations and view all your chatbots</p>
              </div>
              <div className="action-arrow">→</div>
            </Link>
            
            <div className="quick-action" onClick={() => window.open('/api/v1/chatbots', '_blank')}>
              <div className="action-icon">📋</div>
              <div className="action-content">
                <h3>API Documentation</h3>
                <p>Integrate chatbots with your applications</p>
              </div>
              <div className="action-arrow">→</div>
            </div>
            
            <div className="quick-action" onClick={() => alert('Widget generator coming soon!')}>
              <div className="action-icon">🔧</div>
              <div className="action-content">
                <h3>Widget Generator</h3>
                <p>Generate embed codes for your websites</p>
              </div>
              <div className="action-arrow">→</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({ title, value, icon, change, changeType }) {
  return (
    <div className="card stat-card">
      <div className="stat-header">
        <div className="stat-icon">{icon}</div>
        <div className="stat-value">{value}</div>
      </div>
      <div className="stat-title">{title}</div>
      {change && (
        <div className={`stat-change stat-change-${changeType}`}>
          {change}
        </div>
      )}
    </div>
  );
}

export default Dashboard;