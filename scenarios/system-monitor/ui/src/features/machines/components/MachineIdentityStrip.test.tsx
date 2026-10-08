import { describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, screen } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils/renderWithProviders';
import { MachineIdentityStrip } from './MachineIdentityStrip';
import { MachinePresenceNote } from './MachinePresenceNote';
import type { Machine } from '../../../types';

const minimouse: Machine = {
  id: '25c7e426',
  name: 'minimouse',
  os: 'darwin',
  arch: 'amd64',
  online: true,
  heartbeat_fresh: true,
  heartbeat_age_seconds: 4,
  dispatchable: true,
  status: 'NODE_STATUS_ONLINE',
  scopes: ['*:read', '*:write'],
  grant: 'Read and operate; destructive actions withheld'
};

const baseProps = {
  machine: minimouse,
  isStale: false,
  lastSuccessfulFetch: new Date('2026-08-27T14:02:11Z'),
  onRetry: vi.fn(),
  onBackToLocal: vi.fn()
};

const noteProps = {
  machine: minimouse,
  isStale: false,
  lastSuccessfulFetch: new Date('2026-08-27T14:02:11Z'),
  retryAttempt: 0,
  retryIntervalSeconds: 15,
  lastAttemptAt: new Date('2026-08-27T14:02:11Z')
};

const unreachable: Machine = {
  ...minimouse,
  name: 'swarminator',
  dispatchable: false,
  heartbeat_fresh: false,
  heartbeat_age_seconds: 639479,
  readiness: [
    { identity: 'registry_record', passed: true },
    { identity: 'heartbeat_fresh', passed: false },
    { identity: 'channel_held', passed: false }
  ]
};

describe('MachineIdentityStrip', () => {
  it('says which machine is in view and that it is not this computer', () => {
    renderWithProviders(<MachineIdentityStrip {...baseProps} />);
    // The failure this prevents is reading one machine's numbers and acting on
    // another, so the subject and the fact that it is remote are both stated.
    expect(screen.getByRole('status')).toHaveTextContent('Viewing minimouse — darwin/amd64, not this computer');
    expect(screen.getByRole('status')).toHaveTextContent('operate');
  });

  it('always offers the way back to this computer', () => {
    const onBackToLocal = vi.fn();
    renderWithProviders(<MachineIdentityStrip {...baseProps} onBackToLocal={onBackToLocal} />);
    fireEvent.click(screen.getByTestId('back-to-local'));
    expect(onBackToLocal).toHaveBeenCalledOnce();
  });

  it('does not offer a retry while the machine is answering', () => {
    renderWithProviders(<MachineIdentityStrip {...baseProps} />);
    expect(screen.queryByRole('button', { name: /Retry now/ })).not.toBeInTheDocument();
  });

  it('says the readings are the last ones when the subject goes quiet', () => {
    renderWithProviders(<MachineIdentityStrip {...baseProps} isStale />);
    const strip = screen.getByRole('status');
    expect(strip).toHaveTextContent('minimouse stopped responding');
    expect(strip).toHaveTextContent('showing its last reading');
    expect(screen.getByRole('button', { name: /Retry now/ })).toBeInTheDocument();
  });

  it('scales the silence to how long it has lasted', () => {
    const minutesAgo = new Date(Date.now() - 2 * 60 * 1000);
    renderWithProviders(<MachineIdentityStrip {...baseProps} isStale lastSuccessfulFetch={minutesAgo} />);
    expect(screen.getByRole('status')).toHaveTextContent('stopped responding 2 minutes ago');
    cleanup();

    const hoursAgo = new Date(Date.now() - 3 * 60 * 60 * 1000);
    renderWithProviders(<MachineIdentityStrip {...baseProps} isStale lastSuccessfulFetch={hoursAgo} />);
    expect(screen.getByRole('status')).toHaveTextContent('stopped responding 3 hours ago');
    cleanup();

    renderWithProviders(<MachineIdentityStrip {...baseProps} isStale lastSuccessfulFetch={new Date(Date.now() - 1000)} />);
    expect(screen.getByRole('status')).toHaveTextContent('stopped responding 1 seconds ago');
  });

  it('omits the elapsed phrase when there has never been a reading', () => {
    renderWithProviders(<MachineIdentityStrip {...baseProps} isStale lastSuccessfulFetch={null} />);
    expect(screen.getByRole('status')).toHaveTextContent('minimouse stopped responding — showing its last reading');
  });

  it('separates cannot-be-reached from stopped-answering', () => {
    renderWithProviders(<MachineIdentityStrip {...baseProps} machine={unreachable} />);
    const strip = screen.getByRole('status');
    expect(strip).toHaveTextContent('swarminator is not responding — no readings are available from it');
    // A machine that cannot be dispatched to has no last reading to show, so it
    // must not borrow the "showing its last reading" words.
    expect(strip).not.toHaveTextContent('showing its last reading');
  });

  it('offers no grant chip for a machine it cannot reach', () => {
    renderWithProviders(<MachineIdentityStrip {...baseProps} machine={unreachable} />);
    // A grant that cannot be exercised is not a fact worth showing beside a
    // machine that is not answering.
    expect(screen.getByRole('status')).not.toHaveTextContent('operate');
  });
});

describe('MachinePresenceNote', () => {
  it('stays out of the way while the machine is answering', () => {
    const { container } = renderWithProviders(<MachinePresenceNote {...noteProps} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('tells the reader Vrooli reconnects on its own, and how far along it is', () => {
    renderWithProviders(<MachinePresenceNote {...noteProps} isStale retryAttempt={8} />);
    const note = screen.getByTestId('machine-presence-note');
    // Without this sentence a person power-cycles a machine that was coming
    // back by itself.
    expect(note).toHaveTextContent('will reconnect on its own when the machine is reachable again');
    expect(note).toHaveTextContent('attempt 8');
  });

  it('quotes the polling period it was given rather than a fixed promise', () => {
    renderWithProviders(<MachinePresenceNote {...noteProps} isStale retryIntervalSeconds={30} />);
    expect(screen.getByTestId('machine-presence-note')).toHaveTextContent('retrying every 30 seconds');
  });

  it('names the readiness facts behind an unreachable machine', () => {
    renderWithProviders(<MachinePresenceNote {...noteProps} machine={unreachable} />);
    const note = screen.getByTestId('machine-presence-note');
    expect(note).toHaveTextContent('Nothing to read from swarminator');
    expect(note).toHaveTextContent('heartbeat fresh');
    // It cannot report vitals, so it must not be described as if it does.
    expect(note).not.toHaveTextContent('reports its live vitals');
  });

  it('says there is no reading yet rather than inventing a stamp', () => {
    renderWithProviders(<MachinePresenceNote {...noteProps} isStale lastSuccessfulFetch={null} lastAttemptAt={null} retryAttempt={0} />);
    const note = screen.getByTestId('machine-presence-note');
    expect(note).toHaveTextContent('no reading yet');
    expect(note).not.toHaveTextContent('next retry');
    expect(note).not.toHaveTextContent('attempt');
  });

  it('counts down to the next attempt from the attempt it was told about', () => {
    renderWithProviders(<MachinePresenceNote {...noteProps} isStale lastAttemptAt={new Date(Date.now() - 5000)} retryIntervalSeconds={15} />);
    expect(screen.getByTestId('machine-presence-note')).toHaveTextContent('next retry in 10s');
  });

  it('renders an unreachable machine with no readiness facts at all', () => {
    renderWithProviders(<MachinePresenceNote {...noteProps} machine={{ ...unreachable, readiness: undefined }} />);
    expect(screen.getByTestId('machine-presence-note')).toHaveTextContent('Nothing to read from swarminator');
  });

  it('says "never seen" for a machine that never reported an age', () => {
    renderWithProviders(<MachinePresenceNote {...noteProps} machine={{ ...unreachable, heartbeat_age_seconds: undefined }} />);
    expect(screen.getByTestId('machine-presence-note')).toHaveTextContent('never seen');
  });

  it('prefers unreachable over stale when both could apply', () => {
    renderWithProviders(<MachinePresenceNote {...noteProps} machine={unreachable} isStale />);
    // There is no "last reading" to freeze for a channel that never opened.
    expect(screen.getByTestId('machine-presence-note')).toHaveAttribute('data-tone', 'unreachable');
  });
});
