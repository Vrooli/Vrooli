import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Input } from '../input';

describe('Input', () => {
  it('renders correctly with default props', () => {
    render(<Input placeholder="Enter text" />);
    const input = screen.getByPlaceholderText('Enter text');
    expect(input).toBeInTheDocument();
    expect(input).toHaveClass('flex', 'h-10', 'w-full', 'rounded-lg');
  });

  it('applies default styling classes', () => {
    render(<Input />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass(
      'border',
      'border-white/20',
      'bg-white/5',
      'px-3',
      'py-2',
      'text-sm',
      'text-slate-100'
    );
  });

  it('renders with text type by default', () => {
    render(<Input />);
    const input = screen.getByRole('textbox');
    // Input type defaults to text when not specified (browser default)
    expect(input.getAttribute('type')).toBeNull(); // or expect 'text' if explicitly set
  });

  it('supports custom type attribute', () => {
    render(<Input type="email" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('type', 'email');
  });

  it('supports password type', () => {
    render(<Input type="password" />);
    const input = document.querySelector('input[type="password"]');
    expect(input).toBeInTheDocument();
  });

  it('supports number type', () => {
    render(<Input type="number" />);
    const input = screen.getByRole('spinbutton');
    expect(input).toHaveAttribute('type', 'number');
  });

  it('applies custom className', () => {
    render(<Input className="custom-class" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass('custom-class');
  });

  it('handles value changes', async () => {
    const handleChange = vi.fn();
    const user = userEvent.setup();

    render(<Input onChange={handleChange} />);
    const input = screen.getByRole('textbox');

    await user.type(input, 'Hello');
    expect(handleChange).toHaveBeenCalled();
    expect(input).toHaveValue('Hello');
  });

  it('respects disabled state', async () => {
    const handleChange = vi.fn();
    const user = userEvent.setup();

    render(<Input disabled onChange={handleChange} />);
    const input = screen.getByRole('textbox');

    expect(input).toBeDisabled();
    expect(input).toHaveClass('disabled:cursor-not-allowed', 'disabled:opacity-50');

    await user.type(input, 'Test');
    expect(handleChange).not.toHaveBeenCalled();
    expect(input).toHaveValue('');
  });

  it('includes focus styles', () => {
    render(<Input />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass(
      'focus:border-white/40',
      'focus:outline-none',
      'focus:ring-2',
      'focus:ring-white/20'
    );
  });

  it('supports placeholder text', () => {
    render(<Input placeholder="Search..." />);
    const input = screen.getByPlaceholderText('Search...');
    expect(input).toBeInTheDocument();
    expect(input).toHaveClass('placeholder:text-slate-400');
  });

  it('supports required attribute', () => {
    render(<Input required />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('required');
    expect(input).toHaveAttribute('aria-required', 'true');
  });

  it('sets aria-required to false when not required', () => {
    render(<Input />);
    const input = screen.getByRole('textbox');
    expect(input).not.toHaveAttribute('required');
    expect(input).toHaveAttribute('aria-required', 'false');
  });

  it('supports custom aria-invalid attribute', () => {
    render(<Input aria-invalid={true} />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('aria-invalid', 'true');
  });

  it('defaults aria-invalid to false', () => {
    render(<Input />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('aria-invalid', 'false');
  });

  it('supports maxLength attribute', () => {
    render(<Input maxLength={10} />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('maxLength', '10');
  });

  it('supports defaultValue', () => {
    render(<Input defaultValue="Default text" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveValue('Default text');
  });

  it('supports controlled value', () => {
    const { rerender } = render(<Input value="Initial" onChange={() => {}} />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveValue('Initial');

    rerender(<Input value="Updated" onChange={() => {}} />);
    expect(input).toHaveValue('Updated');
  });

  it('includes file upload styles', () => {
    render(<Input type="file" />);
    const input = document.querySelector('input[type="file"]');
    expect(input).toHaveClass('file:border-0', 'file:bg-transparent', 'file:text-sm');
  });

  it('supports onFocus and onBlur events', async () => {
    const handleFocus = vi.fn();
    const handleBlur = vi.fn();
    const user = userEvent.setup();

    render(<Input onFocus={handleFocus} onBlur={handleBlur} />);
    const input = screen.getByRole('textbox');

    await user.click(input);
    expect(handleFocus).toHaveBeenCalledTimes(1);

    await user.tab();
    expect(handleBlur).toHaveBeenCalledTimes(1);
  });

  it('supports autoComplete attribute', () => {
    render(<Input autoComplete="email" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('autoComplete', 'email');
  });

  it('supports name attribute for forms', () => {
    render(<Input name="username" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveAttribute('name', 'username');
  });

  it('supports responsive width classes', () => {
    render(<Input className="w-full sm:w-1/2 md:w-1/3" />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass('w-full', 'sm:w-1/2', 'md:w-1/3');
  });
});
