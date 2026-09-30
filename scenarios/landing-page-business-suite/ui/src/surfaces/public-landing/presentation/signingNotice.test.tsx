import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { DownloadChooser } from './DownloadChooser';
import { resolveSigningNotice } from '../services/downloads.service';
import type { DownloadAsset } from '../../../shared/api/types';

const asset: DownloadAsset = {
  id: 1, bundle_key: 'business_suite', app_key: 'web-console', platform: 'linux',
  artifact_url: '/artifact', release_version: '1.0.0', requires_entitlement: false,
};
const appMetadata = {
  signing_notice: {
    enabled: true,
    title: 'Preview release',
    body: 'Signing is in progress for this build.',
    link_label: 'Review the source',
    link_url: 'https://github.com/Vrooli/Vrooli/tree/master/scenarios/web-console',
  },
};

afterEach(cleanup);

describe('signing notice resolution', () => {
  it('falls back to the app default', () => {
    expect(resolveSigningNotice(appMetadata, {})?.title).toBe('Preview release');
  });
  it('honours an explicit per-platform suppression', () => {
    expect(resolveSigningNotice(appMetadata, { signing_notice: { enabled: false } })).toBeUndefined();
  });
  it('lets a platform override the app default', () => {
    const notice = resolveSigningNotice(appMetadata, { signing_notice: { enabled: true, title: 'Windows build', body: 'The Windows installer is unsigned.' } });
    expect(notice?.title).toBe('Windows build');
  });
  it('returns undefined without usable copy', () => {
    expect(resolveSigningNotice(undefined, undefined)).toBeUndefined();
    expect(resolveSigningNotice({ signing_notice: { enabled: true, title: 'Only a title' } }, undefined)).toBeUndefined();
  });
});

describe('download chooser disclosure', () => {
  const chooser = (options: DownloadAsset[], metadata?: Record<string, unknown>) => render(
    <DownloadChooser title="Downloads" description="Choose a release" options={options} selected="0"
      onSelect={vi.fn()} onPrepare={vi.fn()} state={{ status: 'idle' }} unavailableReason="Unavailable" appMetadata={metadata} />,
  );

  it('shows an info disclosure that opens to the notice copy and source link', () => {
    chooser([asset], appMetadata);
    const trigger = screen.getByRole('button', { name: 'Release trust' });
    expect(trigger).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('Signing is in progress for this build.')).not.toBeInTheDocument();
    fireEvent.click(trigger);
    expect(screen.getByRole('button', { name: 'Release trust' })).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('Signing is in progress for this build.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Review the source' })).toHaveAttribute('href', 'https://github.com/Vrooli/Vrooli/tree/master/scenarios/web-console');
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByText('Signing is in progress for this build.')).not.toBeInTheDocument();
  });

  it('renders no disclosure when the platform suppresses the app default', () => {
    chooser([{ ...asset, metadata: { signing_notice: { enabled: false } } }], appMetadata);
    expect(screen.queryByRole('button', { name: 'Release trust' })).not.toBeInTheDocument();
  });

  it('renders no disclosure when nothing is configured', () => {
    chooser([asset]);
    expect(screen.queryByRole('button', { name: 'Release trust' })).not.toBeInTheDocument();
  });
});
