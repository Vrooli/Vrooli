import React from 'react';
import { SparklesIcon } from '@heroicons/react/24/outline';

const ideas = [
  {
    id: 1,
    title: 'Adaptive Pricing Card',
    description: 'Highlights key plans and dynamically adjusts featured tier based on user intent.',
  },
  {
    id: 2,
    title: 'Guided Form Wizard',
    description: 'Multi-step form pattern with AI-generated helper copy and validation hints.',
  },
  {
    id: 3,
    title: 'Team Activity Timeline',
    description: 'Real-time collaboration timeline with semantic grouping and inline actions.',
  },
];

const AIGenerator: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="flex items-center bg-gray-800 border border-gray-700 rounded-lg p-6">
        <SparklesIcon className="h-12 w-12 text-purple-400 mr-4" />
        <div>
          <h1 className="text-2xl font-semibold text-white">AI Component Generator</h1>
          <p className="text-gray-300 mt-2 max-w-3xl">
            Describe the interface you need and the generator will draft components, tests,
            and documentation that follow system conventions.
          </p>
        </div>
      </div>

      <section className="grid grid-cols-1 lg:grid-cols-3 gap-6 bg-gray-800 border border-gray-700 rounded-lg p-6">
        <div className="lg:col-span-2 space-y-4">
          <label className="space-y-2">
            <span className="text-sm font-medium text-gray-200">Prompt</span>
            <textarea
              rows={6}
              placeholder="Describe the component, required behaviors, states, and accessibility needs."
              className="w-full rounded-md border border-gray-600 bg-gray-900 px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
            />
          </label>
          <button className="px-4 py-2 rounded-md bg-purple-600 hover:bg-purple-700 text-white font-medium">
            Generate component draft
          </button>
        </div>

        <aside className="space-y-4">
          <h2 className="text-lg font-semibold text-white">Inspiration</h2>
          <div className="space-y-3 text-sm text-gray-300">
            {ideas.map((idea) => (
              <article key={idea.id} className="rounded-md border border-gray-700 bg-gray-900 p-4">
                <h3 className="font-semibold text-white">{idea.title}</h3>
                <p className="mt-1 text-gray-400">{idea.description}</p>
              </article>
            ))}
          </div>
        </aside>
      </section>
    </div>
  );
};

export default AIGenerator;
