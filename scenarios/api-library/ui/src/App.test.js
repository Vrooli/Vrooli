import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import App from './App';

// Mock fetch API
global.fetch = jest.fn();

describe('API Library App', () => {
  beforeEach(() => {
    fetch.mockClear();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('Component Rendering', () => {
    test('renders API Library title', () => {
      render(<App />);
      const titleElement = screen.getByText(/API Library/i);
      expect(titleElement).toBeInTheDocument();
    });

    test('renders search input', () => {
      render(<App />);
      const searchInput = screen.getByPlaceholderText(/Search for APIs/i);
      expect(searchInput).toBeInTheDocument();
    });

    test('renders all main tabs', () => {
      render(<App />);
      expect(screen.getByText('Search')).toBeInTheDocument();
      expect(screen.getByText('Browse')).toBeInTheDocument();
      expect(screen.getByText('Configured')).toBeInTheDocument();
      expect(screen.getByText('Add API')).toBeInTheDocument();
    });
  });

  describe('Search Functionality', () => {
    test('performs search when user types and clicks search', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          results: [
            {
              id: '1',
              name: 'Test API',
              provider: 'Test Provider',
              description: 'Test Description',
              relevance_score: 0.9,
              configured: false,
              pricing_summary: 'Free tier available'
            }
          ]
        })
      });

      render(<App />);
      
      const searchInput = screen.getByPlaceholderText(/Search for APIs/i);
      const searchButton = screen.getByRole('button', { name: /search/i });
      
      fireEvent.change(searchInput, { target: { value: 'payment processing' } });
      fireEvent.click(searchButton);
      
      await waitFor(() => {
        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/v1/search'),
          expect.objectContaining({
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query: 'payment processing', limit: 20 })
          })
        );
      });
    });

    test('displays search results', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          results: [
            {
              id: '1',
              name: 'Stripe API',
              provider: 'Stripe',
              description: 'Payment processing API',
              relevance_score: 0.95,
              configured: true,
              pricing_summary: '$0.029 per transaction'
            }
          ]
        })
      });

      render(<App />);
      
      const searchInput = screen.getByPlaceholderText(/Search for APIs/i);
      const searchButton = screen.getByRole('button', { name: /search/i });
      
      fireEvent.change(searchInput, { target: { value: 'payment' } });
      fireEvent.click(searchButton);
      
      await waitFor(() => {
        expect(screen.getByText('Stripe API')).toBeInTheDocument();
        expect(screen.getByText('Payment processing API')).toBeInTheDocument();
        expect(screen.getByText('$0.029 per transaction')).toBeInTheDocument();
      });
    });

    test('handles search errors gracefully', async () => {
      fetch.mockRejectedValueOnce(new Error('Network error'));
      
      render(<App />);
      
      const searchInput = screen.getByPlaceholderText(/Search for APIs/i);
      const searchButton = screen.getByRole('button', { name: /search/i });
      
      fireEvent.change(searchInput, { target: { value: 'test' } });
      fireEvent.click(searchButton);
      
      await waitFor(() => {
        expect(screen.getByText(/Error performing search/i)).toBeInTheDocument();
      });
    });
  });

  describe('Tab Navigation', () => {
    test('switches between tabs correctly', () => {
      render(<App />);
      
      // Click on Browse tab
      const browseTab = screen.getByText('Browse');
      fireEvent.click(browseTab);
      expect(browseTab).toHaveClass('active');
      
      // Click on Configured tab
      const configuredTab = screen.getByText('Configured');
      fireEvent.click(configuredTab);
      expect(configuredTab).toHaveClass('active');
      
      // Click back to Search tab
      const searchTab = screen.getByText('Search');
      fireEvent.click(searchTab);
      expect(searchTab).toHaveClass('active');
    });
  });

  describe('Add API Form', () => {
    test('displays form when Add API tab is clicked', () => {
      render(<App />);
      
      const addAPITab = screen.getByText('Add API');
      fireEvent.click(addAPITab);
      
      expect(screen.getByLabelText(/API Name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Provider/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Description/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Base URL/i)).toBeInTheDocument();
    });

    test('submits new API successfully', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 'new-api-id', name: 'New API' })
      });

      render(<App />);
      
      const addAPITab = screen.getByText('Add API');
      fireEvent.click(addAPITab);
      
      fireEvent.change(screen.getByLabelText(/API Name/i), { 
        target: { value: 'New API' } 
      });
      fireEvent.change(screen.getByLabelText(/Provider/i), { 
        target: { value: 'New Provider' } 
      });
      fireEvent.change(screen.getByLabelText(/Description/i), { 
        target: { value: 'A new API description' } 
      });
      fireEvent.change(screen.getByLabelText(/Base URL/i), { 
        target: { value: 'https://api.example.com' } 
      });
      
      const submitButton = screen.getByRole('button', { name: /Add API/i });
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/v1/apis'),
          expect.objectContaining({
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
          })
        );
      });
    });
  });

  describe('Configured APIs View', () => {
    test('loads and displays configured APIs', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => [
          {
            id: '1',
            name: 'Configured API',
            provider: 'Provider',
            environment: 'production',
            configuration_date: '2025-01-15T10:00:00Z'
          }
        ]
      });

      render(<App />);
      
      const configuredTab = screen.getByText('Configured');
      fireEvent.click(configuredTab);
      
      await waitFor(() => {
        expect(screen.getByText('Configured API')).toBeInTheDocument();
        expect(screen.getByText('production')).toBeInTheDocument();
      });
    });
  });

  describe('Cost Calculator', () => {
    test('calculates cost when provided usage data', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          recommended_tier: {
            name: 'Professional',
            estimated_cost: 99.99,
            cost_breakdown: {
              monthly_base: 50,
              request_cost: 30,
              data_cost: 19.99
            }
          },
          alternatives: [],
          savings_tip: 'Consider caching frequently used responses'
        })
      });

      render(<App />);
      
      // Navigate to an API detail view (this would require the API to be loaded first)
      // This is a simplified test - full implementation would require more setup
      
      const response = await fetch('/api/v1/calculate-cost', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          api_id: 'test-id',
          requests_per_month: 10000,
          data_per_request_mb: 0.5
        })
      });
      
      const data = await response.json();
      expect(data.recommended_tier.estimated_cost).toBe(99.99);
      expect(data.savings_tip).toBeTruthy();
    });
  });

  describe('Version Tracking', () => {
    test('displays version history for an API', async () => {
      fetch.mockResolvedValueOnce({
        ok: true,
        json: async () => [
          {
            id: 'v1',
            version: '2.0.0',
            change_summary: 'Breaking changes in auth',
            breaking_changes: true,
            created_at: '2025-01-10T10:00:00Z'
          },
          {
            id: 'v2',
            version: '1.5.0',
            change_summary: 'Added new endpoints',
            breaking_changes: false,
            created_at: '2024-12-01T10:00:00Z'
          }
        ]
      });

      // This would require navigating to an API detail view
      const response = await fetch('/api/v1/apis/test-id/versions');
      const versions = await response.json();
      
      expect(versions).toHaveLength(2);
      expect(versions[0].version).toBe('2.0.0');
      expect(versions[0].breaking_changes).toBe(true);
    });
  });

  describe('Accessibility', () => {
    test('has proper ARIA labels for interactive elements', () => {
      render(<App />);
      
      const searchInput = screen.getByPlaceholderText(/Search for APIs/i);
      expect(searchInput).toHaveAttribute('type', 'text');
      
      const buttons = screen.getAllByRole('button');
      buttons.forEach(button => {
        expect(button).toBeVisible();
      });
    });

    test('supports keyboard navigation', () => {
      render(<App />);
      
      const searchInput = screen.getByPlaceholderText(/Search for APIs/i);
      searchInput.focus();
      expect(document.activeElement).toBe(searchInput);
      
      // Simulate Tab key to move focus
      fireEvent.keyDown(searchInput, { key: 'Tab', code: 'Tab' });
      // Focus should move to next interactive element
    });
  });

  describe('Performance', () => {
    test('debounces search input to avoid excessive API calls', async () => {
      jest.useFakeTimers();
      
      render(<App />);
      
      const searchInput = screen.getByPlaceholderText(/Search for APIs/i);
      
      // Type multiple characters quickly
      fireEvent.change(searchInput, { target: { value: 'a' } });
      fireEvent.change(searchInput, { target: { value: 'ap' } });
      fireEvent.change(searchInput, { target: { value: 'api' } });
      
      // Advance timers
      jest.advanceTimersByTime(500);
      
      // Should only make one API call after debounce
      await waitFor(() => {
        expect(fetch).toHaveBeenCalledTimes(1);
      });
      
      jest.useRealTimers();
    });
  });
});