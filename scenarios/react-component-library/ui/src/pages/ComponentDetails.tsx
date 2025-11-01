import React from 'react';
import { useParams, Link } from 'react-router-dom';
import { CheckCircleIcon, ClockIcon, WrenchIcon } from '@heroicons/react/24/outline';

const ComponentDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();

  const metadata = {
    stability: 'Stable',
    version: '1.4.0',
    owners: ['Design Systems Team', 'Frontend Guild'],
    tags: ['UI', 'Accessibility', 'Theming'],
  };

  const qualitySignals = [
    { id: 1, label: 'Accessibility audit', status: 'passing', timestamp: '2025-10-25' },
    { id: 2, label: 'Performance benchmark', status: 'passing', timestamp: '2025-10-21' },
    { id: 3, label: 'Visual regression', status: 'pending', timestamp: '2025-10-27' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between bg-gray-800 border border-gray-700 rounded-lg p-6">
        <div>
          <p className="text-sm text-gray-400 uppercase tracking-wide">Component</p>
          <h1 className="text-2xl font-semibold text-white">{id}</h1>
          <p className="text-gray-300 mt-2 max-w-2xl">
            Component profiles consolidate usage guidelines, code references, and testing
            signals so teams can confidently reuse shared UI.
          </p>
        </div>
        <Link
          to="/library"
          className="px-4 py-2 rounded-md bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium"
        >
          Back to Library
        </Link>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <section className="bg-gray-800 border border-gray-700 rounded-lg p-6">
            <h2 className="text-lg font-semibold text-white mb-4">Metadata</h2>
            <dl className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-gray-300">
              <div>
                <dt className="text-gray-400 uppercase text-xs tracking-wide">Stability</dt>
                <dd className="mt-1 font-medium text-white">{metadata.stability}</dd>
              </div>
              <div>
                <dt className="text-gray-400 uppercase text-xs tracking-wide">Current Version</dt>
                <dd className="mt-1 font-medium text-white">{metadata.version}</dd>
              </div>
              <div>
                <dt className="text-gray-400 uppercase text-xs tracking-wide">Owners</dt>
                <dd className="mt-1">{metadata.owners.join(', ')}</dd>
              </div>
              <div>
                <dt className="text-gray-400 uppercase text-xs tracking-wide">Tags</dt>
                <dd className="mt-1 flex flex-wrap gap-2">
                  {metadata.tags.map((tag) => (
                    <span
                      key={tag}
                      className="px-2 py-1 rounded-full bg-gray-700 text-gray-200 text-xs"
                    >
                      {tag}
                    </span>
                  ))}
                </dd>
              </div>
            </dl>
          </section>

          <section className="bg-gray-800 border border-gray-700 rounded-lg p-6">
            <h2 className="text-lg font-semibold text-white mb-4">Quality Signals</h2>
            <div className="space-y-4">
              {qualitySignals.map((signal) => (
                <div key={signal.id} className="flex items-start space-x-4">
                  {signal.status === 'passing' ? (
                    <CheckCircleIcon className="h-6 w-6 text-green-400" />
                  ) : (
                    <ClockIcon className="h-6 w-6 text-yellow-300" />
                  )}
                  <div>
                    <p className="text-sm font-medium text-white">{signal.label}</p>
                    <p className="text-xs text-gray-400">Last run {signal.timestamp}</p>
                  </div>
                </div>
              ))}
            </div>
          </section>
        </div>

        <section className="bg-gray-800 border border-gray-700 rounded-lg p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">Developer Resources</h2>
          <div className="space-y-3 text-sm text-gray-300">
            <p>
              <WrenchIcon className="inline h-5 w-5 text-blue-400 mr-2" />
              Install via `npm install @library/{id}`
            </p>
            <p>
              Copy ready-made Storybook stories, tests, and accessibility guides directly from the
              component repository.
            </p>
            <p className="text-xs text-gray-500">
              Detailed documentation is synced automatically from the design system repository.
            </p>
          </div>
        </section>
      </div>
    </div>
  );
};

export default ComponentDetails;
