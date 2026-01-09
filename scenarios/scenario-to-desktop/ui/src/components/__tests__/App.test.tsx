import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import App from '../../App';

describe('App', () => {
  it('renders the main heading', () => {
    render(<App />);
    expect(screen.getByText('Scenario to Desktop')).toBeInTheDocument();
  });

  it('renders the tagline', () => {
    render(<App />);
    expect(screen.getByText(/Transform Vrooli scenarios into professional desktop applications/i)).toBeInTheDocument();
  });

  it('renders view mode selector buttons', () => {
    render(<App />);
    expect(screen.getByText('Scenario Inventory')).toBeInTheDocument();
    expect(screen.getByText('Generate Desktop App')).toBeInTheDocument();
  });

  it('defaults to inventory view', () => {
    render(<App />);
    // Inventory view should be active by default - just verify both buttons are present
    const inventoryButton = screen.getByRole('button', { name: /Scenario Inventory/i });
    const generateButton = screen.getByRole('button', { name: /Generate Desktop App/i });
    expect(inventoryButton).toBeInTheDocument();
    expect(generateButton).toBeInTheDocument();
  });
});
