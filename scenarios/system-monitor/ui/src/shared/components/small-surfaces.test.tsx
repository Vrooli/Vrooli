import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { ModalsContainer } from '../../features/investigations/modals/ModalsContainer';
import { StatusBadge } from './StatusBadge';
import { SearchInput } from './SearchInput';
import { TimeRangePicker } from '../../features/logs/components/TimeRangePicker';
import { parseMetricsResponse } from '../api/current-metrics-contract';

describe('small shared surfaces', () => {
  it('renders status badges and the modal coordinator closed state', () => {
    render(<StatusBadge variant="success">healthy</StatusBadge>);
    expect(screen.getByText('healthy')).toHaveClass('badge-success');
    render(<ModalsContainer modalState={{ scriptEditor: { isOpen: false, mode: 'view' }, scriptResults: { isOpen: false } }} onCloseScriptEditor={() => undefined} onCloseScriptResults={() => undefined} onExecuteScript={() => Promise.resolve()} />);
    expect(screen.getByText('healthy')).toBeInTheDocument();
  });

  it('parses current metrics through the canonical protobuf contract', () => {
    expect(parseMetricsResponse({ cpu: { state: { measured: 12 } } })).toBeDefined();
  });

  it('forwards search and time range edits', () => {
    const onSearch = vi.fn();
    const onRange = vi.fn();
    render(<><SearchInput placeholder="find" value="" onChange={onSearch} icon={<span>icon</span>} /><TimeRangePicker since="1h" until="now" onChange={onRange} /></>);
    fireEvent.change(screen.getByPlaceholderText('find'), { target: { value: 'disk' } });
    fireEvent.change(screen.getByDisplayValue('1h'), { target: { value: '2h' } });
    fireEvent.change(screen.getByDisplayValue('now'), { target: { value: 'later' } });
    expect(onSearch).toHaveBeenCalledWith('disk');
    expect(onRange).toHaveBeenNthCalledWith(1, { since: '2h' });
    expect(onRange).toHaveBeenNthCalledWith(2, { until: 'later' });
  });
});
