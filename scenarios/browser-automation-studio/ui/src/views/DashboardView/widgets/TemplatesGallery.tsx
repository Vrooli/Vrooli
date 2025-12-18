import React, { useState } from 'react';

interface Template {
  id: string;
  name: string;
  description: string;
  category: 'testing' | 'scraping' | 'forms' | 'monitoring';
  icon: React.ReactNode;
  prompt: string;
}

interface TemplatesGalleryProps {
  onUseTemplate: (prompt: string, templateName: string) => void;
}

const templates: Template[] = [
  {
    id: 'login-test',
    name: 'Login Flow Test',
    description: 'Test a login form with username and password',
    category: 'testing',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
    ),
    prompt: 'Navigate to the login page, enter a test username and password, click the login button, and verify that the user is logged in successfully by checking for a dashboard or welcome message.',
  },
  {
    id: 'screenshot-pages',
    name: 'Screenshot Multiple Pages',
    description: 'Capture screenshots of different pages',
    category: 'monitoring',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
    prompt: 'Navigate to the homepage, take a full-page screenshot. Then navigate to the about page and take another screenshot. Finally, navigate to the contact page and take a screenshot.',
  },
  {
    id: 'form-submit',
    name: 'Form Submission',
    description: 'Fill out and submit a web form',
    category: 'forms',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    ),
    prompt: 'Navigate to a contact form page, fill in the name field with "Test User", email with "test@example.com", and message with "This is a test message". Submit the form and verify success.',
  },
  {
    id: 'price-scraper',
    name: 'Price Scraper',
    description: 'Extract product prices from a page',
    category: 'scraping',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    prompt: 'Navigate to an e-commerce product listing page, extract the product names and prices from the visible items. For each product, capture the title, price, and any discount information.',
  },
  {
    id: 'status-check',
    name: 'Website Status Check',
    description: 'Verify pages are loading correctly',
    category: 'monitoring',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    prompt: 'Navigate to the homepage and verify it loads within 5 seconds. Check that the main navigation is visible. Take a screenshot. Assert that key elements like the logo and menu are present.',
  },
  {
    id: 'data-extraction',
    name: 'Table Data Extraction',
    description: 'Extract data from HTML tables',
    category: 'scraping',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
      </svg>
    ),
    prompt: 'Navigate to a page with a data table, extract all rows from the table including headers. For each row, capture all cell values and structure them as key-value pairs based on the headers.',
  },
];

const categoryColors = {
  testing: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  scraping: 'bg-green-500/20 text-green-400 border-green-500/30',
  forms: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
  monitoring: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
};

const categoryLabels = {
  testing: 'Testing',
  scraping: 'Scraping',
  forms: 'Forms',
  monitoring: 'Monitoring',
};

export const TemplatesGallery: React.FC<TemplatesGalleryProps> = ({ onUseTemplate }) => {
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);

  const filteredTemplates = selectedCategory
    ? templates.filter((t) => t.category === selectedCategory)
    : templates;

  const categories = ['testing', 'scraping', 'forms', 'monitoring'] as const;

  return (
    <div className="bg-flow-surface border border-flow-border rounded-lg p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-flow-text-secondary">Templates</h3>
      </div>

      {/* Category filters */}
      <div className="flex flex-wrap gap-2 mb-4">
        <button
          onClick={() => setSelectedCategory(null)}
          className={`px-2.5 py-1 text-xs rounded-full border transition-colors ${
            selectedCategory === null
              ? 'bg-flow-accent text-white border-flow-accent'
              : 'text-flow-text-secondary border-flow-border hover:border-flow-border-light hover:text-surface'
          }`}
        >
          All
        </button>
        {categories.map((cat) => (
          <button
            key={cat}
            onClick={() => setSelectedCategory(cat === selectedCategory ? null : cat)}
            className={`px-2.5 py-1 text-xs rounded-full border transition-colors ${
              selectedCategory === cat
                ? categoryColors[cat]
                : 'text-flow-text-secondary border-flow-border hover:border-flow-border-light hover:text-surface'
            }`}
          >
            {categoryLabels[cat]}
          </button>
        ))}
      </div>

      {/* Templates grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {filteredTemplates.map((template) => (
          <div
            key={template.id}
            onClick={() => onUseTemplate(template.prompt, template.name)}
            className="group flex items-start gap-3 p-3 bg-flow-node/50 hover:bg-flow-node-hover border border-flow-border/50 hover:border-flow-border rounded-lg cursor-pointer transition-all"
          >
            <div className={`flex items-center justify-center w-9 h-9 rounded-lg ${categoryColors[template.category].split(' ')[0]} ${categoryColors[template.category].split(' ')[1]}`}>
              {template.icon}
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-sm text-surface font-medium truncate group-hover:text-blue-300 transition-colors">
                {template.name}
              </div>
              <div className="text-xs text-flow-text-muted line-clamp-2">
                {template.description}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
