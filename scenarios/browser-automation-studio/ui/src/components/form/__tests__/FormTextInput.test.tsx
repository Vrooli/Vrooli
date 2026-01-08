import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FormTextInput } from '../FormTextInput';

describe('FormTextInput', () => {
  it('renders with label', () => {
    render(<FormTextInput value="test" onChange={() => {}} label="Locale" />);
    expect(screen.getByLabelText('Locale')).toBeInTheDocument();
  });

  it('displays value correctly', () => {
    render(<FormTextInput value="en-US" onChange={() => {}} label="Locale" />);
    expect(screen.getByRole('textbox')).toHaveValue('en-US');
  });

  it('displays empty string when value is undefined', () => {
    render(<FormTextInput value={undefined} onChange={() => {}} label="Locale" />);
    expect(screen.getByRole('textbox')).toHaveValue('');
  });

  it('calls onChange with string when value changes', () => {
    const handleChange = vi.fn();
    render(<FormTextInput value="" onChange={handleChange} label="Locale" />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'en-US' } });

    expect(handleChange).toHaveBeenCalledWith('en-US');
  });

  it('calls onChange with undefined when input is cleared', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(<FormTextInput value="en-US" onChange={handleChange} label="Locale" />);

    const input = screen.getByRole('textbox');
    await user.clear(input);

    expect(handleChange).toHaveBeenCalledWith(undefined);
  });

  it('renders placeholder', () => {
    render(<FormTextInput value={undefined} onChange={() => {}} label="Locale" placeholder="en-US" />);
    expect(screen.getByPlaceholderText('en-US')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(
      <FormTextInput
        value="en-US"
        onChange={() => {}}
        label="Locale"
        description="Browser locale setting"
      />
    );
    expect(screen.getByText('Browser locale setting')).toBeInTheDocument();
  });

  it('does not render description when not provided', () => {
    const { container } = render(<FormTextInput value="en-US" onChange={() => {}} label="Locale" />);
    expect(container.querySelector('p')).not.toBeInTheDocument();
  });

  it('respects disabled state', () => {
    render(<FormTextInput value="en-US" onChange={() => {}} label="Locale" disabled />);
    expect(screen.getByRole('textbox')).toBeDisabled();
  });

  it('applies password type correctly', () => {
    render(<FormTextInput value="secret" onChange={() => {}} label="Password" type="password" />);
    const input = screen.getByLabelText('Password');
    expect(input).toHaveAttribute('type', 'password');
  });

  it('applies email type correctly', () => {
    render(<FormTextInput value="test@example.com" onChange={() => {}} label="Email" type="email" />);
    const input = screen.getByLabelText('Email');
    expect(input).toHaveAttribute('type', 'email');
  });

  it('applies custom className to wrapper', () => {
    const { container } = render(
      <FormTextInput value="test" onChange={() => {}} label="Locale" className="custom-class" />
    );
    expect(container.firstChild).toHaveClass('custom-class');
  });
});
