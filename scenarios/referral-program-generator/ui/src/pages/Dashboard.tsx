import React from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { 
  Target, 
  Zap, 
  BarChart3, 
  DollarSign, 
  Users, 
  TrendingUp,
  ArrowRight,
  Sparkles,
  Settings,
  CheckCircle
} from 'lucide-react';
import { api } from '../lib/api';
import './Dashboard.css';

interface DashboardStats {
  totalPrograms: number;
  totalRevenue: number;
  activeLinks: number;
  conversionRate: number;
  recentPrograms: Array<{
    id: string;
    scenario_name: string;
    commission_rate: number;
    created_at: string;
  }>;
}

export function Dashboard() {
  const { data: stats, isLoading } = useQuery<DashboardStats>({
    queryKey: ['dashboard-stats'],
    queryFn: () => api.get('/api/v1/referral/dashboard/stats').then(res => res.data),
  });

  const features = [
    {
      icon: Target,
      title: 'Smart Analysis',
      description: 'Automatically extract branding, pricing, and structure from any scenario',
      action: 'Analyze Scenario',
      href: '/analyze',
      color: 'var(--color-primary)',
    },
    {
      icon: Zap,
      title: 'Instant Generation',
      description: 'Generate complete referral programs with optimized commission rates',
      action: 'Generate Program',
      href: '/generate',
      color: 'var(--color-secondary)',
    },
    {
      icon: Settings,
      title: 'Auto Implementation',
      description: 'Deploy referral logic directly into scenarios using Claude Code',
      action: 'Implement Now',
      href: '/implement',
      color: 'var(--color-accent)',
    },
    {
      icon: BarChart3,
      title: 'Performance Tracking',
      description: 'Monitor referral performance and optimize conversion rates',
      action: 'View Analytics',
      href: '/analytics',
      color: 'var(--color-info)',
    },
  ];

  const benefits = [
    '🚀 Transform any scenario into a revenue-generating business',
    '🎨 AI-powered branding analysis and matching',
    '💰 Optimized commission structures for maximum profitability',
    '🤖 Automated implementation with zero manual coding',
    '📊 Real-time analytics and performance optimization',
    '🔄 Cross-scenario referral networks for compound growth',
  ];

  return (
    <div className="dashboard">
      {/* Hero Section */}
      <section className="hero">
        <div className="hero-content">
          <div className="hero-badge">
            <Sparkles size={16} />
            <span>Permanent Business Intelligence</span>
          </div>
          <h1 className="hero-title">
            Turn Every Scenario Into a 
            <span className="text-primary"> Revenue Engine</span>
          </h1>
          <p className="hero-description">
            Generate intelligent referral programs that automatically adapt to your scenario's 
            branding, analyze optimal commission rates, and implement complete affiliate systems 
            with just a few clicks.
          </p>
          <div className="hero-actions">
            <Link to="/analyze" className="btn btn-primary btn-lg">
              <Target size={20} />
              Start Analysis
            </Link>
            <Link to="/generate" className="btn btn-outline btn-lg">
              <Zap size={20} />
              Generate Program
            </Link>
          </div>
        </div>
        
        <div className="hero-visual">
          <div className="floating-card">
            <div className="card-icon">
              <DollarSign size={24} />
            </div>
            <div className="card-content">
              <div className="metric-value">$47,892</div>
              <div className="metric-label">Revenue Generated</div>
            </div>
          </div>
          <div className="floating-card delay-1">
            <div className="card-icon">
              <TrendingUp size={24} />
            </div>
            <div className="card-content">
              <div className="metric-value">127%</div>
              <div className="metric-label">Growth Rate</div>
            </div>
          </div>
          <div className="floating-card delay-2">
            <div className="card-icon">
              <Users size={24} />
            </div>
            <div className="card-content">
              <div className="metric-value">2,341</div>
              <div className="metric-label">Active Referrals</div>
            </div>
          </div>
        </div>
      </section>

      {/* Stats Section */}
      {!isLoading && stats && (
        <section className="stats-section">
          <div className="stats-grid">
            <div className="stat-card">
              <div className="stat-icon">
                <Target className="text-primary" />
              </div>
              <div className="stat-content">
                <div className="stat-value">{stats.totalPrograms}</div>
                <div className="stat-label">Active Programs</div>
              </div>
            </div>
            <div className="stat-card">
              <div className="stat-icon">
                <DollarSign className="text-success" />
              </div>
              <div className="stat-content">
                <div className="stat-value">${stats.totalRevenue.toLocaleString()}</div>
                <div className="stat-label">Total Revenue</div>
              </div>
            </div>
            <div className="stat-card">
              <div className="stat-icon">
                <Users className="text-accent" />
              </div>
              <div className="stat-content">
                <div className="stat-value">{stats.activeLinks}</div>
                <div className="stat-label">Active Links</div>
              </div>
            </div>
            <div className="stat-card">
              <div className="stat-icon">
                <TrendingUp className="text-info" />
              </div>
              <div className="stat-content">
                <div className="stat-value">{(stats.conversionRate * 100).toFixed(1)}%</div>
                <div className="stat-label">Conversion Rate</div>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Features Section */}
      <section className="features-section">
        <div className="section-header">
          <h2 className="section-title">Powerful Features</h2>
          <p className="section-description">
            Everything you need to create, deploy, and optimize referral programs
          </p>
        </div>
        
        <div className="features-grid">
          {features.map((feature, index) => {
            const Icon = feature.icon;
            return (
              <div key={index} className="feature-card">
                <div className="feature-icon" style={{ color: feature.color }}>
                  <Icon size={32} />
                </div>
                <div className="feature-content">
                  <h3 className="feature-title">{feature.title}</h3>
                  <p className="feature-description">{feature.description}</p>
                  <Link to={feature.href} className="feature-action">
                    {feature.action}
                    <ArrowRight size={16} />
                  </Link>
                </div>
              </div>
            );
          })}
        </div>
      </section>

      {/* Benefits Section */}
      <section className="benefits-section">
        <div className="benefits-content">
          <div className="benefits-text">
            <h2 className="benefits-title">
              Why Choose Our Referral Generator?
            </h2>
            <p className="benefits-description">
              Built specifically for the Vrooli ecosystem, our referral program generator 
              understands scenario architecture and automatically creates optimized 
              monetization strategies.
            </p>
            <ul className="benefits-list">
              {benefits.map((benefit, index) => (
                <li key={index} className="benefit-item">
                  <CheckCircle size={20} className="benefit-check" />
                  <span>{benefit}</span>
                </li>
              ))}
            </ul>
          </div>
          
          <div className="benefits-visual">
            <div className="process-flow">
              <div className="process-step">
                <div className="step-number">1</div>
                <div className="step-content">
                  <Target size={20} />
                  <span>Analyze</span>
                </div>
              </div>
              <ArrowRight className="process-arrow" />
              <div className="process-step">
                <div className="step-number">2</div>
                <div className="step-content">
                  <Zap size={20} />
                  <span>Generate</span>
                </div>
              </div>
              <ArrowRight className="process-arrow" />
              <div className="process-step">
                <div className="step-number">3</div>
                <div className="step-content">
                  <Settings size={20} />
                  <span>Deploy</span>
                </div>
              </div>
              <ArrowRight className="process-arrow" />
              <div className="process-step">
                <div className="step-number">4</div>
                <div className="step-content">
                  <DollarSign size={20} />
                  <span>Earn</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Recent Programs */}
      {!isLoading && stats?.recentPrograms && stats.recentPrograms.length > 0 && (
        <section className="recent-section">
          <div className="section-header">
            <h2 className="section-title">Recent Programs</h2>
            <Link to="/analytics" className="section-action">
              View All <ArrowRight size={16} />
            </Link>
          </div>
          
          <div className="recent-grid">
            {stats.recentPrograms.map((program) => (
              <div key={program.id} className="recent-card">
                <div className="recent-header">
                  <h3 className="recent-title">{program.scenario_name}</h3>
                  <div className="recent-rate">
                    {(program.commission_rate * 100).toFixed(0)}% commission
                  </div>
                </div>
                <div className="recent-meta">
                  <span className="recent-date">
                    Created {new Date(program.created_at).toLocaleDateString()}
                  </span>
                  <div className="status-indicator status-success">
                    <span>Active</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* CTA Section */}
      <section className="cta-section">
        <div className="cta-content">
          <h2 className="cta-title">Ready to Monetize Your Scenarios?</h2>
          <p className="cta-description">
            Start generating revenue from your Vrooli scenarios in minutes, not days.
          </p>
          <div className="cta-actions">
            <Link to="/analyze" className="btn btn-primary btn-lg">
              <Sparkles size={20} />
              Get Started Now
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}