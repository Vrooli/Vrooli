import React from 'react';
import { Link } from 'react-router-dom';

const mockComponents = [
  {
    id: 'button',
    name: 'Button',
    description: 'Primary action button with multiple variants',
    status: 'stable',
    version: '1.4.0',
    lastUpdated: '2025-10-21',
  },
  {
    id: 'modal',
    name: 'Modal',
    description: 'Accessible modal dialog with focus trapping',
    status: 'beta',
    version: '1.1.2',
    lastUpdated: '2025-10-16',
  },
  {
    id: 'card',
    name: 'Card',
    description: 'Flexible card layout with slots for media and actions',
    status: 'stable',
    version: '2.0.0',
    lastUpdated: '2025-10-01',
  },
];

const statusBadgeStyles: Record<string, string> = {
  stable: 'bg-green-900/60 text-green-300 border border-green-700/60',
  beta: 'bg-yellow-900/60 text-yellow-300 border border-yellow-700/60',
  experimental: 'bg-purple-900/60 text-purple-300 border border-purple-700/60',
};

const ComponentLibrary: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
        <h1 className="text-2xl font-semibold text-white">Component Library</h1>
        <p className="text-gray-300 mt-2 max-w-3xl">
          Browse reusable components curated by the React Component Library team. Each
          entry includes implementation notes, recommended usage patterns, and recent
          quality signals. Select a component to inspect details.
        </p>
      </div>

      <div className="bg-gray-800 rounded-lg border border-gray-700">
        <div className="grid grid-cols-12 gap-4 px-6 py-4 text-xs uppercase tracking-wide text-gray-400">
          <span className="col-span-3">Component</span>
          <span className="col-span-5">Description</span>
          <span className="col-span-2">Status</span>
          <span className="col-span-2 text-right">Last Updated</span>
        </div>
        <div className="divide-y divide-gray-700/60">
          {mockComponents.map((component) => (
            <Link
              key={component.id}
              to={`/component/${component.id}`}
              className="grid grid-cols-12 gap-4 px-6 py-5 hover:bg-gray-700/40 transition-colors"
            >
              <div className="col-span-3">
                <p className="text-white font-medium">{component.name}</p>
                <p className="text-xs text-gray-400">v{component.version}</p>
              </div>
              <p className="col-span-5 text-sm text-gray-300">{component.description}</p>
              <div className="col-span-2">
                <span
                  className={`text-xs font-medium px-3 py-1 rounded-full inline-flex ${
                    statusBadgeStyles[component.status] || statusBadgeStyles.experimental
                  }`}
                >
                  {component.status.toUpperCase()}
                </span>
              </div>
              <p className="col-span-2 text-right text-sm text-gray-400">
                {component.lastUpdated}
              </p>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
};

export default ComponentLibrary;
