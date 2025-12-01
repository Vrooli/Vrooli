import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from './button';

/**
 * Button Component Tests
 * Validates button variants, sizes, and interactions
 */

describe('Button Component', () => {
  it('should render with default variant and size', () => {
    render(<Button>Click me</Button>);
    const button = screen.getByText('Click me');
    expect(button).toBeDefined();
    expect(button.tagName).toBe('BUTTON');
  });

  it('should apply outline variant styles', () => {
    const { container } = render(<Button variant="outline">Outline</Button>);
    const button = container.querySelector('button');
    expect(button?.className).toContain('border');
  });

  it('should apply small size styles', () => {
    const { container } = render(<Button size="sm">Small</Button>);
    const button = container.querySelector('button');
    expect(button?.className).toContain('h-9');
  });

  it('should handle click events', () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click</Button>);

    const button = screen.getByText('Click');
    fireEvent.click(button);

    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('should be disabled when disabled prop is true', () => {
    const handleClick = vi.fn();
    render(<Button disabled onClick={handleClick}>Disabled</Button>);

    const button = screen.getByText('Disabled') as HTMLButtonElement;
    expect(button.disabled).toBe(true);

    fireEvent.click(button);
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('should apply custom className', () => {
    const { container } = render(<Button className="custom-class">Custom</Button>);
    const button = container.querySelector('button');
    expect(button?.className).toContain('custom-class');
  });

  it('should render children correctly', () => {
    render(
      <Button>
        <span>Icon</span>
        <span>Text</span>
      </Button>
    );

    expect(screen.getByText('Icon')).toBeDefined();
    expect(screen.getByText('Text')).toBeDefined();
  });

  it('should support data attributes for testing', () => {
    render(<Button data-testid="test-button">Test</Button>);
    expect(screen.getByTestId('test-button')).toBeDefined();
  });

  it('should have accessible focus styles', () => {
    const { container } = render(<Button>Focus me</Button>);
    const button = container.querySelector('button');
    expect(button?.className).toContain('focus-visible:ring-2');
  });

  it('should have transition styles for smooth interactions', () => {
    const { container } = render(<Button>Transition</Button>);
    const button = container.querySelector('button');
    expect(button?.className).toContain('transition-colors');
  });
});
