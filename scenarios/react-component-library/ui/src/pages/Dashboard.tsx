import React from 'react';
import { Link } from 'react-router-dom';
import {
  CubeIcon,
  SparklesIcon,
  ChartBarIcon,
  BeakerIcon,
  PlusIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline';

const Dashboard = () => {
  // Mock data - in real app would come from API
  const stats = {
    totalComponents: 23,
    passedTests: 19,
    failedTests: 4,
    aiGenerations: 12,
    recentActivity: [
      { id: 1, type: 'create', component: 'Button', time: '2 hours ago', status: 'success' },
      { id: 2, type: 'test', component: 'Modal', time: '3 hours ago', status: 'success' },
      { id: 3, type: 'generate', component: 'Card', time: '5 hours ago', status: 'success' },
      { id: 4, type: 'test', component: 'Dropdown', time: '1 day ago', status: 'failed' },
    ],
    popularComponents: [
      { id: 1, name: 'Button', usageCount: 156, score: 95 },
      { id: 2, name: 'Modal', usageCount: 89, score: 88 },
      { id: 3, name: 'Card', usageCount: 67, score: 92 },
      { id: 4, name: 'Input', usageCount: 54, score: 87 },
    ],
  };

  const quickActions = [
    {
      name: 'Create Component',
      description: 'Build a new component from scratch',
      href: '/create',
      icon: PlusIcon,
      color: 'bg-blue-600 hover:bg-blue-700',
    },
    {
      name: 'AI Generator',
      description: 'Generate components with AI',
      href: '/generate',
      icon: SparklesIcon,
      color: 'bg-purple-600 hover:bg-purple-700',
    },
    {
      name: 'Browse Library',
      description: 'Explore existing components',
      href: '/library',
      icon: CubeIcon,
      color: 'bg-green-600 hover:bg-green-700',
    },
    {
      name: 'Run Tests',
      description: 'Test component quality',
      href: '/testing',
      icon: BeakerIcon,
      color: 'bg-yellow-600 hover:bg-yellow-700',
    },
  ];

  return (
    <div className="space-y-8">
      {/* Welcome Section */}
      <div className="bg-gradient-to-r from-blue-600 to-purple-600 rounded-lg p-6">
        <h1 className="text-3xl font-bold text-white mb-2">
          Welcome to React Component Library
        </h1>
        <p className="text-blue-100 text-lg">
          Build, test, and share reusable React components with AI-powered tools
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
          <div className="flex items-center">
            <CubeIcon className="h-8 w-8 text-blue-400" />
            <div className="ml-4">
              <p className="text-sm text-gray-400">Total Components</p>
              <p className="text-3xl font-bold text-white">{stats.totalComponents}</p>
            </div>
          </div>
        </div>

        <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
          <div className="flex items-center">
            <CheckCircleIcon className="h-8 w-8 text-green-400" />
            <div className="ml-4">
              <p className="text-sm text-gray-400">Passed Tests</p>
              <p className="text-3xl font-bold text-white">{stats.passedTests}</p>
            </div>
          </div>
        </div>

        <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
          <div className="flex items-center">
            <ExclamationTriangleIcon className="h-8 w-8 text-red-400" />
            <div className="ml-4">
              <p className="text-sm text-gray-400">Failed Tests</p>
              <p className="text-3xl font-bold text-white">{stats.failedTests}</p>
            </div>
          </div>
        </div>

        <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
          <div className="flex items-center">
            <SparklesIcon className="h-8 w-8 text-purple-400" />
            <div className="ml-4">
              <p className="text-sm text-gray-400">AI Generations</p>
              <p className="text-3xl font-bold text-white">{stats.aiGenerations}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div>
        <h2 className="text-xl font-semibold text-white mb-4">Quick Actions</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {quickActions.map((action) => (
            <Link
              key={action.name}
              to={action.href}
              className={`${action.color} rounded-lg p-6 text-white transition-colors group`}
            >
              <action.icon className="h-8 w-8 mb-3" />
              <h3 className="text-lg font-semibold mb-2">{action.name}</h3>
              <p className="text-sm opacity-90">{action.description}</p>
            </Link>
          ))}
        </div>
      </div>

      {/* Two Column Layout for Activity and Popular */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Recent Activity */}
        <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
          <h2 className="text-lg font-semibold text-white mb-4">Recent Activity</h2>
          <div className="space-y-4">
            {stats.recentActivity.map((activity) => (
              <div key={activity.id} className="flex items-center space-x-3">
                <div className={`w-2 h-2 rounded-full ${
                  activity.status === 'success' ? 'bg-green-400' : 'bg-red-400'
                }`} />
                <div className="flex-1">
                  <p className="text-sm text-white">
                    {activity.type === 'create' && 'Created'}
                    {activity.type === 'test' && 'Tested'}
                    {activity.type === 'generate' && 'Generated'}
                    {' '}
                    <span className="font-medium text-blue-400">{activity.component}</span>
                  </p>
                  <p className="text-xs text-gray-400">{activity.time}</p>
                </div>
                {activity.status === 'success' ? (
                  <CheckCircleIcon className="h-4 w-4 text-green-400" />
                ) : (
                  <ExclamationTriangleIcon className="h-4 w-4 text-red-400" />
                )}
              </div>
            ))}
          </div>
          <Link
            to="/analytics"
            className="inline-flex items-center text-sm text-blue-400 hover:text-blue-300 mt-4"
          >
            View all activity
            <ChartBarIcon className="ml-1 h-4 w-4" />
          </Link>
        </div>

        {/* Popular Components */}
        <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
          <h2 className="text-lg font-semibold text-white mb-4">Popular Components</h2>
          <div className="space-y-4">
            {stats.popularComponents.map((component, index) => (
              <div key={component.id} className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="flex items-center justify-center w-6 h-6 bg-blue-600 rounded text-xs font-medium">
                    {index + 1}
                  </div>
                  <div>
                    <p className="text-sm font-medium text-white">{component.name}</p>
                    <p className="text-xs text-gray-400">{component.usageCount} uses</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-sm font-medium text-white">{component.score}%</p>
                  <p className="text-xs text-gray-400">Quality Score</p>
                </div>
              </div>
            ))}
          </div>
          <Link
            to="/library"
            className="inline-flex items-center text-sm text-blue-400 hover:text-blue-300 mt-4"
          >
            Browse all components
            <CubeIcon className="ml-1 h-4 w-4" />
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;