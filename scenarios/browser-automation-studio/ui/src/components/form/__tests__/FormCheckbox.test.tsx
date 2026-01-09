import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FormCheckbox } from '../FormCheckbox';

describe('FormCheckbox', () => {
  it('renders with label', () => {
    render(<FormCheckbox checked={false} onChange={() => {}} label="Enable feature" />);
    expect(screen.getByLabelText('Enable feature')).toBeInTheDocument();
  });

  it('displays checked state correctly', () => {
    render(<FormCheckbox checked={true} onChange={() => {}} label="Enable feature" />);
    expect(screen.getByRole('checkbox')).toBeChecked();
  });

  it('displays unchecked state correctly', () => {
    render(<FormCheckbox checked={false} onChange={() => {}} label="Enable feature" />);
    expect(screen.getByRole('checkbox')).not.toBeChecked();
  });

  it('calls onChange with true when checking', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(<FormCheckbox checked={false} onChange={handleChange} label="Enable feature" />);

    await user.click(screen.getByRole('checkbox'));

    expect(handleChange).toHaveBeenCalledWith(true);
  });

  it('calls onChange with false when unchecking', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(<FormCheckbox checked={true} onChange={handleChange} label="Enable feature" />);

    await user.click(screen.getByRole('checkbox'));

    expect(handleChange).toHaveBeenCalledWith(false);
  });

  it('renders description when provided', () => {
    render(
      <FormCheckbox
        checked={true}
        onChange={() => {}}
        label="Enable feature"
        description="When enabled, this feature will be active"
      />
    );
    expect(screen.getByText('When enabled, this feature will be active')).toBeInTheDocument();
  });

  it('does not render description when not provided', () => {
    const { container } = render(
      <FormCheckbox checked={true} onChange={() => {}} label="Enable feature" />
    );
    expect(container.querySelector('p')).not.toBeInTheDocument();
  });

  it('respects disabled state', () => {
    render(<FormCheckbox checked={true} onChange={() => {}} label="Enable feature" disabled />);
    expect(screen.getByRole('checkbox')).toBeDisabled();
  });

  it('does not call onChange when disabled', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(<FormCheckbox checked={false} onChange={handleChange} label="Enable feature" disabled />);

    await user.click(screen.getByRole('checkbox'));

    expect(handleChange).not.toHaveBeenCalled();
  });

  it('applies custom className to wrapper', () => {
    const { container } = render(
      <FormCheckbox checked={true} onChange={() => {}} label="Enable feature" className="custom-class" />
    );
    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('can be toggled by clicking label text', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(<FormCheckbox checked={false} onChange={handleChange} label="Enable feature" />);

    await user.click(screen.getByText('Enable feature'));

    expect(handleChange).toHaveBeenCalledWith(true);
  });
});
