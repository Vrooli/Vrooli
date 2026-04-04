import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { Layout } from './Layout';

describe('Layout', () => {
  it('renders the layout with header', () => {
    render(
      <MemoryRouter>
        <Layout>
          <div>Test Content</div>
        </Layout>
      </MemoryRouter>
    );

    // Check that the header title is present
    expect(screen.getByText('Deployment Manager')).toBeDefined();
  });

  it('renders navigation items', () => {
    render(
      <MemoryRouter>
        <Layout>
          <div>Test Content</div>
        </Layout>
      </MemoryRouter>
    );

    // Check that navigation items are present
    expect(screen.getByText('Dashboard')).toBeDefined();
    expect(screen.getByText('Profiles')).toBeDefined();
    expect(screen.getByText('Analyze')).toBeDefined();
    expect(screen.getByText('Deployments')).toBeDefined();
  });

  it('renders children content', () => {
    render(
      <MemoryRouter>
        <Layout>
          <div>Test Content</div>
        </Layout>
      </MemoryRouter>
    );

    // Check that children are rendered
    expect(screen.getByText('Test Content')).toBeDefined();
  });
});
