import React from 'react';

export interface TemplateField {
  /** Field key used in placeholder replacement (e.g., 'username' for {{username}}) */
  key: string;
  /** Display label for the input */
  label: string;
  /** Placeholder text for the input */
  placeholder: string;
  /** Whether the field is required */
  required: boolean;
  /** Default value if not provided */
  defaultValue?: string;
  /** Input type: 'text' for single line, 'textarea' for multi-line */
  type?: 'text' | 'textarea';
}

export interface Template {
  id: string;
  name: string;
  description: string;
  category: 'testing' | 'scraping' | 'forms' | 'monitoring';
  icon: React.ReactNode;
  /** Prompt template with placeholders: {{url}}, {{fieldKey}}, etc. */
  promptTemplate: string;
  /** Fields to collect from user (URL is always collected separately) */
  fields: TemplateField[];
}

export const templates: Template[] = [
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
    promptTemplate: 'Navigate to {{url}}, find the login form, enter username "{{username}}" and password "{{password}}", click the login button, and verify successful authentication by checking for a dashboard or welcome message.',
    fields: [
      {
        key: 'username',
        label: 'Username',
        placeholder: 'e.g., testuser',
        required: true,
      },
      {
        key: 'password',
        label: 'Password',
        placeholder: 'e.g., testpass123',
        required: true,
      },
    ],
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
    promptTemplate: 'Navigate to {{url}}, take a full-page screenshot. Then navigate to each of these pages and take screenshots: {{pages}}.',
    fields: [
      {
        key: 'pages',
        label: 'Pages to screenshot',
        placeholder: 'e.g., /about, /pricing, /contact',
        required: true,
      },
    ],
  },
  {
    id: 'form-fill',
    name: 'Form Submission',
    description: 'Fill out and submit a form',
    category: 'forms',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    ),
    promptTemplate: 'Navigate to {{url}} and fill out the form with the following data: {{formData}}. Then submit the form and verify submission success.',
    fields: [
      {
        key: 'formData',
        label: 'Form data (JSON)',
        placeholder: '{"name": "John", "email": "john@example.com"}',
        required: true,
        type: 'textarea',
      },
    ],
  },
  {
    id: 'price-monitor',
    name: 'Price Monitor',
    description: 'Check product price and alert on changes',
    category: 'monitoring',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 1.343-3 3 0 .656.21 1.264.567 1.764L9 16h6l-.567-3.236A2.983 2.983 0 0015 11c0-1.657-1.343-3-3-3z" />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 2v2m0 16v2m8-10h2M2 12H4m15.364-6.364l1.414 1.414M5.222 18.778l1.414-1.414m0-11.314L5.222 5.222m13.142 13.142l-1.414-1.414" />
      </svg>
    ),
    promptTemplate: 'Navigate to {{url}} and find the current price for the product. Compare it to the target price {{targetPrice}} and report if it is lower.',
    fields: [
      {
        key: 'targetPrice',
        label: 'Target price',
        placeholder: 'e.g., $99.99',
        required: true,
      },
    ],
  },
];
