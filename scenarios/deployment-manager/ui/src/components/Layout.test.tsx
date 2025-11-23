import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { Layout } from './Layout';

describe('Layout', () => {
  it('renders the layout with header', () => {
    render(
      <BrowserRouter>
        <Layout>
          <div>Test Content</div>
        </Layout>
      </BrowserRouter>
    );

    // Check that the header title is present
    expect(screen.getByText('Deployment Manager')).toBeDefined();
  });

  it('renders navigation items', () => {
    render(
      <BrowserRouter>
        <Layout>
          <div>Test Content</div>
        </Layout>
      </BrowserRouter>
    );

    // Check that navigation items are present
    expect(screen.getByText('Dashboard')).toBeDefined();
    expect(screen.getByText('Profiles')).toBeDefined();
    expect(screen.getByText('Analyze')).toBeDefined();
    expect(screen.getByText('Deployments')).toBeDefined();
  });

  it('renders children content', () => {
    render(
      <BrowserRouter>
        <Layout>
          <div>Test Content</div>
        </Layout>
      </BrowserRouter>
    );

    // Check that children are rendered
    expect(screen.getByText('Test Content')).toBeDefined();
  });
});
