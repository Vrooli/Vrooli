import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import type { ReactNode } from 'react';
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { BrowserRouter } from 'react-router-dom';
import { SectionEditor } from './SectionEditor';
import * as controller from '../controllers/sectionEditorController';
import * as api from '../../../shared/api';
import type { LandingConfigResponse } from '../../../shared/api';
import { ToastProvider } from '../../../shared/ui/Toast';

let routeParams = { variantSlug: 'test-variant', sectionId: 'section-1-hero' };

// Mock the controller module
vi.mock('../controllers/sectionEditorController', () => ({
  loadSectionEditor: vi.fn(),
  persistExistingSectionContent: vi.fn(),
  loadVariantContext: vi.fn(),
}));

vi.mock('../../../shared/api', () => ({
  getLandingConfig: vi.fn(),
  listVariants: vi.fn(),
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

// Also mock the separate useLandingVariant hook file (used by RuntimeSignalStrip)
vi.mock('../../../app/providers/useLandingVariant', () => ({
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
}));

// Mock RuntimeSignalStrip to avoid context issues
vi.mock('../components/RuntimeSignalStrip', () => ({
  RuntimeSignalStrip: () => <div data-testid="runtime-signal-strip-mock">Runtime Signal Strip</div>,
}));

// Mock useParams
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => routeParams,
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

const mockLandingConfig: LandingConfigResponse = {
  variant: {
    slug: 'test-variant',
    name: 'Test Variant',
  },
  sections: [],
  downloads: [],
  header: {
    branding: { mode: 'logo_and_name', label: 'Test Variant', mobile_preference: 'auto' },
    nav: { links: [] },
    ctas: {
      primary: { mode: 'inherit_hero', variant: 'solid' },
      secondary: { mode: 'downloads', variant: 'ghost' },
    },
    behavior: { sticky: true, hide_on_scroll: false },
  },
  fallback: false,
};

describe('SectionEditor [REQ:CUSTOM-SPLIT,CUSTOM-LIVE]', () => {
  beforeEach(() => {
    routeParams = { variantSlug: 'test-variant', sectionId: 'section-1-hero' };
    vi.clearAllMocks();
    vi.mocked(controller.loadSectionEditor).mockResolvedValue(mockControllerState);
    vi.mocked(controller.persistExistingSectionContent).mockResolvedValue(mockControllerState);
    vi.mocked(controller.loadVariantContext).mockResolvedValue(mockVariantContext);
    vi.mocked(api.getLandingConfig).mockResolvedValue(mockLandingConfig);
    vi.mocked(api.listVariants).mockResolvedValue({
      variants: [
        { slug: 'test-variant', name: 'Test Variant', status: 'active' },
        { slug: 'compare-variant', name: 'Compare Variant', status: 'active' },
      ],
    });
  });

  const renderEditor = () => {
    return render(
      <BrowserRouter>
        <ToastProvider>
          <SectionEditor />
        </ToastProvider>
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
      expect(controller.loadSectionEditor).toHaveBeenCalledWith('test-variant', 'section-1-hero');
    });

    // Verify form fields are populated
    const titleInput = await screen.findByTestId('content-title-input');
    if (!(titleInput instanceof HTMLInputElement)) {
      throw new Error('expected content title input to be a native input');
    }
    expect(titleInput.value).toBe('Test Title');

    const subtitleInput = await screen.findByTestId('content-subtitle-input');
    if (!(subtitleInput instanceof HTMLTextAreaElement)) {
      throw new Error('expected content subtitle input to be a native textarea');
    }
    expect(subtitleInput.value).toBe('Test Subtitle');
  });

  it('should display section type selector', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('section-type-input')).toBeInTheDocument();
    });

    const typeSelect = screen.getByTestId('section-type-input');
    if (!(typeSelect instanceof HTMLSelectElement)) {
      throw new Error('expected section type input to be a native select');
    }
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
    const contextCard = screen.getByTestId('variant-context-card');
    expect(within(contextCard).getByText(/Ops Leader/i)).toBeInTheDocument();
    expect(within(contextCard).getByText(/Emphasize governance/i)).toBeInTheDocument();
  });

  it('displays styling guardrails pulled from styling.json', async () => {
    renderEditor();

    await waitFor(() => {
      expect(screen.getByTestId('styling-guardrails-card')).toBeInTheDocument();
    });

    expect(screen.getByText(/Styling & Tone Guardrails/)).toBeInTheDocument();
    expect(screen.getByText(/Primary CTA/i)).toBeInTheDocument();
  });

  it('updates section fields, toggles enabled state, and persists the edited content', async () => {
    renderEditor();
    const title = await screen.findByTestId('content-title-input');
    fireEvent.change(title, { target: { value: 'Updated hero' } });
    fireEvent.change(screen.getByTestId('content-subtitle-input'), { target: { value: 'Updated subtitle' } });
    fireEvent.change(screen.getByTestId('content-cta-text-input'), { target: { value: 'Start now' } });
    fireEvent.change(screen.getByTestId('content-cta-url-input'), { target: { value: '/start' } });
    fireEvent.click(screen.getByTestId('section-enabled-input'));
    fireEvent.change(screen.getByTestId('section-order-input'), { target: { value: '4' } });
    fireEvent.click(screen.getByTestId('save-section'));
    await waitFor(() => {
      expect(controller.persistExistingSectionContent).toHaveBeenCalledWith(
        'test-variant',
        'section-1-hero',
        expect.objectContaining({ title: 'Updated hero', subtitle: 'Updated subtitle', cta_text: 'Start now', cta_url: '/start' }),
      );
    });
  });

  it('allows a new section to select each supported renderer for live preview', async () => {
    routeParams = { variantSlug: 'test-variant', sectionId: 'new' };
    renderEditor();
    const type = await screen.findByTestId('section-type-input');
    for (const sectionType of ['features', 'pricing', 'cta', 'testimonials', 'faq', 'footer', 'video', 'downloads'] as const) {
      fireEvent.change(type, { target: { value: sectionType } });
      expect(type).toHaveValue(sectionType);
      expect(screen.getByTestId('section-preview')).toBeInTheDocument();
    }
  });
});
