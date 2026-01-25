import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { ReactElement } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DocsViewer } from './DocsViewer';
import type { DocEntry, DocContent } from '../../../shared/api';

const { mockGetDocsTree, mockGetDocContent } = vi.hoisted(() => ({
  mockGetDocsTree: vi.fn(),
  mockGetDocContent: vi.fn(),
}));

vi.mock('../components/RuntimeSignalStrip', () => ({
  RuntimeSignalStrip: () => <div data-testid="runtime-signal-mock" />,
}));

vi.mock('../../../shared/api', () => ({
  getDocsTree: mockGetDocsTree,
  getDocContent: mockGetDocContent,
  adminLogout: vi.fn(),
}));

const mockTree: DocEntry[] = [
  {
    name: 'getting-started',
    path: 'getting-started',
    isDir: true,
    children: [
      { name: 'installation.md', path: 'getting-started/installation.md', isDir: false },
      { name: 'configuration.md', path: 'getting-started/configuration.md', isDir: false },
    ],
  },
  {
    name: 'api',
    path: 'api',
    isDir: true,
    children: [
      { name: 'endpoints.md', path: 'api/endpoints.md', isDir: false },
    ],
  },
  { name: 'README.md', path: 'README.md', isDir: false },
];

const mockDoc: DocContent = {
  path: 'getting-started/installation.md',
  title: 'Installation Guide',
  content: '# Installation\n\nThis is the installation guide.\n\n## Prerequisites\n\nYou need Node.js installed.',
};

const renderWithRouter = (component: ReactElement, route = '/admin/docs') => {
  return render(<MemoryRouter initialEntries={[route]}>{component}</MemoryRouter>);
};

describe('DocsViewer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetDocsTree.mockResolvedValue(mockTree);
    mockGetDocContent.mockResolvedValue(mockDoc);
  });

  describe('loading state', () => {
    it('shows loading state initially', () => {
      mockGetDocsTree.mockReturnValue(new Promise(() => {})); // Never resolves
      renderWithRouter(<DocsViewer />);

      expect(screen.getByTestId('docs-loading')).toBeInTheDocument();
      expect(screen.getByText('Loading documentation...')).toBeInTheDocument();
    });
  });

  describe('error state', () => {
    it('shows error message when tree loading fails', async () => {
      mockGetDocsTree.mockRejectedValue(new Error('Network error'));
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByTestId('docs-error')).toBeInTheDocument();
      });
      expect(screen.getByText('Network error')).toBeInTheDocument();
    });
  });

  describe('empty state', () => {
    it('shows empty state when no docs found', async () => {
      mockGetDocsTree.mockResolvedValue([]);
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByTestId('docs-empty')).toBeInTheDocument();
      });
      expect(screen.getByText('No Documentation Found')).toBeInTheDocument();
    });
  });

  describe('content display', () => {
    it('renders the documentation tree after loading', async () => {
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByTestId('docs-content')).toBeInTheDocument();
      });
      expect(screen.getByTestId('docs-tree')).toBeInTheDocument();
    });

    it('shows folder names in the tree', async () => {
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByText('getting-started')).toBeInTheDocument();
      });
      expect(screen.getByText('api')).toBeInTheDocument();
    });

    it('auto-selects and loads first document', async () => {
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(mockGetDocContent).toHaveBeenCalledWith('getting-started/installation.md');
      });

      await waitFor(() => {
        expect(screen.getByText('Installation Guide')).toBeInTheDocument();
      });
    });

    it('loads requested document when doc query param is provided', async () => {
      renderWithRouter(<DocsViewer />, '/admin/docs?doc=api/endpoints.md');

      await waitFor(() => {
        expect(mockGetDocContent).toHaveBeenCalledWith('api/endpoints.md');
      });
    });

    it('renders markdown content', async () => {
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByText('Installation')).toBeInTheDocument();
      });
      expect(screen.getByText('Prerequisites')).toBeInTheDocument();
    });

    it('shows document path in description', async () => {
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByText('docs/getting-started/installation.md')).toBeInTheDocument();
      });
    });
  });

  describe('tree navigation', () => {
    it('selects a different document when clicked', async () => {
      const user = userEvent.setup();
      const anotherDoc: DocContent = {
        path: 'api/endpoints.md',
        title: 'API Endpoints',
        content: '# API Endpoints\n\nDocumentation for API endpoints.',
      };
      mockGetDocContent.mockResolvedValueOnce(mockDoc).mockResolvedValueOnce(anotherDoc);

      renderWithRouter(<DocsViewer />);

      // Wait for initial load
      await waitFor(() => {
        expect(screen.getByText('Installation Guide')).toBeInTheDocument();
      });

      // Click on the endpoints file
      const endpointsButton = screen.getByText('endpoints');
      await user.click(endpointsButton);

      await waitFor(() => {
        expect(mockGetDocContent).toHaveBeenCalledWith('api/endpoints.md');
      });

      // Wait for the new document path to show in the description
      await waitFor(() => {
        expect(screen.getByText('docs/api/endpoints.md')).toBeInTheDocument();
      });
    });

    it('toggles folder expansion when clicked', async () => {
      const user = userEvent.setup();
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByText('getting-started')).toBeInTheDocument();
      });

      // Folder should be expanded initially (root dirs auto-expand)
      expect(screen.getByText('installation')).toBeInTheDocument();

      // Click to collapse
      const folderButton = screen.getByText('getting-started');
      await user.click(folderButton);

      // Children should be hidden
      await waitFor(() => {
        expect(screen.queryByText('installation')).not.toBeInTheDocument();
      });

      // Click to expand again
      await user.click(folderButton);

      await waitFor(() => {
        expect(screen.getByText('installation')).toBeInTheDocument();
      });
    });
  });

  describe('refresh functionality', () => {
    it('reloads tree when refresh button is clicked', async () => {
      const user = userEvent.setup();
      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByTestId('docs-content')).toBeInTheDocument();
      });

      mockGetDocsTree.mockClear();

      const refreshButton = screen.getByTestId('docs-refresh');
      await user.click(refreshButton);

      await waitFor(() => {
        expect(mockGetDocsTree).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe('document loading states', () => {
    it('shows loading state when loading a document', async () => {
      let resolveDoc: (doc: DocContent) => void;
      mockGetDocContent.mockReturnValue(
        new Promise<DocContent>((resolve) => {
          resolveDoc = resolve;
        })
      );

      renderWithRouter(<DocsViewer />);

      await waitFor(() => {
        expect(screen.getByTestId('docs-loading-doc')).toBeInTheDocument();
      });

      // Resolve the document
      resolveDoc!(mockDoc);

      await waitFor(() => {
        expect(screen.queryByTestId('docs-loading-doc')).not.toBeInTheDocument();
      });
    });
  });

  describe('header', () => {
    it('renders the header with title', async () => {
      renderWithRouter(<DocsViewer />);

      expect(screen.getByTestId('docs-header')).toBeInTheDocument();
      expect(screen.getByText('Template Documentation')).toBeInTheDocument();
    });
  });
});
