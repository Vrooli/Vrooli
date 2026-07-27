import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { Stepper } from './stepper';

const steps = [
  { id: 'provider', label: 'Provider' },
  { id: 'credentials', label: 'Credentials' },
  { id: 'verify', label: 'Verify' },
];

describe('Stepper', () => {
  it('renders semantic progress and permits only completed/current steps by default', () => {
    const onStepClick = vi.fn();
    render(<Stepper steps={steps} currentStep={1} onStepClick={onStepClick} aria-label="Storage setup" />);

    expect(screen.getByRole('navigation', { name: 'Storage setup' })).toBeInTheDocument();
    const buttons = screen.getAllByRole('button');
    expect(buttons).toHaveLength(3);
    expect(buttons[0]).toBeEnabled();
    expect(buttons[1]).toBeEnabled();
    expect(buttons[2]).toBeDisabled();
    fireEvent.click(buttons[0]!);
    expect(onStepClick).toHaveBeenCalledWith(0);
  });

  it('honors explicit lifecycle states, displayed context, and future navigation', () => {
    const onStepClick = vi.fn();
    render(<Stepper steps={steps} currentStep={0} onStepClick={onStepClick} allowFutureClicks displayedStep={2} stepStates={['completed', 'running', 'skipped']} />);

    const buttons = screen.getAllByRole('button');
    buttons.forEach((button, index) => {
      expect(button).toBeEnabled();
      fireEvent.click(button);
      expect(onStepClick).toHaveBeenLastCalledWith(index);
    });
    expect(screen.getByText('Provider').previousElementSibling?.className).toContain('bg-emerald-500');
    expect(screen.getByText('Credentials').previousElementSibling?.className).toContain('bg-slate-900');
  });
});
