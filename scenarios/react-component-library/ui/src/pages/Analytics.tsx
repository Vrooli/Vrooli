import React from 'react';
import { ChartBarIcon } from '@heroicons/react/24/outline';

const metrics = [
  { id: 1, label: 'Weekly component downloads', value: '1,482', trend: '+12%' },
  { id: 2, label: 'Design handoff compliance', value: '94%', trend: '+4%' },
  { id: 3, label: 'Average accessibility score', value: '98%', trend: '+2%' },
  { id: 4, label: 'Average performance rating', value: '93', trend: '+5' },
];

const Analytics: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="flex items-center bg-gray-800 border border-gray-700 rounded-lg p-6">
        <ChartBarIcon className="h-12 w-12 text-sky-400 mr-4" />
        <div>
          <h1 className="text-2xl font-semibold text-white">Library Analytics</h1>
          <p className="text-gray-300 mt-2 max-w-3xl">
            Monitor adoption, quality, and performance signals across the design system. Use
            these insights to prioritise improvements.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-6">
        {metrics.map((metric) => (
          <div key={metric.id} className="bg-gray-800 border border-gray-700 rounded-lg p-5">
            <p className="text-xs uppercase text-gray-400 tracking-wide">{metric.label}</p>
            <p className="text-3xl font-semibold text-white mt-2">{metric.value}</p>
            <p className="text-sm text-green-400 mt-1">{metric.trend} vs previous period</p>
          </div>
        ))}
      </div>

      <section className="bg-gray-800 border border-gray-700 rounded-lg p-6 space-y-4">
        <h2 className="text-lg font-semibold text-white">Adoption highlights</h2>
        <ul className="space-y-2 text-sm text-gray-300 list-disc list-inside">
          <li>Designers across 6 product pods reused shared components last week.</li>
          <li>AI-generated prototypes converted to production components in under 2 days.</li>
          <li>Performance budgets triggered alerts for 3 components, all resolved within 24 hours.</li>
        </ul>
      </section>
    </div>
  );
};

export default Analytics;
