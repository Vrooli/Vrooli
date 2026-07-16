import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';
import { screen, waitFor, within } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils';
import { BrowserRouter } from 'react-router-dom';
import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { ContentSectionSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/content_pb';
import { VariantSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/variant_pb';
import {
  LandingConfigResponseSchema,
  LandingVariantSummarySchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/config_pb';
import { SectionEditor } from './SectionEditor';
import * as controller from '../controllers/sectionEditorController';
import type { SectionEditorState, VariantContext } from '../controllers/sectionEditorController';
import * as api from '../../../shared/api';

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

// Mock useParams (mutable so tests can exercise the new-section and no-variant paths)
const paramsHolder = vi.hoisted(() => ({ value: { variantSlug: 'test-variant', sectionId: '1' } as Record<string, string | undefined> }));
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => paramsHolder.value,
  };
});

const heroContent = {
  title: 'Test Title',
  subtitle: 'Test Subtitle',
  cta_text: 'Get Started',
  cta_url: '/signup',
};

const mockSection = create(ContentSectionSchema, {
  id: 1n,
  variantId: 1n,
  sectionType: 'hero',
  enabled: true,
  order: 0,
  content: heroContent,
  createdAt: timestampFromDate(new Date()),
  updatedAt: timestampFromDate(new Date()),
});

const mockControllerState: SectionEditorState = {
  section: mockSection,
  form: {
    sectionType: mockSection.sectionType,
    enabled: mockSection.enabled,
    order: mockSection.order,
    content: heroContent,
  },
};

const mockVariantContext: VariantContext = {
  variant: create(VariantSchema, {
    slug: 'test-variant',
    name: 'Test Variant',
  }),
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

const mockLandingConfig = create(LandingConfigResponseSchema, {
  variant: create(LandingVariantSummarySchema, {
    slug: 'test-variant',
    name: 'Test Variant',
  }),
  sections: [],
  downloads: [],
  fallback: false,
});

describe('SectionEditor [REQ:CUSTOM-SPLIT,CUSTOM-LIVE]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    paramsHolder.value = { variantSlug: 'test-variant', sectionId: '1' };
    vi.mocked(controller.loadSectionEditor).mockResolvedValue(mockControllerState);
    vi.mocked(controller.loadVariantContext).mockResolvedValue(mockVariantContext);
    vi.mocked(api.getLandingConfig).mockResolvedValue(mockLandingConfig);
    vi.mocked(api.listVariants).mockResolvedValue([
      create(VariantSchema, { slug: 'test-variant', name: 'Test Variant', status: 'active' }),
      create(VariantSchema, { slug: 'compare-variant', name: 'Compare Variant', status: 'active' }),
    ]);
  });

  const renderEditor = () => {
    return renderWithProviders(
      <BrowserRouter>
        <SectionEditor />
      </BrowserRouter>,
      { withoutRouter: true }
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
      expect(controller.loadSectionEditor).toHaveBeenCalledWith(1n);
    });

    // Verify form fields are populated (await the render that follows the load).
    const titleInput = (await screen.findByTestId('content-title-input')) as HTMLInputElement;
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
    const disabledSection = create(ContentSectionSchema, { ...mockSection, enabled: false });
    vi.mocked(controller.loadSectionEditor).mockResolvedValue({
      section: disabledSection,
      form: {
        sectionType: disabledSection.sectionType,
        enabled: disabledSection.enabled,
        order: disabledSection.order,
        content: heroContent,
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

  it('edits content fields, toggles enablement, and reorders the section', async () => {
    const user = (await import('@testing-library/user-event')).default.setup();
    renderEditor();
    await screen.findByTestId('content-title-input');

    await user.type(screen.getByTestId('content-title-input'), '!');
    expect((screen.getByTestId('content-title-input') as HTMLInputElement).value).toContain('!');
    await user.type(screen.getByTestId('content-subtitle-input'), ' more');
    await user.type(screen.getByTestId('content-cta-text-input'), ' now');
    await user.type(screen.getByTestId('content-cta-url-input'), '/x');

    const enabled = screen.getByTestId('section-enabled-input');
    await user.click(enabled);
    expect(enabled).not.toBeChecked();

    const order = screen.getByTestId('section-order-input');
    await user.clear(order);
    await user.type(order, '3');
    expect((order as HTMLInputElement).value).toBe('3');
  });

  it('saves the section content through the controller', async () => {
    const user = (await import('@testing-library/user-event')).default.setup();
    vi.mocked(controller.persistExistingSectionContent).mockResolvedValue(mockControllerState);
    renderEditor();
    await screen.findByTestId('save-section');

    await user.type(screen.getByTestId('content-title-input'), ' edited');
    await user.click(screen.getByTestId('save-section'));
    await waitFor(() => expect(controller.persistExistingSectionContent).toHaveBeenCalled());
  });

  it.each(['features', 'pricing', 'cta', 'video', 'testimonials', 'faq', 'footer', 'downloads'])(
    'renders the %s section preview from its loaded type',
    async (sectionType) => {
      vi.mocked(controller.loadSectionEditor).mockResolvedValue({
        section: create(ContentSectionSchema, { id: 1n, variantId: 1n, sectionType, enabled: true, order: 0, content: { title: 'T' } }),
        form: { sectionType, enabled: true, order: 0, content: { title: 'T' } },
      } as never);
      renderEditor();
      expect(await screen.findByTestId('section-preview')).toBeInTheDocument();
      const typeSelect = (await screen.findByTestId('section-type-input')) as HTMLSelectElement;
      expect(typeSelect.value).toBe(sectionType);
    },
  );

  it('renders without variant context when no variant slug is in the route', async () => {
    paramsHolder.value = { sectionId: '1' };
    renderEditor();
    // The section form still loads; variant-scoped panels are skipped.
    expect(await screen.findByTestId('section-form')).toBeInTheDocument();
    expect(screen.queryByTestId('variant-context-card')).not.toBeInTheDocument();
  });

  it('renders the new-section form without fetching an existing section', async () => {
    paramsHolder.value = { variantSlug: 'test-variant', sectionId: 'new' };
    renderEditor();
    expect(await screen.findByTestId('section-form')).toBeInTheDocument();
    expect(controller.loadSectionEditor).not.toHaveBeenCalled();
  });

  it('surfaces a variant-guidance load error', async () => {
    vi.mocked(controller.loadVariantContext).mockRejectedValue(new Error('guidance offline'));
    renderEditor();
    expect(await screen.findByText(/guidance offline/i)).toBeInTheDocument();
  });

  it('surfaces a live-preview load error', async () => {
    vi.mocked(api.getLandingConfig).mockRejectedValue(new Error('preview offline'));
    renderEditor();
    expect((await screen.findAllByText(/preview offline/i)).length).toBeGreaterThan(0);
  });

  it('tolerates a variant-options load failure', async () => {
    vi.mocked(api.listVariants).mockRejectedValue(new Error('variants offline'));
    renderEditor();
    // The editor still renders its form despite the compare-list failure.
    expect(await screen.findByTestId('section-form')).toBeInTheDocument();
  });

  it('surfaces a section load error', async () => {
    vi.mocked(controller.loadSectionEditor).mockRejectedValue(new Error('section offline'));
    renderEditor();
    expect(await screen.findByText(/section offline/i)).toBeInTheDocument();
  });

  it('surfaces an error when saving the section fails', async () => {
    const user = (await import('@testing-library/user-event')).default.setup();
    vi.mocked(controller.persistExistingSectionContent).mockRejectedValue(new Error('save blew up'));
    renderEditor();
    await screen.findByTestId('save-section');
    await user.type(screen.getByTestId('content-title-input'), '!');
    await user.click(screen.getByTestId('save-section'));
    expect(await screen.findByText(/save blew up/i)).toBeInTheDocument();
  });

  it('edits the hero image URL content field', async () => {
    const user = (await import('@testing-library/user-event')).default.setup();
    renderEditor();
    const imageField = await screen.findByTestId('content-image-url-input');
    await user.type(imageField, 'https://cdn/hero.png');
    expect((imageField as HTMLInputElement).value).toContain('https://cdn/hero.png');
  });

  it('loads a comparison variant configuration when selected', async () => {
    const user = (await import('@testing-library/user-event')).default.setup();
    renderEditor();
    await screen.findByTestId('section-form');

    const compareSelect = (await screen.findByRole('option', { name: /Compare Variant/i })).closest(
      'select',
    ) as HTMLSelectElement;
    await user.selectOptions(compareSelect, 'compare-variant');
    await waitFor(() => expect(api.getLandingConfig).toHaveBeenCalledWith('compare-variant'));
  });
});
