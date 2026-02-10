import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import TechStackTab from '@/components/tabs/TechStackTab';
import { createMockApp } from './fixtures';

describe('TechStackTab', () => {
  it('shows loading state', () => {
    render(<TechStackTab app={null} techStack={null} loading />);
    expect(screen.getByText('Loading tech stack information...')).toBeInTheDocument();
  });

  it('shows error with retry', () => {
    const onRetry = vi.fn();
    render(<TechStackTab app={createMockApp()} techStack={null} error="Failed" onRetry={onRetry} />);
    expect(screen.getByText('Failed')).toBeInTheDocument();
    expect(screen.getByText('Retry')).toBeInTheDocument();
  });

  it('renders tech stack components', () => {
    const app = createMockApp({ tech_stack: ['React', 'TypeScript', 'Vite'] });
    render(<TechStackTab app={app} techStack={null} />);
    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('TypeScript')).toBeInTheDocument();
    expect(screen.getByText('Vite')).toBeInTheDocument();
  });

  it('renders consolidated healthy badge', () => {
    const app = createMockApp({
      dependencies: [
        {
          name: 'postgres',
          type: 'database',
          required: true,
          enabled: true,
          status: 'running',
          running: true,
          healthy: true,
          installed: true,
        },
      ],
    });
    render(<TechStackTab app={app} techStack={null} />);
    expect(screen.getByText('Healthy')).toBeInTheDocument();
    // Should NOT show Required when running
    expect(screen.queryByText('Required')).not.toBeInTheDocument();
  });

  it('renders consolidated unhealthy + required badges', () => {
    const app = createMockApp({
      dependencies: [
        {
          name: 'redis',
          type: 'cache',
          required: true,
          enabled: true,
          status: 'stopped',
          running: false,
          healthy: false,
          installed: true,
        },
      ],
    });
    render(<TechStackTab app={app} techStack={null} />);
    expect(screen.getByText('Stopped')).toBeInTheDocument();
    expect(screen.getByText('Required')).toBeInTheDocument();
  });

  it('renders port allocations', () => {
    render(
      <TechStackTab
        app={createMockApp()}
        techStack={{ ports: { UI_PORT: 3000, API_PORT: 4000 }, tags: [] }}
      />,
    );
    expect(screen.getByText('UI_PORT')).toBeInTheDocument();
    expect(screen.getByText('3000')).toBeInTheDocument();
  });
});
