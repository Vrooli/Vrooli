import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FormSelect } from '../FormSelect';

const options = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'no-preference', label: 'No Preference' },
];

describe('FormSelect', () => {
  it('renders with label', () => {
    render(<FormSelect value="light" onChange={() => {}} label="Theme" options={options} />);
    expect(screen.getByLabelText('Theme')).toBeInTheDocument();
  });

  it('displays selected value correctly', () => {
    render(<FormSelect value="dark" onChange={() => {}} label="Theme" options={options} />);
    expect(screen.getByRole('combobox')).toHaveValue('dark');
  });

  it('renders all options', () => {
    render(<FormSelect value="light" onChange={() => {}} label="Theme" options={options} />);

    expect(screen.getByRole('option', { name: 'Light' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Dark' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'No Preference' })).toBeInTheDocument();
  });

  it('calls onChange with selected value', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(<FormSelect value="light" onChange={handleChange} label="Theme" options={options} />);

    await user.selectOptions(screen.getByRole('combobox'), 'dark');

    expect(handleChange).toHaveBeenCalledWith('dark');
  });

  it('renders description when provided', () => {
    render(
      <FormSelect
        value="light"
        onChange={() => {}}
        label="Theme"
        options={options}
        description="Choose your preferred color scheme"
      />
    );
    expect(screen.getByText('Choose your preferred color scheme')).toBeInTheDocument();
  });

  it('does not render description when not provided', () => {
    const { container } = render(
      <FormSelect value="light" onChange={() => {}} label="Theme" options={options} />
    );
    expect(container.querySelector('p')).not.toBeInTheDocument();
  });

  it('respects disabled state', () => {
    render(<FormSelect value="light" onChange={() => {}} label="Theme" options={options} disabled />);
    expect(screen.getByRole('combobox')).toBeDisabled();
  });

  it('applies custom className to wrapper', () => {
    const { container } = render(
      <FormSelect
        value="light"
        onChange={() => {}}
        label="Theme"
        options={options}
        className="custom-class"
      />
    );
    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('handles undefined value by falling back to first option', () => {
    render(<FormSelect value={undefined} onChange={() => {}} label="Theme" options={options} />);
    // When value is undefined, the select falls back to the first option
    expect(screen.getByRole('combobox')).toHaveValue('light');
  });
});
