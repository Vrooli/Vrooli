import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { SectionEditor } from './SectionEditor';
import * as api from '../../../shared/api';

// Mock the API module
vi.mock('../../../shared/api', () => ({
  getSection: vi.fn(),
  updateSection: vi.fn(),
  createSection: vi.fn(),
}));

// Mock useParams
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ variantSlug: 'test-variant', sectionId: '1' }),
  };
});

const mockSection = {
  id: 1,
  variant_id: 1,
  section_type: 'hero' as const,
  enabled: true,
  order: 0,
  content: {
    title: 'Test Title',
    subtitle: 'Test Subtitle',
    cta_text: 'Get Started',
    cta_url: '/signup',
  },
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe('SectionEditor [REQ:CUSTOM-SPLIT,CUSTOM-LIVE]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getSection).mockResolvedValue(mockSection);
  });

  const renderEditor = () => {
    return render(
      <BrowserRouter>
        <SectionEditor />
      </BrowserRouter>
    );
  };

  it('[REQ:CUSTOM-SPLIT] should render split layout with form and preview columns', async () => {
    renderEditor();

    // Wait for section to load
    await waitFor(() => {
      expect(screen.getByTestId('section-form')).toBeInTheDocument();
    });

    // Verify form column exists
    const formColumn = screen.getByTestId('section-form');
    expect(formColumn).toBeInTheDocument();

    // Verify preview column exists
    const previewColumn = screen.getByTestId('section-preview');
    expect(previewColumn).toBeInTheDocument();

    // Verify they are in a grid layout (both should be present)
    const formParent = formColumn.parentElement;
    expect(formParent?.className).toContain('grid');
  });

  it('[REQ:CUSTOM-SPLIT] should have responsive grid classes for mobile stacking', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('section-form')).toBeInTheDocument();
    });

    const formColumn = screen.getByTestId('section-form');
    const gridContainer = formColumn.parentElement;

    // Check for lg:grid-cols-2 class (splits at large breakpoint)
    expect(gridContainer?.className).toContain('lg:grid-cols-2');
  });

  it('[REQ:CUSTOM-LIVE] should render live preview with debounced content updates', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('section-preview')).toBeInTheDocument();
    });

    const preview = screen.getByTestId('section-preview');

    // Wait for debounced content to update (300ms debounce + buffer)
    await waitFor(
      () => {
        expect(preview).toHaveTextContent('Test Title');
      },
      { timeout: 500 }
    );

    expect(preview).toHaveTextContent('Test Subtitle');
    expect(preview).toHaveTextContent('Get Started');
  });

  it('[REQ:CUSTOM-LIVE] should display 300ms debounce indicator', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByText(/Updates in 300ms/i)).toBeInTheDocument();
    });
  });

  it('should load section data on mount', async () => {
    renderEditor();

    await waitFor(() => {
      expect(api.getSection).toHaveBeenCalledWith(1);
    });

    // Verify form fields are populated
    const titleInput = screen.getByTestId('content-title-input') as HTMLInputElement;
    expect(titleInput.value).toBe('Test Title');

    const subtitleInput = screen.getByTestId('content-subtitle-input') as HTMLTextAreaElement;
    expect(subtitleInput.value).toBe('Test Subtitle');
  });

  it('should display section type selector', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('section-type-input')).toBeInTheDocument();
    });

    const typeSelect = screen.getByTestId('section-type-input') as HTMLSelectElement;
    expect(typeSelect.value).toBe('hero');
  });

  it('should render hero-specific preview when section type is hero', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('section-preview')).toBeInTheDocument();
    });

    const preview = screen.getByTestId('section-preview');

    // Hero preview should show title as h1
    const heroTitle = preview.querySelector('h1');
    expect(heroTitle).toBeInTheDocument();

    // Wait for debounced content to update (300ms debounce + buffer)
    await waitFor(
      () => {
        expect(heroTitle?.textContent).toContain('Test Title');
      },
      { timeout: 500 }
    );
  });

  it('should show disabled indicator when section is disabled', async () => {
    const disabledSection = { ...mockSection, enabled: false };
    vi.mocked(api.getSection).mockResolvedValue(disabledSection);

    renderEditor();

    await waitFor(() => {
      expect(screen.getByText(/Section is currently disabled/i)).toBeInTheDocument();
    });
  });

  it('should have save button', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('save-section')).toBeInTheDocument();
    });

    const saveButton = screen.getByTestId('save-section');
    expect(saveButton).toHaveTextContent('Save');
  });

  it('should show sticky positioning for preview on large screens', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('section-preview')).toBeInTheDocument();
    });

    const previewContainer = screen.getByTestId('section-preview').parentElement?.parentElement;
    expect(previewContainer?.className).toContain('lg:sticky');
  });
});
