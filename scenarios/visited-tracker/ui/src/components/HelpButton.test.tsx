import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TooltipProvider } from './ui/tooltip';
import { HelpButton } from './HelpButton';

const renderWithProvider = (ui: React.ReactElement) => {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
};

describe('HelpButton', () => {
  it('should render help icon button', () => {
    renderWithProvider(<HelpButton content="Test help content" />);
    const button = screen.getByRole('button', { name: /help/i });
    expect(button).toBeInTheDocument();
  });

  it('should have proper accessibility attributes', () => {
    renderWithProvider(<HelpButton content="Accessibility test" />);
    const button = screen.getByRole('button', { name: /help/i });
    expect(button).toHaveAttribute('aria-label', 'Help');
  });

  it('should render with correct icon size', () => {
    renderWithProvider(<HelpButton content="Size test" />);
    const button = screen.getByRole('button');
    const svg = button.querySelector('svg');
    expect(svg).toHaveClass('h-4', 'w-4');
  });
});
