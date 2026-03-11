// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-component-tests
// [REQ:RRV-UI-001] App component - Unit tests for main application component
import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import App from './App';
import { renderWithProviders, createMockHealthResponse } from './test-utils';

// Mock the api module
vi.mock('./lib/api', () => ({
  fetchHealth: vi.fn(),
}));

import { fetchHealth } from './lib/api';

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    beforeEach(() => {
      vi.mocked(fetchHealth).mockResolvedValue(createMockHealthResponse());
    });

    it('renders the header text', () => {
      renderWithProviders(<App />);
      expect(screen.getByText('Reference React Vite')).toBeInTheDocument();
    });

    it('renders the scenario template badge', () => {
      renderWithProviders(<App />);
      expect(screen.getByText('Scenario Template')).toBeInTheDocument();
    });

    it('renders the description text', () => {
      renderWithProviders(<App />);
      expect(screen.getByText(/This starter UI is intentionally minimal/)).toBeInTheDocument();
    });

    it('renders the API Health section', () => {
      renderWithProviders(<App />);
      expect(screen.getByText('API Health')).toBeInTheDocument();
    });

    it('renders the Refresh button', () => {
      renderWithProviders(<App />);
      expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
    });
  });

  describe('API health check', () => {
    it('shows loading state initially', () => {
      // Keep the promise pending
      vi.mocked(fetchHealth).mockImplementation(() => new Promise(() => {}));

      renderWithProviders(<App />);
      expect(screen.getByText(/Checking API status/i)).toBeInTheDocument();
    });

    it('displays health status on success', async () => {
      vi.mocked(fetchHealth).mockResolvedValue(createMockHealthResponse({
        status: 'healthy',
        service: 'test-service',
      }));

      renderWithProviders(<App />);

      await waitFor(() => {
        expect(screen.getByText('Status: healthy')).toBeInTheDocument();
      });
      expect(screen.getByText('Service: test-service')).toBeInTheDocument();
    });

    it('displays error message on API failure', async () => {
      vi.mocked(fetchHealth).mockRejectedValue(new Error('Network error'));

      renderWithProviders(<App />);

      await waitFor(() => {
        expect(screen.getByText(/Unable to reach the API/i)).toBeInTheDocument();
      });
    });
  });
});
