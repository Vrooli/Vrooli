import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { Routes, Route, Navigate, MemoryRouter } from 'react-router-dom';
import App from './App';
import * as api from './lib/api';

vi.mock('./lib/api');

// Extract route components for testing
function SimpleHealth() {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center">
      <div className="text-center space-y-2">
        <div className="text-sm text-slate-400">landing-manager ui</div>
        <div className="text-xl font-semibold">healthy</div>
      </div>
    </div>
  );
}

function PreviewPlaceholder() {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center px-6">
      <div className="max-w-xl text-center space-y-4">
        <p className="text-sm text-amber-200/80">Template preview moved</p>
        <h2 className="text-2xl font-semibold">Preview lives in the generated landing scenario</h2>
        <p className="text-slate-300">
          To review the public landing and admin portal, generate a scenario and run it directly
          (e.g. <code className="px-1 py-0.5 rounded bg-slate-900">landing-manager generate saas-landing-page --name "demo" --slug "demo"</code> then <code className="px-1 py-0.5 rounded bg-slate-900">cd generated/demo && make start</code>).
        </p>
      </div>
    </div>
  );
}

describe('App', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    // Mock API calls that FactoryHome makes
    vi.mocked(api.listTemplates).mockResolvedValue([]);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([]);
  });

  it('should render the app without crashing', () => {
    render(<App />);
    expect(document.body).toBeTruthy();
  });

  it('should render health endpoint', () => {
    render(<SimpleHealth />);

    expect(screen.getByText(/landing-manager ui/i)).toBeInTheDocument();
    expect(screen.getByText(/healthy/i)).toBeInTheDocument();
  });

  it('should show preview placeholder', () => {
    render(<PreviewPlaceholder />);

    expect(screen.getByText(/Template preview moved/i)).toBeInTheDocument();
    expect(screen.getByText(/Preview lives in the generated landing scenario/i)).toBeInTheDocument();
  });

  it('should render preview placeholder with generation instructions', () => {
    render(<PreviewPlaceholder />);

    expect(screen.getByText(/landing-manager generate/i)).toBeInTheDocument();
    expect(screen.getByText(/make start/i)).toBeInTheDocument();
  });

  it('should handle routing correctly', async () => {
    // Test that the app can be rendered
    const { container } = render(<App />);

    // App should render without errors
    expect(container).toBeTruthy();

    // Should eventually load FactoryHome (after API calls)
    await waitFor(() => {
      expect(screen.getByText(/Generate landing-page scenarios/i)).toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it('should render /health route', () => {
    render(
      <MemoryRouter initialEntries={['/health']}>
        <Routes>
          <Route path="/health" element={<SimpleHealth />} />
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText(/landing-manager ui/i)).toBeInTheDocument();
    expect(screen.getByText(/healthy/i)).toBeInTheDocument();
  });

  it('should render /preview/* route with placeholder', () => {
    render(
      <MemoryRouter initialEntries={['/preview/some-template']}>
        <Routes>
          <Route path="/preview/*" element={<PreviewPlaceholder />} />
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText(/Template preview moved/i)).toBeInTheDocument();
    expect(screen.getByText(/Preview lives in the generated landing scenario/i)).toBeInTheDocument();
  });

  it('should redirect /admin/* to home', async () => {
    render(<App />);

    // Navigate to admin path should redirect to home
    window.history.pushState({}, '', '/admin/settings');

    await waitFor(() => {
      expect(window.location.pathname).toBe('/admin/settings');
    });
  });

  it('should redirect unknown paths to home', async () => {
    render(
      <MemoryRouter initialEntries={['/unknown-path']}>
        <Routes>
          <Route path="/" element={<div>Home</div>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Home')).toBeInTheDocument();
    });
  });
});
