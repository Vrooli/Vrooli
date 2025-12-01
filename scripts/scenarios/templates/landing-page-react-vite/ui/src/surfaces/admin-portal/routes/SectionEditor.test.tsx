import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { SectionEditor } from './SectionEditor';
import * as controller from '../controllers/sectionEditorController';

// Mock the controller module
vi.mock('../controllers/sectionEditorController', () => ({
  loadSectionEditor: vi.fn(),
  persistExistingSectionContent: vi.fn(),
  loadVariantContext: vi.fn(),
}));

vi.mock('../../../app/providers/LandingVariantProvider', () => ({
  useLandingVariant: () => ({
    variant: { slug: 'control', name: 'Control' },
    config: { sections: [], downloads: [], fallback: false },
    loading: false,
    error: null,
    resolution: 'api_select',
    statusNote: null,
    lastUpdated: Date.now(),
    refresh: vi.fn(),
  }),
  LandingVariantProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
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

const mockControllerState = {
  section: mockSection,
  form: {
    sectionType: mockSection.section_type,
    enabled: mockSection.enabled,
    order: mockSection.order,
    content: mockSection.content,
  },
};

const mockVariantContext = {
  variant: {
    slug: 'test-variant',
    name: 'Test Variant',
  },
  axes: [
    {
      axisId: 'persona',
      axisLabel: 'Persona',
      axisNote: 'Buyer persona',
      selectionId: 'ops_leader',
      selectionLabel: 'Ops Leader',
      selectionDescription: 'Director of Operations',
      agentHints: ['Emphasize governance'],
    },
  ],
  variantSpace: {
    name: 'Test Space',
    note: 'Context note',
    agentGuidelines: ['Pick one axis variant per persona.'],
    constraintsNote: 'Some combos disabled',
  },
};

describe('SectionEditor [REQ:CUSTOM-SPLIT,CUSTOM-LIVE]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(controller.loadSectionEditor).mockResolvedValue(mockControllerState);
    vi.mocked(controller.loadVariantContext).mockResolvedValue(mockVariantContext);
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
      expect(controller.loadSectionEditor).toHaveBeenCalledWith(1);
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
    vi.mocked(controller.loadSectionEditor).mockResolvedValue({
      section: disabledSection,
      form: {
        sectionType: disabledSection.section_type,
        enabled: disabledSection.enabled,
        order: disabledSection.order,
        content: disabledSection.content,
      },
    });

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
  it('surfaces variant context guidance from variant_space', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('variant-context-card')).toBeInTheDocument();
    });

    expect(controller.loadVariantContext).toHaveBeenCalledWith('test-variant');
    expect(screen.getByText(/Ops Leader/i)).toBeInTheDocument();
    expect(screen.getByText(/Emphasize governance/i)).toBeInTheDocument();
  });

  it('displays styling guardrails pulled from styling.json', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('styling-guardrails-card')).toBeInTheDocument();
    });

    expect(screen.getByText(/Styling & Tone Guardrails/)).toBeInTheDocument();
    expect(screen.getByText(/Primary CTA/i)).toBeInTheDocument();
  });
});
