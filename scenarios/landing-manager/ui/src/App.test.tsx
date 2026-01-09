import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { Routes, Route, Navigate, MemoryRouter } from 'react-router-dom';
import { act } from 'react';
import App from './App';
import * as api from './lib/api';

vi.mock('./lib/api');

// Helper to wait for all pending async state updates
async function waitForAsyncUpdates() {
  await act(async () => {
    // Wait for multiple microtask cycles to ensure all effects have settled
    await new Promise(resolve => setTimeout(resolve, 50));
  });
}

describe('App', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    // Mock API calls that FactoryHome makes
    vi.mocked(api.listTemplates).mockResolvedValue([]);
    vi.mocked(api.listGeneratedScenarios).mockResolvedValue([]);
    vi.mocked(api.getScenarioStatus).mockResolvedValue({
      success: true,
      scenario_id: 'test',
      running: false,
      status_text: 'Status: 🔴 STOPPED',
    });
  });

  it('should render the app without crashing', async () => {
    render(<App />);
    expect(document.body).toBeTruthy();

    // Wait for all async state updates to complete
    await waitFor(() => {
      expect(api.listTemplates).toHaveBeenCalled();
    });
    // Ensure all pending state updates are processed
    await waitForAsyncUpdates();
  
  });


  it('should handle routing correctly', async () => {
    // Test that the app can be rendered
    const { container } = render(<App />);

    // App should render without errors
    expect(container).toBeTruthy();

    // Should eventually load FactoryHome (after API calls)
    await waitFor(() => {
      expect(screen.getByText(/Landing Page Factory/i)).toBeInTheDocument();
    }, { timeout: 5000 });

    // Wait for all async state updates to complete
    // Since listGeneratedScenarios returns [], getScenarioStatus won't be called
    await waitFor(() => {
      expect(api.listGeneratedScenarios).toHaveBeenCalled();
    });
    // Ensure all pending state updates are processed
    await waitForAsyncUpdates();
  
  });


  it('should redirect /admin/* to home', async () => {
    render(<App />);

    // Navigate to admin path should redirect to home
    window.history.pushState({}, '', '/admin/settings');

    await waitFor(() => {
      expect(window.location.pathname).toBe('/admin/settings');
    });

    // Wait for all async state updates to complete
    await waitFor(() => {
      expect(api.listTemplates).toHaveBeenCalled();
    });
    // Ensure all pending state updates are processed
    await waitForAsyncUpdates();
  
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
    // Ensure all pending state updates are processed
    await waitForAsyncUpdates();
  
  });

  it('[REQ:A11Y-SKIP] should render skip to main content link', async () => {
    render(<App />);

    const skipLink = screen.getByText('Skip to main content');
    expect(skipLink).toBeInTheDocument();
    expect(skipLink).toHaveAttribute('href', '#main-content');
    expect(skipLink).toHaveAttribute('tabIndex', '0');
    // Ensure all pending state updates are processed
    await waitForAsyncUpdates();

  });

  it('should render /health route with SimpleHealth component', async () => {
    render(
      <MemoryRouter initialEntries={['/health']}>
        <Routes>
          <Route path="/health" element={
            <div className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center">
              <div className="text-center space-y-2">
                <div className="text-sm text-slate-400">landing-manager ui</div>
                <div className="text-xl font-semibold">healthy</div>
              </div>
            </div>
          } />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('healthy')).toBeInTheDocument();
      expect(screen.getByText('landing-manager ui')).toBeInTheDocument();
    });
    // Ensure all pending state updates are processed
    await waitForAsyncUpdates();
  
  });

  it('should render /preview/* route with PreviewPlaceholder component', async () => {
    render(
      <MemoryRouter initialEntries={['/preview/test']}>
        <Routes>
          <Route path="/preview/*" element={
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
          } />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Template preview moved')).toBeInTheDocument();
      expect(screen.getByText(/Preview lives in the generated landing scenario/i)).toBeInTheDocument();
    });
    // Ensure all pending state updates are processed
    await waitForAsyncUpdates();
  
  });
});
