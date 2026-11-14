import { describe, it, expect, vi } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { createRef } from 'react';
import { renderWithProviders as render } from '../../test-utils';
import { Input } from './input';

describe('Input', () => {
  it('renders a text input and forwards value changes', () => {
    const onChange = vi.fn();
    render(<Input placeholder="Email" onChange={onChange} />);
    const input = screen.getByPlaceholderText('Email');
    fireEvent.change(input, { target: { value: 'hi@example.com' } });
    expect(onChange).toHaveBeenCalled();
    expect(input).toHaveValue('hi@example.com');
  });

  it('applies the given type and passes through native attributes', () => {
    render(<Input type="password" aria-label="secret" disabled />);
    const input = screen.getByLabelText('secret');
    expect(input).toHaveAttribute('type', 'password');
    expect(input).toBeDisabled();
  });

  it('merges custom className with the base styles', () => {
    render(<Input aria-label="named" className="custom-input" />);
    const input = screen.getByLabelText('named');
    expect(input).toHaveClass('custom-input');
    expect(input.className).toContain('rounded-lg');
  });

  it('forwards a ref to the underlying element', () => {
    const ref = createRef<HTMLInputElement>();
    render(<Input ref={ref} aria-label="ref" />);
    expect(ref.current).toBeInstanceOf(HTMLInputElement);
  });
});
