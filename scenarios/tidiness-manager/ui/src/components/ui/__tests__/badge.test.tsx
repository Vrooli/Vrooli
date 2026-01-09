import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Badge } from '../badge';

describe('Badge', () => {
  it('renders with default variant and size', () => {
    render(<Badge>Default</Badge>);
    const badge = screen.getByText('Default');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('bg-slate-700', 'text-slate-100', 'text-xs');
  });

  it('renders with active variant', () => {
    render(<Badge variant="active">Active</Badge>);
    const badge = screen.getByText('Active');
    expect(badge).toHaveClass('bg-blue-700', 'text-white');
  });

  it('renders with success variant', () => {
    render(<Badge variant="success">Success</Badge>);
    const badge = screen.getByText('Success');
    expect(badge).toHaveClass('bg-green-700', 'text-white');
  });

  it('renders with warning variant', () => {
    render(<Badge variant="warning">Warning</Badge>);
    const badge = screen.getByText('Warning');
    expect(badge).toHaveClass('bg-yellow-700', 'text-white');
  });

  it('renders with error variant', () => {
    render(<Badge variant="error">Error</Badge>);
    const badge = screen.getByText('Error');
    expect(badge).toHaveClass('bg-red-700', 'text-white');
  });

  it('renders with paused variant', () => {
    render(<Badge variant="paused">Paused</Badge>);
    const badge = screen.getByText('Paused');
    expect(badge).toHaveClass('bg-orange-700', 'text-white');
  });

  it('renders with outline variant', () => {
    render(<Badge variant="outline">Outline</Badge>);
    const badge = screen.getByText('Outline');
    expect(badge).toHaveClass('border', 'border-white/30', 'text-slate-200');
  });

  it('renders with ghost variant', () => {
    render(<Badge variant="ghost">Ghost</Badge>);
    const badge = screen.getByText('Ghost');
    expect(badge).toHaveClass('bg-white/5', 'text-slate-300');
  });

  it('renders with small size', () => {
    render(<Badge size="sm">Small</Badge>);
    const badge = screen.getByText('Small');
    expect(badge).toHaveClass('text-[10px]', 'px-2', 'py-0.5');
  });

  it('renders with large size', () => {
    render(<Badge size="lg">Large</Badge>);
    const badge = screen.getByText('Large');
    expect(badge).toHaveClass('text-sm', 'px-3', 'py-1');
  });

  it('applies custom className', () => {
    render(<Badge className="custom-class">Custom</Badge>);
    const badge = screen.getByText('Custom');
    expect(badge).toHaveClass('custom-class');
  });

  it('includes role="status"', () => {
    render(<Badge>Status Badge</Badge>);
    const badge = screen.getByText('Status Badge');
    expect(badge).toHaveAttribute('role', 'status');
  });

  it('includes default aria-label based on variant', () => {
    render(<Badge variant="success">Success Message</Badge>);
    const badge = screen.getByText('Success Message');
    expect(badge).toHaveAttribute('aria-label', 'Status: success');
  });

  it('accepts custom aria-label', () => {
    render(<Badge aria-label="Custom Status">Badge</Badge>);
    const badge = screen.getByText('Badge');
    expect(badge).toHaveAttribute('aria-label', 'Custom Status');
  });

  it('renders children correctly', () => {
    render(
      <Badge>
        <span>Icon</span>
        <span>Text</span>
      </Badge>
    );
    expect(screen.getByText('Icon')).toBeInTheDocument();
    expect(screen.getByText('Text')).toBeInTheDocument();
  });

  it('supports responsive sizing with custom classes', () => {
    render(<Badge className="text-xs sm:text-sm md:text-base">Responsive</Badge>);
    const badge = screen.getByText('Responsive');
    expect(badge).toHaveClass('text-xs', 'sm:text-sm', 'md:text-base');
  });

  it('includes focus ring classes', () => {
    render(<Badge>Focus Test</Badge>);
    const badge = screen.getByText('Focus Test');
    expect(badge).toHaveClass('focus:outline-none', 'focus:ring-2', 'focus:ring-white/50');
  });
});
