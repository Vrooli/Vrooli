import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { InvestigationScriptsPage } from './InvestigationScriptsPage';
import type { InvestigationScript } from '../../../types';

const mocks = vi.hoisted(() => ({ protoFetch: vi.fn() }));
vi.mock('../../../shared/api/apiFetch', () => ({ protoFetch: mocks.protoFetch }));
vi.mock('../../../shared/components/LazyScriptHighlighter', () => ({
  ScriptHighlighter: ({ content }: { content: string }) => <pre>{content}</pre>,
}));

const script = {
  id: 'disk-check', name: 'Disk check', description: 'Inspect disk pressure', category: 'storage', author: 'system',
  enabled: true, executionMode: 'native', createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-02T00:00:00Z',
} as unknown as InvestigationScript;
const secondScript = { ...script, id: 'network-check', name: 'Network check', description: 'Inspect network', category: 'network', enabled: false, executionMode: 'shell' } as unknown as InvestigationScript;

describe('InvestigationScriptsPage', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    Object.defineProperty(window, 'innerWidth', { value: 1440, configurable: true });
  });

  beforeEach(() => {
    mocks.protoFetch.mockReset();
    Object.defineProperty(window, 'innerWidth', { value: 1440, configurable: true });
  });

  it('loads, filters, edits, runs, saves, refreshes, and creates scripts', async () => {
    const onOpen = vi.fn();
    const onExecute = vi.fn().mockResolvedValue(undefined);
    const onSave = vi.fn().mockResolvedValue(undefined);
    mocks.protoFetch
      .mockResolvedValueOnce({ scripts: [script, secondScript] })
      .mockResolvedValueOnce({ content: '#!/bin/sh\necho disk' })
      .mockResolvedValueOnce({ scripts: [script, secondScript] });
    render(<InvestigationScriptsPage onOpenScriptEditor={onOpen} onExecuteScript={onExecute} onSaveScript={onSave} />);

    expect(await screen.findByText('Investigation Scripts Library')).toBeInTheDocument();
    expect((await screen.findAllByText((_, element) => element?.textContent?.includes('echo disk') ?? false)).length).toBeGreaterThan(0);
    const search = screen.getByPlaceholderText('Search scripts by name, id, or category');
    fireEvent.change(search, { target: { value: 'network' } });
    expect(screen.getByText('Network check')).toBeInTheDocument();
    expect(screen.getByText('1 Scripts')).toBeInTheDocument();
    fireEvent.change(search, { target: { value: '' } });

    fireEvent.click(screen.getByRole('button', { name: /^EDIT$/ }));
    const nameInput = screen.getAllByRole('textbox')[1];
    expect(nameInput).toHaveValue('Disk check');
    fireEvent.change(nameInput, { target: { value: 'Disk inspection' } });
    fireEvent.click(screen.getByRole('button', { name: 'ENABLED' }));
    expect(screen.getByRole('button', { name: 'DISABLED' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^RUN$/ }));
    await waitFor(() => { expect(onExecute).toHaveBeenCalledWith('disk-check', '#!/bin/sh\necho disk'); });
    fireEvent.click(screen.getByRole('button', { name: /^SAVE$/ }));
    await waitFor(() => { expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ name: 'Disk inspection', enabled: false }), '#!/bin/sh\necho disk'); });
    fireEvent.click(screen.getByRole('button', { name: /NEW SCRIPT/ }));
    expect(onOpen).toHaveBeenCalledWith(undefined, '', 'create');
    fireEvent.click(screen.getByRole('button', { name: /REFRESH/ }));
    await waitFor(() => { expect(mocks.protoFetch).toHaveBeenCalledTimes(3); });
  });

  it('shows load errors and the no-match state', async () => {
    mocks.protoFetch.mockRejectedValueOnce(new Error('scripts unavailable'));
    render(<InvestigationScriptsPage onOpenScriptEditor={vi.fn()} />);
    expect(await screen.findByText('Failed to load scripts: scripts unavailable')).toBeInTheDocument();
    mocks.protoFetch
      .mockResolvedValueOnce({ scripts: [script] })
      .mockResolvedValueOnce({ content: '' });
    fireEvent.click(screen.getByRole('button', { name: /REFRESH/ }));
    await screen.findByText(/1 Scripts/);
    fireEvent.change(screen.getByPlaceholderText('Search scripts by name, id, or category'), { target: { value: 'does-not-exist' } });
    expect(screen.getByText('No scripts match the current search.')).toBeInTheDocument();
  });

  it('opens a selected script directly on narrow screens and reports content failures', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 600, configurable: true });
    const onOpen = vi.fn();
    mocks.protoFetch
      .mockResolvedValueOnce({ scripts: [script] })
      .mockResolvedValueOnce({ content: 'echo ok', script })
      .mockRejectedValueOnce(new Error('content unavailable'));
    vi.spyOn(window, 'alert').mockImplementation(() => undefined);
    render(<InvestigationScriptsPage onOpenScriptEditor={onOpen} />);
    await screen.findByText('Disk check');
    fireEvent.click(screen.getByRole('button', { name: /Disk check/ }));
    await waitFor(() => { expect(onOpen).toHaveBeenCalledWith(script, 'echo ok', 'view'); });
    fireEvent.click(screen.getByRole('button', { name: /Disk check/ }));
    await waitFor(() => { expect(window.alert).toHaveBeenCalledWith('Failed to load script: content unavailable'); });
  });
});
