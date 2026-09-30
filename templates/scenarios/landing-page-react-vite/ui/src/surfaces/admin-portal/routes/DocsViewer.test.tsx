import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../../../test-utils';
import { DocsViewer } from './DocsViewer';

const { mockGetDocsTree, mockGetDocContent } = vi.hoisted(() => ({
  mockGetDocsTree: vi.fn(),
  mockGetDocContent: vi.fn(),
}));

vi.mock('../../../shared/api', () => ({
  getDocsTree: mockGetDocsTree,
  getDocContent: mockGetDocContent,
}));

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: React.ReactNode }) => <div data-testid="admin-layout">{children}</div>,
}));

const tree = [
  {
    path: 'guide',
    name: 'guide',
    isDir: true,
    children: [{ path: 'guide/intro.md', name: 'intro.md', isDir: false }],
  },
  { path: 'readme.md', name: 'readme.md', isDir: false },
];

const richMarkdown = [
  '# Getting Started',
  '## Configuration',
  '### Options',
  '#### Details',
  'A paragraph with **bold**, *italic*, `code`, an [external](https://example.com) and a [local](/docs) link.',
  '> Remember to save',
  '---',
  '- first bullet',
  '- second bullet',
  '1. step one',
  '2. step two',
  '| Name | Value |',
  '| --- | --- |',
  '| alpha | 1 |',
  '```',
  'const answer = 42;',
  '```',
].join('\n');

beforeEach(() => {
  vi.clearAllMocks();
  mockGetDocsTree.mockResolvedValue(tree);
  mockGetDocContent.mockResolvedValue({ title: 'Intro', content: richMarkdown });
});

describe('DocsViewer', () => {
  it('loads the tree, auto-expands root dirs, and auto-selects the first file', async () => {
    renderWithProviders(<DocsViewer />);
    await waitFor(() => expect(mockGetDocsTree).toHaveBeenCalled());
    expect(await screen.findByText('Template Documentation')).toBeInTheDocument();
    // Root directory auto-expanded so its child file is visible.
    expect(screen.getByText('intro')).toBeInTheDocument();
    expect(mockGetDocContent).toHaveBeenCalledWith('guide/intro.md');
  });

  it('renders the full markdown feature set for the selected doc', async () => {
    renderWithProviders(<DocsViewer />);
    expect(await screen.findByRole('heading', { name: 'Getting Started', level: 1 })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Configuration', level: 2 })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Options', level: 3 })).toBeInTheDocument();
    expect(screen.getByText('bold')).toBeInTheDocument();
    expect(screen.getByText('first bullet')).toBeInTheDocument();
    expect(screen.getByText('step one')).toBeInTheDocument();
    // Table header + body rendered.
    const table = screen.getByRole('table');
    expect(within(table).getByText('Name')).toBeInTheDocument();
    expect(within(table).getByText('alpha')).toBeInTheDocument();
    // Blockquote and code block.
    expect(screen.getByText('Remember to save')).toBeInTheDocument();
    expect(screen.getByText('const answer = 42;')).toBeInTheDocument();
    // External link opens in a new tab, local link does not.
    const external = screen.getByRole('link', { name: 'external' });
    expect(external).toHaveAttribute('target', '_blank');
    expect(screen.getByRole('link', { name: 'local' })).not.toHaveAttribute('target');
  });

  it('collapses and re-expands a directory when its toggle is clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocsViewer />);
    await screen.findByText('intro');

    await user.click(screen.getByRole('button', { name: /guide/i }));
    expect(screen.queryByText('intro')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /guide/i }));
    expect(screen.getByText('intro')).toBeInTheDocument();
  });

  it('loads a different document when a file is selected', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocsViewer />);
    await screen.findByText('Getting Started');

    mockGetDocContent.mockResolvedValueOnce({ title: 'Readme', content: '# Readme body' });
    await user.click(screen.getByText('readme'));

    await waitFor(() => expect(mockGetDocContent).toHaveBeenCalledWith('readme.md'));
    expect(await screen.findByRole('heading', { name: 'Readme body', level: 1 })).toBeInTheDocument();
  });

  it('shows the empty state when no docs exist', async () => {
    mockGetDocsTree.mockResolvedValue([]);
    renderWithProviders(<DocsViewer />);
    expect(await screen.findByText('No Documentation Found')).toBeInTheDocument();
  });

  it('surfaces an error when the tree fails to load', async () => {
    mockGetDocsTree.mockRejectedValue(new Error('tree exploded'));
    renderWithProviders(<DocsViewer />);
    expect(await screen.findByText('tree exploded')).toBeInTheDocument();
  });

  it('reloads the tree when Refresh is clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocsViewer />);
    await screen.findByText('Template Documentation');
    mockGetDocsTree.mockClear();
    await user.click(screen.getByTestId('docs-refresh'));
    await waitFor(() => expect(mockGetDocsTree).toHaveBeenCalledTimes(1));
  });
});
