import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Select } from '../select';

describe('Select', () => {
  it('renders correctly with default props', () => {
    render(
      <Select>
        <option value="">Choose option</option>
        <option value="1">Option 1</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toBeInTheDocument();
  });

  it('applies default styling classes', () => {
    render(
      <Select>
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveClass(
      'flex',
      'h-10',
      'w-full',
      'rounded-lg',
      'border',
      'border-white/20',
      'bg-white/5',
      'px-3',
      'py-2'
    );
  });

  it('applies custom className', () => {
    render(
      <Select className="custom-class">
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveClass('custom-class');
  });

  it('renders options correctly', () => {
    render(
      <Select>
        <option value="1">Option 1</option>
        <option value="2">Option 2</option>
        <option value="3">Option 3</option>
      </Select>
    );

    expect(screen.getByText('Option 1')).toBeInTheDocument();
    expect(screen.getByText('Option 2')).toBeInTheDocument();
    expect(screen.getByText('Option 3')).toBeInTheDocument();
  });

  it('handles value changes', async () => {
    const handleChange = vi.fn();
    const user = userEvent.setup();

    render(
      <Select onChange={handleChange}>
        <option value="">Select</option>
        <option value="1">Option 1</option>
        <option value="2">Option 2</option>
      </Select>
    );

    const select = screen.getByRole('combobox');
    await user.selectOptions(select, '1');

    expect(handleChange).toHaveBeenCalled();
    expect(select).toHaveValue('1');
  });

  it('respects disabled state', async () => {
    const handleChange = vi.fn();
    const user = userEvent.setup();

    render(
      <Select disabled onChange={handleChange}>
        <option value="1">Option 1</option>
        <option value="2">Option 2</option>
      </Select>
    );

    const select = screen.getByRole('combobox');
    expect(select).toBeDisabled();
    expect(select).toHaveClass('disabled:cursor-not-allowed', 'disabled:opacity-50');

    // Trying to select should not work
    await user.selectOptions(select, '2');
    expect(handleChange).not.toHaveBeenCalled();
  });

  it('includes focus styles', () => {
    render(
      <Select>
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveClass(
      'focus:border-white/40',
      'focus:outline-none',
      'focus:ring-2',
      'focus:ring-white/20'
    );
  });

  it('includes hover transition', () => {
    render(
      <Select>
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveClass('hover:bg-white/10', 'transition-colors');
  });

  it('supports required attribute', () => {
    render(
      <Select required>
        <option value="">Select</option>
        <option value="1">Option 1</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveAttribute('required');
    expect(select).toHaveAttribute('aria-required', 'true');
  });

  it('sets aria-required to false when not required', () => {
    render(
      <Select>
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).not.toHaveAttribute('required');
    expect(select).toHaveAttribute('aria-required', 'false');
  });

  it('supports custom aria-invalid attribute', () => {
    render(
      <Select aria-invalid={true}>
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveAttribute('aria-invalid', 'true');
  });

  it('defaults aria-invalid to false', () => {
    render(
      <Select>
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveAttribute('aria-invalid', 'false');
  });

  it('supports defaultValue', () => {
    render(
      <Select defaultValue="2">
        <option value="1">Option 1</option>
        <option value="2">Option 2</option>
        <option value="3">Option 3</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveValue('2');
  });

  it('supports controlled value', () => {
    const { rerender } = render(
      <Select value="1" onChange={() => {}}>
        <option value="1">Option 1</option>
        <option value="2">Option 2</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveValue('1');

    rerender(
      <Select value="2" onChange={() => {}}>
        <option value="1">Option 1</option>
        <option value="2">Option 2</option>
      </Select>
    );
    expect(select).toHaveValue('2');
  });

  it('supports name attribute for forms', () => {
    render(
      <Select name="category">
        <option value="1">Category 1</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveAttribute('name', 'category');
  });

  it('includes option background styling', () => {
    render(
      <Select>
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    // Check for option styling classes
    expect(select).toHaveClass('[&>option]:bg-slate-900', '[&>option]:text-slate-100');
  });

  it('supports multiple attribute', () => {
    render(
      <Select multiple>
        <option value="1">Option 1</option>
        <option value="2">Option 2</option>
        <option value="3">Option 3</option>
      </Select>
    );
    const select = screen.getByRole('listbox');
    expect(select).toHaveAttribute('multiple');
  });

  it('supports optgroup', () => {
    render(
      <Select>
        <optgroup label="Group 1">
          <option value="1">Option 1</option>
          <option value="2">Option 2</option>
        </optgroup>
        <optgroup label="Group 2">
          <option value="3">Option 3</option>
          <option value="4">Option 4</option>
        </optgroup>
      </Select>
    );

    expect(screen.getByText('Option 1')).toBeInTheDocument();
    expect(screen.getByText('Option 3')).toBeInTheDocument();
  });

  it('supports responsive width classes', () => {
    render(
      <Select className="w-full sm:w-1/2 md:w-1/3">
        <option>Test</option>
      </Select>
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveClass('w-full', 'sm:w-1/2', 'md:w-1/3');
  });
});
