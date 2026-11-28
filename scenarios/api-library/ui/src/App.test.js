import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import axios from 'axios';
import App from './App';

jest.mock('axios', () => {
  const get = jest.fn().mockReturnValue({ data: [] });
  const post = jest.fn().mockReturnValue({ data: { results: [] } });
  const axios = { get, post };
  axios.default = axios;
  return { __esModule: true, default: axios, get, post };
});

jest.mock('@vrooli/api-base', () => ({
  resolveApiBase: () => '/api',
  buildApiUrl: (path, { baseUrl } = {}) => `${baseUrl || '/api'}${path}`,
}));

jest.mock('@vrooli/react-component-library/AppShell/2', () => {
  const React = require('react');
  return {
    AppShell: ({ brand, items, onNavigate, children }) => React.createElement(
      'div',
      { 'data-testid': 'app-shell' },
      React.createElement('div', { 'data-testid': 'app-shell-brand' }, brand),
      React.createElement(
        'nav',
        { 'aria-label': 'Application navigation' },
        items.map((item) => React.createElement(
          'a',
          {
            key: item.id,
            href: item.href,
            'aria-current': item.current ? 'page' : undefined,
            onClick: (event) => {
              event.preventDefault();
              onNavigate(item);
            },
          },
          item.label,
        )),
      ),
      children,
    ),
  };
}, { virtual: true });

describe('API Library shell', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    axios.get.mockReturnValue({ data: [] });
    axios.post.mockReturnValue({ data: { results: [] } });
  });

  test('mounts the governed shell and preserves tab navigation', () => {
    render(<App />);

    expect(screen.getByTestId('app-shell')).toBeInTheDocument();
    expect(screen.getByTestId('app-shell-brand')).toHaveTextContent('API Library');
    expect(screen.getByRole('link', { name: 'Search APIs' })).toHaveAttribute('href', '#search');
    expect(screen.getByRole('link', { name: 'Configured (0)' })).toHaveAttribute('href', '#configured');

    fireEvent.click(screen.getByRole('link', { name: 'Request Research' }));

    expect(screen.getByRole('heading', { name: 'Request API Research' })).toBeInTheDocument();
  });

  test('keeps search requests on the existing API contract', async () => {
    render(<App />);

    fireEvent.change(screen.getByPlaceholderText(/Search for APIs/i), {
      target: { value: 'send email' },
    });
    fireEvent.click(await screen.findByRole('button', { name: 'Search' }));

    expect(axios.post).toHaveBeenCalledWith('/api/search', {
      query: 'send email',
      limit: 20,
    });
  });
});
