import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TemplateGrid } from '../TemplateGrid';

// Mock the API module
vi.mock('../../lib/api', () => ({
  fetchTemplates: vi.fn(() => Promise.resolve({
    templates: [
      {
        name: 'Basic Desktop App',
        type: 'basic',
        description: 'Simple desktop application',
        complexity: 'low',
        features: ['Native menus', 'Auto-updater']
      },
      {
        name: 'Advanced Desktop App',
        type: 'advanced',
        description: 'Full-featured desktop application',
        complexity: 'medium',
        features: ['System tray', 'Global shortcuts']
      }
    ]
  }))
}));

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});

const renderWithClient = (component: React.ReactElement) => {
  return render(
    <QueryClientProvider client={queryClient}>
      {component}
    </QueryClientProvider>
  );
};

describe('TemplateGrid', () => {
  it('renders template cards', async () => {
    const mockOnSelect = vi.fn();

    renderWithClient(
      <TemplateGrid selectedTemplate="basic" onSelect={mockOnSelect} />
    );

    await waitFor(() => {
      expect(screen.getByText('Basic Desktop App')).toBeInTheDocument();
      expect(screen.getByText('Advanced Desktop App')).toBeInTheDocument();
    });
  });

  it('displays template descriptions', async () => {
    const mockOnSelect = vi.fn();

    renderWithClient(
      <TemplateGrid selectedTemplate="basic" onSelect={mockOnSelect} />
    );

    await waitFor(() => {
      expect(screen.getByText(/Simple desktop application/i)).toBeInTheDocument();
      expect(screen.getByText(/Full-featured desktop application/i)).toBeInTheDocument();
    });
  });

  it('shows feature badges', async () => {
    const mockOnSelect = vi.fn();

    renderWithClient(
      <TemplateGrid selectedTemplate="basic" onSelect={mockOnSelect} />
    );

    await waitFor(() => {
      expect(screen.getByText('Native menus')).toBeInTheDocument();
      expect(screen.getByText('System tray')).toBeInTheDocument();
    });
  });
});
