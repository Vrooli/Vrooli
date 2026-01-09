import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FormNumberInput } from '../FormNumberInput';

describe('FormNumberInput', () => {
  it('renders with label', () => {
    render(<FormNumberInput value={100} onChange={() => {}} label="Width" />);
    expect(screen.getByLabelText('Width')).toBeInTheDocument();
  });

  it('displays value correctly', () => {
    render(<FormNumberInput value={1920} onChange={() => {}} label="Width" />);
    expect(screen.getByRole('spinbutton')).toHaveValue(1920);
  });

  it('displays empty string when value is undefined', () => {
    render(<FormNumberInput value={undefined} onChange={() => {}} label="Width" />);
    expect(screen.getByRole('spinbutton')).toHaveValue(null);
  });

  it('calls onChange with parsed number when value changes', () => {
    const handleChange = vi.fn();
    render(<FormNumberInput value={undefined} onChange={handleChange} label="Width" />);

    const input = screen.getByRole('spinbutton');
    fireEvent.change(input, { target: { value: '500' } });

    expect(handleChange).toHaveBeenCalledWith(500);
  });

  it('calls onChange with undefined when input is cleared', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(<FormNumberInput value={100} onChange={handleChange} label="Width" />);

    const input = screen.getByRole('spinbutton');
    await user.clear(input);

    expect(handleChange).toHaveBeenCalledWith(undefined);
  });

  it('uses parseFloat when step has decimals', () => {
    const handleChange = vi.fn();
    render(<FormNumberInput value={undefined} onChange={handleChange} label="Scale" step={0.5} />);

    const input = screen.getByRole('spinbutton');
    fireEvent.change(input, { target: { value: '1.5' } });

    expect(handleChange).toHaveBeenCalledWith(1.5);
  });

  it('uses parseInt when step is integer', () => {
    const handleChange = vi.fn();
    render(<FormNumberInput value={undefined} onChange={handleChange} label="Count" step={1} />);

    const input = screen.getByRole('spinbutton');
    fireEvent.change(input, { target: { value: '42' } });

    expect(handleChange).toHaveBeenCalledWith(42);
  });

  it('renders placeholder', () => {
    render(<FormNumberInput value={undefined} onChange={() => {}} label="Width" placeholder="1920" />);
    expect(screen.getByPlaceholderText('1920')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(
      <FormNumberInput
        value={100}
        onChange={() => {}}
        label="Width"
        description="Enter viewport width"
      />
    );
    expect(screen.getByText('Enter viewport width')).toBeInTheDocument();
  });

  it('does not render description when not provided', () => {
    const { container } = render(<FormNumberInput value={100} onChange={() => {}} label="Width" />);
    expect(container.querySelector('p')).not.toBeInTheDocument();
  });

  it('respects disabled state', () => {
    render(<FormNumberInput value={100} onChange={() => {}} label="Width" disabled />);
    expect(screen.getByRole('spinbutton')).toBeDisabled();
  });

  it('applies min and max attributes', () => {
    render(<FormNumberInput value={50} onChange={() => {}} label="Width" min={0} max={100} />);
    const input = screen.getByRole('spinbutton');
    expect(input).toHaveAttribute('min', '0');
    expect(input).toHaveAttribute('max', '100');
  });

  it('applies custom className to wrapper', () => {
    const { container } = render(
      <FormNumberInput value={100} onChange={() => {}} label="Width" className="custom-class" />
    );
    expect(container.firstChild).toHaveClass('custom-class');
  });
});
