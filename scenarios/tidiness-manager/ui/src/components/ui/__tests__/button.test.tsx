import { describe, it, expect, vi } from 'vitest';
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../../test-utils";
import userEvent from '@testing-library/user-event';
import { Button } from '../button';

describe('Button', () => {
  it('renders with default variant and size', () => {
    renderWithProviders(<Button>Default</Button>);
    const button = screen.getByRole('button');
    expect(button).toBeInTheDocument();
    expect(button).toHaveClass('bg-slate-50', 'text-slate-900', 'h-10', 'px-4');
  });

  it('renders with primary variant', () => {
    renderWithProviders(<Button variant="primary">Primary</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('bg-blue-600', 'text-white');
  });

  it('renders with success variant', () => {
    renderWithProviders(<Button variant="success">Success</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('bg-green-600', 'text-white');
  });

  it('renders with destructive variant', () => {
    renderWithProviders(<Button variant="destructive">Destructive</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('bg-red-600', 'text-white');
  });

  it('renders with outline variant', () => {
    renderWithProviders(<Button variant="outline">Outline</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('border', 'border-white/30', 'text-white');
  });

  it('renders with ghost variant', () => {
    renderWithProviders(<Button variant="ghost">Ghost</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('text-white', 'hover:bg-white/10');
  });

  it('renders with link variant', () => {
    renderWithProviders(<Button variant="link">Link</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('text-blue-400', 'underline-offset-4');
  });

  it('renders with small size', () => {
    renderWithProviders(<Button size="sm">Small</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('h-9', 'px-3', 'text-xs', 'min-h-[2.25rem]');
  });

  it('renders with large size', () => {
    renderWithProviders(<Button size="lg">Large</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('h-12', 'px-6', 'text-base');
  });

  it('renders with icon size', () => {
    renderWithProviders(<Button size="icon">🔍</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('h-11', 'w-11');
  });

  it('applies custom className', () => {
    renderWithProviders(<Button className="custom-class">Custom</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('custom-class');
  });

  it('handles onClick events', async () => {
    const handleClick = vi.fn();
    const user = userEvent.setup();

    renderWithProviders(<Button onClick={handleClick}>Click me</Button>);
    const button = screen.getByRole('button');

    await user.click(button);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('respects disabled state', async () => {
    const handleClick = vi.fn();
    const user = userEvent.setup();

    renderWithProviders(<Button onClick={handleClick} disabled>Disabled</Button>);
    const button = screen.getByRole('button');

    expect(button).toBeDisabled();
    expect(button).toHaveClass('disabled:pointer-events-none', 'disabled:opacity-60');

    // User interaction should fail on disabled button
    await user.click(button);
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('includes focus-visible styles', () => {
    renderWithProviders(<Button>Focus Test</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass(
      'focus-visible:outline-none',
      'focus-visible:ring-2',
      'focus-visible:ring-white/50'
    );
  });

  it('includes transition classes', () => {
    renderWithProviders(<Button>Transition</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('transition-colors');
  });

  it('supports type attribute', () => {
    renderWithProviders(<Button type="submit">Submit</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('type', 'submit');
  });

  it('renders children correctly', () => {
    renderWithProviders(
      <Button>
        <span>Icon</span>
        <span>Text</span>
      </Button>
    );
    expect(screen.getByText('Icon')).toBeInTheDocument();
    expect(screen.getByText('Text')).toBeInTheDocument();
  });

  it('supports asChild prop with Slot', () => {
    renderWithProviders(
      <Button asChild>
        <a href="/test">Link Button</a>
      </Button>
    );
    const link = screen.getByText('Link Button');
    expect(link.tagName).toBe('A');
    expect(link).toHaveAttribute('href', '/test');
  });

  it('does not add role when asChild is true', () => {
    renderWithProviders(
      <Button asChild>
        <a href="/test">Link</a>
      </Button>
    );
    const link = screen.getByText('Link');
    expect(link).not.toHaveAttribute('role', 'button');
  });

  it('supports responsive sizing with custom classes', () => {
    renderWithProviders(<Button className="h-8 sm:h-10 md:h-12">Responsive</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('h-8', 'sm:h-10', 'md:h-12');
  });

  it('handles keyboard events', async () => {
    const handleClick = vi.fn();
    const user = userEvent.setup();

    renderWithProviders(<Button onClick={handleClick}>Keyboard Test</Button>);
    const button = screen.getByRole('button');

    button.focus();
    await user.keyboard('{Enter}');
    expect(handleClick).toHaveBeenCalledTimes(1);

    await user.keyboard(' ');
    expect(handleClick).toHaveBeenCalledTimes(2);
  });
});
