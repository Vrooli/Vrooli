import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../../../test-utils';
import { DownloadSettings } from './DownloadSettings';

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: React.ReactNode }) => <div data-testid="admin-layout">{children}</div>,
}));

const { mockList, mockCreate, mockSave } = vi.hoisted(() => ({
  mockList: vi.fn(),
  mockCreate: vi.fn(),
  mockSave: vi.fn(),
}));

vi.mock('../../../shared/api', () => ({
  listDownloadAppsAdmin: mockList,
  createDownloadAppAdmin: mockCreate,
  saveDownloadAppAdmin: mockSave,
}));

const app = () => ({
  appKey: 'suite',
  name: 'Automation Suite',
  tagline: 'Ship faster',
  description: 'A bundle',
  installOverview: 'Overview',
  installSteps: ['Download', 'Run installer'],
  displayOrder: 0,
  storefronts: [
    { store: 'app_store', url: 'https://apps.apple.com/suite', label: 'App Store', badge: '' },
  ],
  platforms: [
    {
      platform: 'mac',
      artifactUrl: 'https://cdn/suite.dmg',
      releaseVersion: '1.2.0',
      releaseNotes: 'Notes',
      requiresEntitlement: true,
      metadata: { enabled: true, size_mb: 120 },
    },
  ],
});

beforeEach(() => {
  vi.clearAllMocks();
  mockList.mockResolvedValue([app()]);
  mockSave.mockResolvedValue(app());
  mockCreate.mockResolvedValue({ ...app(), appKey: 'new_bundle', name: 'New Bundle App' });
  vi.spyOn(window, 'open').mockImplementation(() => null);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('DownloadSettings', () => {
  it('loads apps and renders the health overview', async () => {
    renderWithProviders(<DownloadSettings />);
    expect(await screen.findByTestId('download-card-suite')).toBeInTheDocument();
    const health = screen.getByTestId('downloads-health');
    expect(within(health).getByText('1 app')).toBeInTheDocument();
    // Only the mac platform is configured; the other two are missing.
    expect(within(health).getByText('1 platform')).toBeInTheDocument();
    expect(within(health).getByText('2 missing')).toBeInTheDocument();
    expect(within(health).getByText('1 store link')).toBeInTheDocument();
    expect(within(health).getByText('All saved')).toBeInTheDocument();
  });

  it('shows the empty state when no apps are configured', async () => {
    mockList.mockResolvedValue([]);
    renderWithProviders(<DownloadSettings />);
    expect(await screen.findByTestId('downloads-empty-state')).toBeInTheDocument();
    expect(screen.getByText('No download apps configured yet')).toBeInTheDocument();
  });

  it('adds a new app card and surfaces the unsaved-changes banner once edited', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    await user.click(screen.getByTestId('downloads-add-app'));
    const newCard = (await screen.findByText('New Bundle App')).closest(
      '[data-testid^="download-card-"]',
    ) as HTMLElement;
    // A freshly added app is not dirty until a field changes.
    await user.type(within(newCard).getByDisplayValue('New Bundle App'), ' X');
    expect(await screen.findByTestId('download-settings-dirty-banner')).toBeInTheDocument();
    expect(screen.getByTestId('downloads-save-all')).toBeInTheDocument();
  });

  it('marks an app dirty when a field changes and clears it on reset', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    const nameInput = screen.getByDisplayValue('Automation Suite');
    await user.type(nameInput, ' Pro');
    expect(await screen.findByTestId('download-settings-dirty-banner')).toBeInTheDocument();

    await user.click(screen.getByTestId('download-reset-suite'));
    expect(screen.queryByTestId('download-settings-dirty-banner')).not.toBeInTheDocument();
    expect(screen.getByDisplayValue('Automation Suite')).toBeInTheDocument();
  });

  it('saves an existing app through the update endpoint', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    await user.type(screen.getByDisplayValue('Automation Suite'), ' v2');
    await user.click(screen.getByTestId('download-save-suite'));

    await waitFor(() =>
      expect(mockSave).toHaveBeenCalledWith('suite', expect.objectContaining({ appKey: 'suite' })),
    );
  });

  it('creates a brand new app through the create endpoint', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    await user.click(screen.getByTestId('downloads-add-app'));
    const cardRoot = (await screen.findByText('New Bundle App')).closest(
      '[data-testid^="download-card-"]',
    ) as HTMLElement;
    // Edit a field so the card becomes dirty and its Save button enables.
    await user.type(within(cardRoot).getByDisplayValue('New Bundle App'), ' Edited');
    await user.click(within(cardRoot).getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(mockCreate).toHaveBeenCalled());
  });

  it('blocks saving a new app when the app key is cleared', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    await user.click(screen.getByTestId('downloads-add-app'));
    const newCard = (await screen.findByText('New Bundle App')).closest(
      '[data-testid^="download-card-"]',
    ) as HTMLElement;
    const appKeyInput = within(newCard).getByPlaceholderText('automation_suite');
    // Clearing the pre-filled key both makes the card dirty and empties the key.
    await user.clear(appKeyInput);
    await user.click(within(newCard).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('App key is required before saving.')).toBeInTheDocument();
    expect(mockCreate).not.toHaveBeenCalled();
  });

  it('reports "no data" when a save resolves without a payload', async () => {
    const user = userEvent.setup();
    mockSave.mockResolvedValue(null);
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');
    await user.type(screen.getByDisplayValue('Automation Suite'), ' x');
    await user.click(screen.getByTestId('download-save-suite'));
    expect(await screen.findByText('Save returned no data')).toBeInTheDocument();
  });

  it('surfaces a save error returned by the API', async () => {
    const user = userEvent.setup();
    mockSave.mockRejectedValue(new Error('save exploded'));
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    await user.type(screen.getByDisplayValue('Automation Suite'), ' edit');
    await user.click(screen.getByTestId('download-save-suite'));
    expect(await screen.findByText('save exploded')).toBeInTheDocument();
  });

  it('surfaces a load error', async () => {
    mockList.mockRejectedValue(new Error('load down'));
    renderWithProviders(<DownloadSettings />);
    expect(await screen.findByText('load down')).toBeInTheDocument();
  });

  it('previews the public landing in a new tab', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');
    await user.click(screen.getByTestId('downloads-preview'));
    expect(window.open).toHaveBeenCalledWith('/', '_blank', 'noopener,noreferrer');
  });

  it('edits platform installer fields and toggles platform enablement', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    // The MAC platform is enabled, so its fields are editable and uniquely
    // identifiable by their seeded values.
    await user.type(screen.getByDisplayValue('https://cdn/suite.dmg'), '?v=2');
    expect(screen.getByDisplayValue('https://cdn/suite.dmg?v=2')).toBeInTheDocument();
    const version = screen.getByDisplayValue('1.2.0');
    await user.clear(version);
    await user.type(version, '2.0.0');
    expect(screen.getByDisplayValue('2.0.0')).toBeInTheDocument();
    // Release notes edit exercises the textarea change handler.
    await user.type(screen.getByDisplayValue('Notes'), ' updated');
    // App-level description / install fields and platform size.
    await user.type(screen.getByDisplayValue('A bundle'), '!');
    await user.type(screen.getByDisplayValue('Overview'), '!');
    const size = screen.getByDisplayValue('120');
    await user.clear(size);
    await user.type(size, '256');
    expect(screen.getByDisplayValue('256')).toBeInTheDocument();

    // Enabled checkboxes are ordered storefronts first (apple, google), then
    // platforms (windows, mac, linux). Toggle the WINDOWS platform on.
    const enabledToggles = screen.getAllByRole('checkbox', { name: 'Enabled' });
    await user.click(enabledToggles[2]!);
    expect(enabledToggles[2]!).toBeChecked();

    // Toggle the MAC "requires entitlement" flag.
    const entitlementToggles = screen.getAllByRole('checkbox', { name: /Requires entitlement/i });
    await user.click(entitlementToggles[1]!);
    expect(entitlementToggles[1]!).not.toBeChecked();
  });

  it('edits storefront fields and toggles a storefront on', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    // Apple storefront is enabled; edit its URL and label.
    const appleUrl = screen.getByDisplayValue('https://apps.apple.com/suite');
    await user.type(appleUrl, '/v2');
    expect(screen.getByDisplayValue('https://apps.apple.com/suite/v2')).toBeInTheDocument();

    // Enable the Google Play storefront (second "Enabled" checkbox: apple, google, ...).
    const enabledToggles = screen.getAllByRole('checkbox', { name: 'Enabled' });
    await user.click(enabledToggles[1]!);
    expect(enabledToggles[1]!).toBeChecked();

    // Now the Google Play label/url/badge inputs are editable.
    await user.type(screen.getByDisplayValue('Google Play'), ' Store');
    // Edit the Apple badge (initially empty) via the badge input following the URL.
    const appleBadge = screen.getByDisplayValue('App Store');
    await user.type(appleBadge, ' US');
    expect(screen.getByDisplayValue('App Store US')).toBeInTheDocument();

    // Exercise the drag-leave handler on a card.
    const card = screen.getByTestId('download-card-suite');
    fireEvent.dragLeave(card);
  });

  it('reorders apps via drag and drop', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    await user.click(screen.getByTestId('downloads-add-app'));
    const cards = screen.getAllByTestId(/^download-card-/);
    expect(cards.length).toBe(2);

    const store: Record<string, string> = {};
    const dataTransfer = {
      effectAllowed: '',
      dropEffect: '',
      setData: (k: string, v: string) => {
        store[k] = v;
      },
      getData: (k: string) => store[k] ?? '',
    };

    fireEvent.dragStart(cards[0]!, { dataTransfer });
    fireEvent.dragOver(cards[1]!, { dataTransfer });
    fireEvent.drop(cards[1]!, { dataTransfer });
    fireEvent.dragEnd(cards[0]!, { dataTransfer });

    // Reordering marks the cards dirty (displayOrder changed).
    expect(await screen.findByTestId('download-settings-dirty-banner')).toBeInTheDocument();
  });

  it('deserializes an app with both storefronts and implicit platform enablement', async () => {
    mockList.mockResolvedValue([
      {
        appKey: 'suite',
        name: 'Automation Suite',
        tagline: '',
        description: '',
        installOverview: '',
        installSteps: [],
        displayOrder: 0,
        storefronts: [
          { store: 'app_store', url: 'https://apps.apple.com/x', label: 'App Store', badge: 'New' },
          { store: 'play_store', url: 'https://play.google.com/x', label: 'Google Play', badge: '' },
        ],
        platforms: [
          // Implicit enablement via content (no metadata.enabled), string size.
          { platform: 'mac', artifactUrl: 'https://cdn/x.dmg', releaseVersion: '1.0.0', releaseNotes: '', requiresEntitlement: false, metadata: { size_mb: '88' } },
          // Explicitly disabled despite content.
          { platform: 'windows', artifactUrl: 'https://cdn/x.exe', releaseVersion: '1.0.0', releaseNotes: '', requiresEntitlement: true, metadata: { enabled: false } },
        ],
      },
    ]);
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');
    const health = screen.getByTestId('downloads-health');
    // Both storefronts have URLs -> two store links.
    expect(within(health).getByText('2 store links')).toBeInTheDocument();
  });

  it('reports a per-card error when one app in a bulk save fails', async () => {
    const user = userEvent.setup();
    mockSave.mockRejectedValueOnce(new Error('bulk boom'));
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');
    await user.type(screen.getByDisplayValue('Automation Suite'), ' x');
    await user.click(screen.getByTestId('downloads-save-all'));
    expect(await screen.findByText('bulk boom')).toBeInTheDocument();
  });

  it('saves all dirty apps at once', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DownloadSettings />);
    await screen.findByTestId('download-card-suite');

    await user.type(screen.getByDisplayValue('Automation Suite'), ' bulk');
    await user.click(screen.getByTestId('downloads-save-all'));
    await waitFor(() => expect(mockSave).toHaveBeenCalled());
  });
});
