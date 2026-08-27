import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { MachinePicker } from './MachinePicker';
import type { Machine } from '../../../types';

const fleet: Machine[] = [
  { id: '', name: 'This machine', os: 'linux', arch: 'x86_64', online: true, heartbeat_fresh: true, dispatchable: true, status: 'local' },
  { id: 'mini', name: 'minimouse', os: 'darwin', arch: 'amd64', online: true, heartbeat_fresh: true, heartbeat_age_seconds: 8, dispatchable: true, status: 'online', scopes: ['*:read', '*:write'] },
  { id: 'swarm', name: 'swarminator', os: 'linux', arch: 'amd64', online: false, heartbeat_fresh: false, heartbeat_age_seconds: 639479, dispatchable: false, status: 'offline', readiness: [{ identity: 'heartbeat_fresh', passed: false }] }
];

const renderPicker = (overrides: Partial<React.ComponentProps<typeof MachinePicker>> = {}) => {
  const props = {
    machines: fleet,
    selectedMachineID: '',
    onSelectMachine: vi.fn(),
    onAddMachine: vi.fn(),
    ...overrides
  };
  render(<MachinePicker {...props} />);
  return props;
};

describe('MachinePicker', () => {
  it('shows reachability and grant per row, so choosing is an informed choice', () => {
    renderPicker();
    fireEvent.click(screen.getByTestId('machine-picker'));

    const listbox = screen.getByRole('listbox', { name: 'Machine' });
    const options = within(listbox).getAllByRole('option');
    expect(options.map(option => option.textContent)).toEqual([
      'This machinelinux · x86_64',
      'minimousedarwin · amd64 · 8s agooperate',
      'swarminatornot responding · 7d agono actions'
    ]);
  });

  it('keeps a machine that cannot answer choosable rather than hiding it', () => {
    const { onSelectMachine } = renderPicker();
    fireEvent.click(screen.getByTestId('machine-picker'));
    // Hiding it reads as "deleted" and sends people looking for it; disabling
    // it hides the reason. It is dimmed, labelled, and still selectable.
    fireEvent.click(screen.getByRole('option', { name: /swarminator/ }));
    expect(onSelectMachine).toHaveBeenCalledWith('swarm');
  });

  it('puts only options inside the listbox', () => {
    renderPicker();
    fireEvent.click(screen.getByTestId('machine-picker'));
    const listbox = screen.getByRole('listbox', { name: 'Machine' });
    // "Add a machine…" is a command, not a machine. Inside the listbox it
    // would be announced as a fourth machine to view.
    expect(within(listbox).queryByTestId('add-machine')).not.toBeInTheDocument();
    expect(screen.getByTestId('add-machine')).toBeInTheDocument();
  });

  it('opens onto the current selection and moves with the arrow keys', () => {
    renderPicker({ selectedMachineID: 'mini' });
    fireEvent.click(screen.getByTestId('machine-picker'));

    expect(screen.getByRole('option', { name: /minimouse/ })).toHaveFocus();
    fireEvent.keyDown(screen.getByRole('listbox', { name: 'Machine' }), { key: 'ArrowDown' });
    expect(screen.getByRole('option', { name: /swarminator/ })).toHaveFocus();
    fireEvent.keyDown(screen.getByRole('listbox', { name: 'Machine' }), { key: 'ArrowDown' });
    expect(screen.getByRole('option', { name: /This machine/ })).toHaveFocus();
  });

  it('returns focus to the control after Escape', () => {
    renderPicker();
    const trigger = screen.getByTestId('machine-picker');
    fireEvent.click(trigger);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('wraps at both ends and jumps with Home and End', () => {
    renderPicker();
    fireEvent.click(screen.getByTestId('machine-picker'));
    const listbox = screen.getByRole('listbox', { name: 'Machine' });

    fireEvent.keyDown(listbox, { key: 'ArrowUp' });
    expect(screen.getByRole('option', { name: /swarminator/ })).toHaveFocus();
    fireEvent.keyDown(listbox, { key: 'Home' });
    expect(screen.getByRole('option', { name: /This machine/ })).toHaveFocus();
    fireEvent.keyDown(listbox, { key: 'End' });
    expect(screen.getByRole('option', { name: /swarminator/ })).toHaveFocus();
    // A key with no binding must not move the selection.
    fireEvent.keyDown(listbox, { key: 'a' });
    expect(screen.getByRole('option', { name: /swarminator/ })).toHaveFocus();
  });

  it('renders nothing when there are no machines at all', () => {
    const { container } = render(<MachinePicker machines={[]} selectedMachineID="" onSelectMachine={vi.fn()} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('omits the linking row when the host offers no way to link', () => {
    renderPicker({ onAddMachine: undefined });
    fireEvent.click(screen.getByTestId('machine-picker'));
    expect(screen.queryByTestId('add-machine')).not.toBeInTheDocument();
  });

  it('falls back to the first machine when the selected id is unknown', () => {
    renderPicker({ selectedMachineID: 'retired' });
    // Better than an empty control: the picker still names something real
    // while App reconciles the stale id away.
    expect(screen.getByTestId('machine-picker')).toHaveTextContent('This machine');
  });

  it('closes and reports the choice when a row is clicked', () => {
    const { onSelectMachine } = renderPicker();
    fireEvent.click(screen.getByTestId('machine-picker'));
    fireEvent.click(screen.getByRole('option', { name: /minimouse/ }));
    expect(onSelectMachine).toHaveBeenCalledWith('mini');
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
  });

  it('closes before handing off to the linking action', () => {
    const { onAddMachine } = renderPicker();
    fireEvent.click(screen.getByTestId('machine-picker'));
    fireEvent.click(screen.getByTestId('add-machine'));
    expect(onAddMachine).toHaveBeenCalledOnce();
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
  });

  it('names the machine in view without being opened', () => {
    renderPicker({ selectedMachineID: 'mini' });
    expect(screen.getByTestId('machine-picker')).toHaveTextContent('minimouse');
    expect(screen.getByTestId('machine-picker')).toHaveAttribute('data-remote', 'true');
  });
});
