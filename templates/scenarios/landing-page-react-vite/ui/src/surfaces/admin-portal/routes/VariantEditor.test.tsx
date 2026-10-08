import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor, fireEvent, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../../../test-utils';
import { VariantEditor } from './VariantEditor';

let routeSlug = 'hero';
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async (importActual) => {
  const actual = await importActual<typeof import('react-router-dom')>();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ slug: routeSlug }),
  };
});

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: React.ReactNode }) => <div data-testid="admin-layout">{children}</div>,
}));

// Monaco is unavailable in jsdom; swap it for a controlled textarea seam that
// still exercises the onMount/onValidate wiring the editor sets up.
vi.mock('@monaco-editor/react', () => ({
  __esModule: true,
  default: ({
    value,
    onChange,
    onMount,
    onValidate,
  }: {
    value?: string;
    onChange?: (v: string) => void;
    onMount?: (editor: unknown, monaco: unknown) => void;
    onValidate?: (markers: unknown[]) => void;
  }) => {
    React.useEffect(() => {
      const uri = 'inmemory://model/landing-variant.json';
      const monaco = {
        languages: { json: { jsonDefaults: { setDiagnosticsOptions: () => {} } } },
        editor: {
          getModelMarkers: () => [{ message: 'schema mismatch', startLineNumber: 1, startColumn: 2 }],
          onDidChangeMarkers: (cb: (changed: { resource: { toString(): string } }[]) => void) => {
            cb([{ resource: { toString: () => uri } }]);
            return { dispose: () => {} };
          },
        },
        Uri: { parse: (s: string) => ({ toString: () => s }) },
      };
      onMount?.({}, monaco);
      onValidate?.([{ message: 'bad', startLineNumber: 2, startColumn: 3 }]);
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);
    return <textarea data-testid="monaco-editor" value={value ?? ''} onChange={(e) => onChange?.(e.target.value)} />;
  },
}));

const { mockLoadData, mockLoadSpace, mockLoadSnapshot, mockPersist, mockPersistSnapshot } = vi.hoisted(() => ({
  mockLoadData: vi.fn(),
  mockLoadSpace: vi.fn(),
  mockLoadSnapshot: vi.fn(),
  mockPersist: vi.fn(),
  mockPersistSnapshot: vi.fn(),
}));

// Partial mock: keep the pure helpers (validateVariantForm, buildAxesSelection,
// hydrateFormFromVariant, sanitizeSlugInput) real so their branches are covered.
vi.mock('../controllers/variantEditorController', async (importActual) => {
  const actual = await importActual<typeof import('../controllers/variantEditorController')>();
  return {
    ...actual,
    loadVariantEditorData: mockLoadData,
    loadVariantSpaceDefinition: mockLoadSpace,
    loadVariantSnapshot: mockLoadSnapshot,
    persistVariant: mockPersist,
    persistVariantSnapshot: mockPersistSnapshot,
  };
});

const variantSpace = {
  _name: 'Conversion space',
  _note: 'note',
  axes: {
    tone: { _note: 'Tone', variants: [{ id: 'bold', label: 'Bold' }] },
  },
} as never;

const existingVariant = {
  id: 5n,
  slug: 'hero',
  name: 'Hero Variant',
  description: 'primary',
  weight: 45,
  axes: { tone: 'bold' },
  headerConfig: undefined,
} as never;

const sections = [
  { id: 11n, sectionType: 'hero', enabled: true, order: 0, content: {} },
  { id: 12n, sectionType: 'pricing', enabled: true, order: 1, content: {} },
] as never;

beforeEach(() => {
  vi.clearAllMocks();
  routeSlug = 'hero';
  mockLoadSpace.mockResolvedValue(variantSpace);
  mockLoadData.mockResolvedValue({ variant: existingVariant, sections });
  mockLoadSnapshot.mockResolvedValue({ variant: { slug: 'hero' }, sections: [] });
  mockPersist.mockResolvedValue(undefined);
  mockPersistSnapshot.mockResolvedValue({ variant: { slug: 'hero' }, sections: [] });
});

describe('VariantEditor [REQ:VARIANT-MGMT]', () => {
  it('loads an existing variant and hydrates the form and section list', async () => {
    renderWithProviders(<VariantEditor />);
    await waitFor(() => expect(mockLoadData).toHaveBeenCalledWith('hero'));
    expect(await screen.findByRole('heading', { name: 'Edit Variant' })).toBeInTheDocument();
    expect(screen.getByTestId('variant-name-input')).toHaveValue('Hero Variant');
    expect(screen.getByTestId('variant-description-input')).toHaveValue('primary');
    expect(screen.getByTestId('section-11')).toBeInTheDocument();
    expect(screen.getByTestId('section-12')).toBeInTheDocument();
  });

  it('falls back to an empty configurator name when the variant is unnamed', async () => {
    mockLoadData.mockResolvedValue({
      variant: { ...(existingVariant as Record<string, unknown>), name: '', slug: 'anon' },
      sections,
    } as never);
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    expect(screen.getByTestId('variant-name-input')).toHaveValue('');
  });

  it('shows the empty-state prompt when the variant has no sections', async () => {
    mockLoadData.mockResolvedValue({ variant: existingVariant, sections: [] } as never);
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    expect(screen.getByTestId('add-section')).toBeInTheDocument();
    // The section list is absent when there are none.
    expect(screen.queryByTestId('section-11')).not.toBeInTheDocument();
  });

  it('renders disabled-section styling and update dates in the section list', async () => {
    const { timestampFromDate } = await import('@bufbuild/protobuf/wkt');
    mockLoadData.mockResolvedValue({
      variant: existingVariant,
      sections: [
        { id: 11n, sectionType: 'hero', enabled: false, order: 0, content: {}, updatedAt: timestampFromDate(new Date('2025-01-01')) },
        { id: 12n, sectionType: 'pricing', enabled: true, order: 1, content: {} },
      ],
    } as never);
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('section-11');
    expect(screen.getByTestId('section-12')).toBeInTheDocument();
  });

  it('navigates to the section editor when editing or adding a section', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('section-11');
    await user.click(screen.getByTestId('edit-section-11'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization/variants/hero/sections/11');
    await user.click(screen.getByTestId('add-section'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization/variants/hero/sections/new');
  });

  it('saves an existing variant and refetches', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    await user.click(screen.getByTestId('save-variant'));
    await waitFor(() =>
      expect(mockPersist).toHaveBeenCalledWith(expect.objectContaining({ isNew: false, slugFromRoute: 'hero' })),
    );
    // fetchVariant runs on load and again after save.
    await waitFor(() => expect(mockLoadData).toHaveBeenCalledTimes(2));
  });

  it('surfaces a load error', async () => {
    mockLoadData.mockRejectedValue(new Error('variant gone'));
    renderWithProviders(<VariantEditor />);
    expect(await screen.findByText('variant gone')).toBeInTheDocument();
  });

  it('surfaces a save error from persistVariant', async () => {
    mockPersist.mockRejectedValue(new Error('persist failed'));
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    await user.click(screen.getByTestId('save-variant'));
    expect(await screen.findByText('persist failed')).toBeInTheDocument();
  });

  it('clears a validation error once the form is edited', async () => {
    routeSlug = 'new';
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    await user.click(screen.getByTestId('save-variant'));
    expect(await screen.findByTestId('variant-validation-error')).toBeInTheDocument();
    await user.type(screen.getByTestId('variant-name-input'), 'X');
    expect(screen.queryByTestId('variant-validation-error')).not.toBeInTheDocument();
  });

  it('edits the weight and description fields', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    const weight = screen.getByTestId('variant-weight-input');
    fireEvent.change(weight, { target: { value: '75' } });
    expect((weight as HTMLInputElement).value).toBe('75');
    await user.type(screen.getByTestId('variant-description-input'), ' more');
  });

  it('hides the primary CTA when its mode is set to hidden', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    const primaryCta = screen.getByDisplayValue('Use hero CTA');
    await user.selectOptions(primaryCta, 'hidden');
    expect((primaryCta as HTMLSelectElement).value).toBe('hidden');
    await user.selectOptions(primaryCta, 'downloads');
    expect((primaryCta as HTMLSelectElement).value).toBe('downloads');
  });

  it('requires a slug when creating a new variant with a name', async () => {
    routeSlug = 'new';
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    await user.type(screen.getByTestId('variant-name-input'), 'Named Only');
    await user.click(screen.getByTestId('save-variant'));
    expect(await screen.findByTestId('variant-validation-error')).toHaveTextContent('Slug is required');
  });

  it('renders an existing header configuration with links and custom CTAs', async () => {
    mockLoadData.mockResolvedValue({
      variant: {
        ...(existingVariant as Record<string, unknown>),
        headerConfig: {
          branding: { mode: 'logo_and_name', label: 'Hero', mobilePreference: 'auto' },
          nav: {
            links: [
              { id: 'n1', type: 'downloads', label: 'Get it', anchor: 'downloads', visibleOn: { desktop: true, mobile: false } },
              {
                id: 'n2',
                type: 'menu',
                label: 'Resources',
                visibleOn: { desktop: true, mobile: true },
                children: [{ id: 'n2a', type: 'custom', label: 'Blog', href: '/blog', visibleOn: { desktop: true, mobile: true } }],
              },
            ],
          },
          ctas: {
            primary: { mode: 'custom', label: 'Buy', href: '/buy', variant: 'solid' },
            secondary: { mode: 'custom', label: 'Docs', href: '/docs', variant: 'ghost' },
          },
          behavior: { sticky: true, hideOnScroll: false },
        },
      },
      sections,
    } as never);
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    // The existing menu link renders its dropdown editor.
    expect(screen.getByText('Dropdown menu')).toBeInTheDocument();
    expect(screen.getAllByText('Downloads anchor').length).toBeGreaterThan(0);
  });

  it('validates required fields before creating a new variant', async () => {
    routeSlug = 'new';
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    expect(await screen.findByRole('heading', { name: 'New Variant' })).toBeInTheDocument();
    // Empty name -> validation error, no persist.
    await user.click(screen.getByTestId('save-variant'));
    expect(await screen.findByTestId('variant-validation-error')).toHaveTextContent('Name is required');
    expect(mockPersist).not.toHaveBeenCalled();
  });

  it('creates a new variant and navigates to its edit route', async () => {
    routeSlug = 'new';
    mockPersist.mockResolvedValue({ slug: 'brand-new', name: 'Brand New' });
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    // Wait for the axes to seed so the form validates.
    await waitFor(() => expect(mockLoadSpace).toHaveBeenCalled());

    await user.type(screen.getByTestId('variant-name-input'), 'Brand New');
    await user.type(screen.getByTestId('variant-slug-input'), 'brandnew');
    await user.click(screen.getByTestId('save-variant'));

    await waitFor(() => expect(mockPersist).toHaveBeenCalledWith(expect.objectContaining({ isNew: true })));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization/variants/brand-new');
  });

  it('disables the JSON editor tab for new variants', async () => {
    routeSlug = 'new';
    renderWithProviders(<VariantEditor />);
    const jsonTab = await screen.findByRole('button', { name: 'JSON Editor' });
    expect(jsonTab).toBeDisabled();
  });

  it('edits and saves the variant JSON snapshot', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');

    await user.click(screen.getByRole('button', { name: 'JSON Editor' }));
    const editor = await screen.findByTestId('monaco-editor');
    fireEvent.change(editor, {
      target: { value: JSON.stringify({ variant: { slug: 'hero' }, sections: [] }) },
    });
    await user.click(screen.getByTestId('save-variant'));

    await waitFor(() => expect(mockPersistSnapshot).toHaveBeenCalledWith('hero', expect.any(Object)));
  });

  it('adds, edits, and removes navigation links in the header configurator', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');

    // Default config has no manual links.
    expect(screen.getByText(/No manual links added/)).toBeInTheDocument();

    // Add a dropdown menu (seeds two child items).
    await user.click(screen.getByRole('button', { name: 'Add menu' }));
    expect(screen.getByText('Dropdown menu')).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'Remove' })).toHaveLength(2);

    // Add and then remove a menu item.
    await user.click(screen.getByRole('button', { name: 'Add item' }));
    expect(screen.getAllByRole('button', { name: 'Remove' })).toHaveLength(3);
    await user.click(screen.getAllByRole('button', { name: 'Remove' })[0]!);
    expect(screen.getAllByRole('button', { name: 'Remove' })).toHaveLength(2);

    // Toggle desktop visibility off.
    const desktopToggle = screen.getByRole('checkbox', { name: /Desktop/i });
    await user.click(desktopToggle);
    expect(desktopToggle).not.toBeChecked();

    // Removing the link returns to the empty state.
    await user.click(screen.getByRole('button', { name: '×' }));
    expect(screen.getByText(/No manual links added/)).toBeInTheDocument();
  });

  it('adds a downloads anchor nav link and edits menu children', async () => {
    // Include a downloads section so the "Downloads anchor" option is enabled.
    mockLoadData.mockResolvedValue({
      variant: existingVariant,
      sections: [...(sections as never[]), { id: 20n, sectionType: 'downloads', enabled: true, order: 2, content: {} }],
    } as never);
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');

    const navSelect = screen.getByRole('option', { name: 'Select target' }).closest('select') as HTMLSelectElement;
    const dlOption = within(navSelect).getByRole('option', { name: /Downloads anchor/ }) as HTMLOptionElement;
    await user.selectOptions(navSelect, dlOption.value);
    await user.click(screen.getByRole('button', { name: 'Add link' }));

    // Add a menu, then add and edit a child item.
    await user.click(screen.getByRole('button', { name: 'Add menu' }));
    await user.click(screen.getByRole('button', { name: 'Add item' }));
    const childLabel = screen.getAllByDisplayValue('First link')[0]!;
    await user.type(childLabel, ' x');
    expect(screen.queryByText(/No manual links added/)).not.toBeInTheDocument();
  });

  it('adds a navigation link that targets a content section', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');

    const navSelect = screen.getByRole('option', { name: /Section · hero/ }).closest('select') as HTMLSelectElement;
    const heroOption = screen.getByRole('option', { name: /Section · hero/ }) as HTMLOptionElement;
    await user.selectOptions(navSelect, heroOption.value);
    await user.click(screen.getByRole('button', { name: 'Add link' }));

    expect(screen.queryByText(/No manual links added/)).not.toBeInTheDocument();
  });

  it('switches branding modes and configures a custom primary CTA', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');

    // Branding display mode select.
    const brandingSelect = screen.getByDisplayValue('Logo + Name');
    await user.selectOptions(brandingSelect, 'logo');
    expect((brandingSelect as HTMLSelectElement).value).toBe('logo');

    // Primary CTA -> custom reveals label/href inputs.
    const primaryCta = screen.getByDisplayValue('Use hero CTA');
    await user.selectOptions(primaryCta, 'custom');
    // The primary CTA's custom label input is the first "Button label" field in
    // the DOM (the secondary CTA's downloads mode also renders one).
    const labelInput = screen.getAllByPlaceholderText('Button label')[0]!;
    await user.type(labelInput, 'Buy now');
    expect(labelInput).toHaveValue('Buy now');

    // Mobile emphasis select.
    const mobileSelect = screen.getByDisplayValue('Show both');
    await user.selectOptions(mobileSelect, 'stacked');
    expect((mobileSelect as HTMLSelectElement).value).toBe('stacked');

    // Secondary CTA -> custom reveals its own label + href inputs.
    const secondaryCta = screen.getByDisplayValue('Downloads anchor');
    await user.selectOptions(secondaryCta, 'custom');
    const labels = screen.getAllByPlaceholderText('Button label');
    await user.type(labels[labels.length - 1]!, 'Learn more');
    const hrefs = screen.getAllByPlaceholderText('https://example.com');
    await user.type(hrefs[hrefs.length - 1]!, 'https://learn.example.com');
    expect((secondaryCta as HTMLSelectElement).value).toBe('custom');
  });

  it('edits dropdown menu items and reorders links in the configurator', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');

    // Two links: a section link and a menu, so the move buttons are enabled.
    const navSelect = screen.getByRole('option', { name: /Section · hero/ }).closest('select') as HTMLSelectElement;
    const heroOption = screen.getByRole('option', { name: /Section · hero/ }) as HTMLOptionElement;
    await user.selectOptions(navSelect, heroOption.value);
    await user.click(screen.getByRole('button', { name: 'Add link' }));
    await user.click(screen.getByRole('button', { name: 'Add menu' }));

    // Edit a menu child's label and URL.
    const itemLabels = screen.getAllByText('Item label');
    expect(itemLabels.length).toBeGreaterThan(0);
    const childLabelInput = screen.getAllByDisplayValue('First link')[0]!;
    await user.type(childLabelInput, ' edited');

    // Move the first link down.
    const downButtons = screen.getAllByRole('button', { name: '↓' });
    await user.click(downButtons[0]!);
    expect(screen.getAllByRole('button', { name: '×' }).length).toBe(2);
  });

  it('surfaces schema issues from the editor and exercises the copy actions', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');

    await user.click(screen.getByRole('button', { name: 'JSON Editor' }));
    // onMount/onValidate populated the schema-issue panel.
    const issues = await screen.findByTestId('variant-json-schema-issues');
    expect(issues).toBeInTheDocument();

    // Copy handlers run against userEvent's clipboard stub without throwing.
    await user.click(screen.getByRole('button', { name: /Copy variant schema/i }));
    await user.click(within(issues).getByRole('button', { name: /copy/i }));
    expect(screen.getByTestId('variant-json-editor')).toBeInTheDocument();
  });

  it('surfaces a snapshot load error in the JSON editor', async () => {
    mockLoadSnapshot.mockRejectedValue(new Error('snapshot gone'));
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    await user.click(screen.getByRole('button', { name: 'JSON Editor' }));
    expect(await screen.findByText(/snapshot gone/i)).toBeInTheDocument();
  });

  it('surfaces a snapshot save error from the API', async () => {
    mockPersistSnapshot.mockRejectedValue(new Error('snapshot save failed'));
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    await user.click(screen.getByRole('button', { name: 'JSON Editor' }));
    const editor = await screen.findByTestId('monaco-editor');
    fireEvent.change(editor, { target: { value: JSON.stringify({ variant: { slug: 'hero' }, sections: [] }) } });
    await user.click(screen.getByTestId('save-variant'));
    expect(await screen.findByText(/snapshot save failed/i)).toBeInTheDocument();
  });

  it('toggles mobile visibility and reorders links up in the configurator', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    // Two links so both move directions are enabled.
    await user.click(screen.getByRole('button', { name: 'Add menu' }));
    await user.click(screen.getByRole('button', { name: 'Add menu' }));

    const mobileToggles = screen.getAllByRole('checkbox', { name: /Mobile/i });
    await user.click(mobileToggles[0]!);
    expect(mobileToggles[0]!).not.toBeChecked();

    // Move the second link up.
    const upButtons = screen.getAllByRole('button', { name: '↑' });
    await user.click(upButtons[1]!);
    expect(screen.getAllByRole('button', { name: '×' }).length).toBe(2);
  });

  it('renders configurator branches for varied existing links and custom CTAs', async () => {
    mockLoadData.mockResolvedValue({
      variant: {
        ...(existingVariant as Record<string, unknown>),
        headerConfig: {
          branding: { mode: 'name', label: 'Hero', mobilePreference: 'name' },
          nav: {
            links: [
              // visibleOn omitted -> desktop/mobile fall back to true
              { id: 'a', type: 'custom', label: 'Docs', href: '/docs' },
              // a plain link with no sectionType -> "custom target" description
              { id: 'b', type: 'link', label: 'Plain' },
              // a menu with no children -> "No items yet"
              { id: 'c', type: 'menu', label: 'EmptyMenu', children: [] },
              // downloads link -> "Downloads anchor" description
              { id: 'd', type: 'downloads', label: 'Get', anchor: 'downloads', visibleOn: { desktop: true, mobile: true } },
            ],
          },
          ctas: {
            primary: { mode: 'custom', label: 'Buy', href: '/buy', variant: 'solid' },
            secondary: { mode: 'custom', label: 'Docs', href: '/docs', variant: 'ghost' },
          },
          behavior: { sticky: true, hideOnScroll: false },
        },
      },
      sections,
    } as never);
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');
    expect(screen.getByText('No items yet.')).toBeInTheDocument();
    // Both custom CTA label inputs render.
    expect(screen.getAllByPlaceholderText('Button label').length).toBeGreaterThanOrEqual(2);
  });

  it('reports invalid JSON without calling the API', async () => {
    const user = userEvent.setup();
    renderWithProviders(<VariantEditor />);
    await screen.findByTestId('variant-name-input');

    await user.click(screen.getByRole('button', { name: 'JSON Editor' }));
    const editor = await screen.findByTestId('monaco-editor');
    fireEvent.change(editor, { target: { value: 'not json' } });
    await user.click(screen.getByTestId('save-variant'));

    expect(await screen.findByText(/Invalid JSON/)).toBeInTheDocument();
    expect(mockPersistSnapshot).not.toHaveBeenCalled();
  });
});
