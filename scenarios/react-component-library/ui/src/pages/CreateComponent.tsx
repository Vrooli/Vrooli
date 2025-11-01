import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { SparklesIcon } from '@heroicons/react/24/outline';

const defaultForm = {
  name: '',
  category: 'Forms',
  description: '',
  includeTests: true,
  includeDocs: true,
};

type ComponentDraft = typeof defaultForm;

const CreateComponent: React.FC = () => {
  const navigate = useNavigate();
  const [form, setForm] = useState<ComponentDraft>(defaultForm);
  const [submitting, setSubmitting] = useState(false);

  const handleChange = (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => {
    const target = event.target;
    const { name, value, type } = target;
    const nextValue = type === 'checkbox' ? (target as HTMLInputElement).checked : value;

    setForm((prev) => ({
      ...prev,
      [name]: nextValue,
    }));
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    setTimeout(() => {
      navigate('/library', { replace: true });
    }, 800);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between bg-gray-800 border border-gray-700 rounded-lg p-6">
        <div>
          <h1 className="text-2xl font-semibold text-white">Create Component</h1>
          <p className="text-gray-300 mt-2 max-w-3xl">
            Scaffold new components with documentation, testing harnesses, and Storybook files.
            The generator keeps shared conventions aligned across product teams.
          </p>
        </div>
        <SparklesIcon className="h-12 w-12 text-purple-400" />
      </div>

      <form onSubmit={handleSubmit} className="bg-gray-800 border border-gray-700 rounded-lg p-6 space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <label className="space-y-2">
            <span className="text-sm font-medium text-gray-200">Component name</span>
            <input
              required
              name="name"
              value={form.name}
              onChange={handleChange}
              placeholder="e.g. InputField"
              className="w-full rounded-md border border-gray-600 bg-gray-900 px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
            />
          </label>

          <label className="space-y-2">
            <span className="text-sm font-medium text-gray-200">Category</span>
            <select
              name="category"
              value={form.category}
              onChange={handleChange}
              className="w-full rounded-md border border-gray-600 bg-gray-900 px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
            >
              <option>Forms</option>
              <option>Layout</option>
              <option>Feedback</option>
              <option>Data display</option>
            </select>
          </label>
        </div>

        <label className="space-y-2">
          <span className="text-sm font-medium text-gray-200">Description</span>
          <textarea
            name="description"
            rows={4}
            value={form.description}
            onChange={handleChange}
            placeholder="Describe the component's responsibilities and best practices"
            className="w-full rounded-md border border-gray-600 bg-gray-900 px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
          />
        </label>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <label className="inline-flex items-center space-x-3 text-sm text-gray-200">
            <input
              type="checkbox"
              name="includeTests"
              checked={form.includeTests}
              onChange={handleChange}
              className="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500 focus:ring-blue-500"
            />
            <span>Generate unit and accessibility tests</span>
          </label>

          <label className="inline-flex items-center space-x-3 text-sm text-gray-200">
            <input
              type="checkbox"
              name="includeDocs"
              checked={form.includeDocs}
              onChange={handleChange}
              className="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500 focus:ring-blue-500"
            />
            <span>Include documentation template</span>
          </label>
        </div>

        <button
          type="submit"
          disabled={submitting}
          className="px-4 py-2 rounded-md bg-blue-600 hover:bg-blue-700 disabled:opacity-70 text-white font-medium"
        >
          {submitting ? 'Creating…' : 'Create component'}
        </button>
      </form>
    </div>
  );
};

export default CreateComponent;
