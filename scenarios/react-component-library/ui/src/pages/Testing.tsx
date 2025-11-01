import React from 'react';
import { CheckCircleIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline';

const Testing: React.FC = () => {
  const suites = [
    {
      id: 'accessibility',
      name: 'Accessibility Suite',
      description: 'Runs axe-core on all interactive states and reports WCAG regressions.',
      status: 'passing',
      lastRun: '2025-10-26 14:32',
    },
    {
      id: 'visual',
      name: 'Visual Regression',
      description: 'Captures component snapshots and compares across baselines.',
      status: 'failing',
      lastRun: '2025-10-24 09:18',
    },
    {
      id: 'performance',
      name: 'Performance Benchmarks',
      description: 'Records interaction timings and bundle size thresholds.',
      status: 'passing',
      lastRun: '2025-10-25 11:05',
    },
  ];

  return (
    <div className="space-y-6">
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
        <h1 className="text-2xl font-semibold text-white">Testing Workbench</h1>
        <p className="text-gray-300 mt-2 max-w-3xl">
          Run unified quality suites for every component. Results are streamed back to this
          dashboard so designers and engineers stay aligned.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {suites.map((suite) => (
          <article key={suite.id} className="bg-gray-800 border border-gray-700 rounded-lg p-6 space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">{suite.name}</h2>
              {suite.status === 'passing' ? (
                <CheckCircleIcon className="h-6 w-6 text-green-400" />
              ) : (
                <ExclamationTriangleIcon className="h-6 w-6 text-yellow-400" />
              )}
            </div>
            <p className="text-sm text-gray-300">{suite.description}</p>
            <p className="text-xs text-gray-400">Last run: {suite.lastRun}</p>
            <button className="px-4 py-2 rounded-md bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium">
              Run suite
            </button>
          </article>
        ))}
      </div>
    </div>
  );
};

export default Testing;
