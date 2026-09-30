import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { FilterBar } from './FilterBar';

vi.mock('./TimeRangePicker', () => ({ TimeRangePicker: () => <div data-testid="time-range" /> }));

const filters = { units: [], priority: '', since: '1h ago', until: 'now', boot: '', grep: '', kernel: false };

describe('FilterBar', () => {
  it('commits valid filters and exposes all interactive controls', () => {
    const onChange = vi.fn();
    const onReset = vi.fn();
    render(<FilterBar filters={filters} units={['systemd', 'kernel']} boots={[{ index: 0, bootId: 'boot-zero' }, { index: 1, bootId: '' }]} onChange={onChange} onReset={onReset} />);
    fireEvent.change(screen.getByLabelText('Unit'), { target: { value: 'systemd' } });
    fireEvent.change(screen.getByLabelText('Priority'), { target: { value: '3' } });
    fireEvent.change(screen.getByLabelText('Boot'), { target: { value: 'boot-zero' } });
    const grep = screen.getByPlaceholderText('oom_killer');
    fireEvent.change(grep, { target: { value: 'oom.*' } });
    fireEvent.keyDown(grep, { key: 'Enter' });
    fireEvent.click(screen.getByLabelText('Kernel only'));
    fireEvent.click(screen.getByRole('button', { name: 'Reset' }));
    expect(onChange).toHaveBeenCalledWith({ units: ['systemd'] });
    expect(onChange).toHaveBeenCalledWith({ grep: 'oom.*' });
    expect(onReset).toHaveBeenCalledOnce();
  });

  it('shows invalid regex errors and clears a selected unit', () => {
    const onChange = vi.fn();
    render(<FilterBar filters={{ ...filters, units: ['systemd'] }} units={['systemd']} boots={[]} onChange={onChange} onReset={vi.fn()} />);
    fireEvent.change(screen.getByLabelText('Unit'), { target: { value: '' } });
    const grep = screen.getByPlaceholderText('oom_killer');
    fireEvent.change(grep, { target: { value: '[' } });
    fireEvent.blur(grep);
    expect(screen.getByText(/invalid/i)).toBeInTheDocument();
    expect(onChange).toHaveBeenCalledWith({ units: [] });
  });
});
