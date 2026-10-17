import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import { PstoreArtifactList } from './PstoreArtifactList';
import { BootHistoryTimeline } from './BootHistoryTimeline';

describe('forensics artifact variants', () => {
  it('renders unavailable, empty, and populated pstore states', () => {
    const { rerender } = render(<PstoreArtifactList envelope={{ available: false, reason: 'not mounted', generatedAt: '' }} />);
    expect(screen.getByText('not mounted')).toBeInTheDocument();
    rerender(<PstoreArtifactList envelope={{ available: true, generatedAt: '', data: { path: '/sys/fs/pstore', entries: [] } }} />);
    expect(screen.getByText(/No artifacts/)).toBeInTheDocument();
    rerender(<PstoreArtifactList envelope={{ available: true, generatedAt: '', data: { path: '/sys/fs/pstore', entries: [{ name: 'dmesg-1', kind: 'dmesg', size: 10, modified: '' }] } }} />);
    expect(screen.getByText('dmesg-1')).toBeInTheDocument();
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('renders unavailable, empty, and classified boot history states', () => {
    const { rerender } = render(<BootHistoryTimeline envelope={{ available: false, generatedAt: '', reason: 'journal unavailable' }} />);
    expect(screen.getByText('journal unavailable')).toBeInTheDocument();
    rerender(<BootHistoryTimeline envelope={{ available: true, generatedAt: '', data: { boots: [] } }} />);
    expect(screen.getByText('No boot records available.')).toBeInTheDocument();
    rerender(<BootHistoryTimeline envelope={{ available: true, generatedAt: '', data: { boots: [
      { index: 0, bootId: '', firstEntry: '2026-01-01T00:00:00Z', lastEntry: '2026-01-01T00:01:00Z', clean: true },
      { index: 1, bootId: 'boot-1', firstEntry: '2026-01-01T00:00:00Z', lastEntry: '2026-01-01T00:01:00Z', clean: true },
      { index: 2, bootId: 'boot-2', firstEntry: '2026-01-01T00:00:00Z', lastEntry: '2026-01-01T00:01:00Z', clean: false, reason: 'crash' },
    ] } }} />);
    expect(screen.getByText('Current boot')).toBeInTheDocument();
    expect(screen.getByText('Clean shutdown')).toBeInTheDocument();
    expect(screen.getByText('Unclean shutdown')).toBeInTheDocument();
  });
});
