import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { VariantEditor } from './VariantEditor';
import * as variantHook from '../hooks/useVariantForm';

const navigate = vi.fn();
const useParams = vi.fn<() => { slug?: string }>();
vi.mock('react-router-dom', () => ({ useNavigate: () => navigate, useParams: () => useParams() }));
vi.mock('../hooks/useVariantForm');
vi.mock('@monaco-editor/react', () => ({ default: ({ value, onChange }: { value: string; onChange: (value: string) => void }) => <textarea aria-label="Variant JSON document" value={value} onChange={(event) => { onChange(event.target.value); }} /> }));
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));
vi.mock('../components/HeaderConfigurator', () => ({ HeaderConfigurator: () => <div>Header configurator</div> }));
vi.mock('../components/RuntimeSignalStrip', () => ({ RuntimeSignalStrip: () => <div>Runtime signal</div> }));
const toast = { success: vi.fn(), error: vi.fn() };
vi.mock('../../../shared/ui/useToast', () => ({ useToast: () => toast }));

function formState(overrides: Record<string, unknown> = {}) {
  return { variant: null, sections: [], loading: false, error: null, validationError: null, variantSpace: null, axesSelection: {}, updateAxesSelection: vi.fn(), form: { name: '', slug: '', description: '', weight: 50 }, updateFormField: vi.fn(), headerConfig: {}, setHeaderConfig: vi.fn(), setActiveTab: vi.fn(), isJsonTab: false, currentSaving: false, savingLabel: 'Save Variant', snapshotDraft: '{"variant":{},"sections":[]}', setSnapshotDraft: vi.fn(), snapshotError: null, snapshotLoading: false, schemaIssues: [], copyStatus: null, handleSave: vi.fn().mockResolvedValue({ success: true }), handleSaveJson: vi.fn(), handleEditorMount: vi.fn(), handleCopyIssues: vi.fn(), handleCopySchema: vi.fn(), ...overrides } as unknown as ReturnType<typeof variantHook.useVariantForm>;
}

describe('VariantEditor', () => {
  beforeEach(() => { vi.clearAllMocks(); useParams.mockReturnValue({ slug: 'new' }); });

  it('sanitizes new variant form input and navigates only after a successful creation', async () => {
    const state = formState({ handleSave: vi.fn().mockResolvedValue({ success: true, savedVariant: { slug: 'enterprise-b' } }) });
    vi.mocked(variantHook.useVariantForm).mockReturnValue(state);
    render(<VariantEditor />);
    expect(screen.getByText('New Variant')).toBeInTheDocument();
    fireEvent.change(screen.getByTestId('variant-name-input'), { target: { value: 'Enterprise B' } });
    fireEvent.change(screen.getByTestId('variant-slug-input'), { target: { value: ' Enterprise B! ' } });
    fireEvent.change(screen.getByTestId('variant-weight-input'), { target: { value: '70' } });
    fireEvent.click(screen.getByTestId('save-variant'));
    await waitFor(() => { expect(state.handleSave).toHaveBeenCalledOnce(); });
    expect(state.updateFormField).toHaveBeenCalledWith('name', 'Enterprise B');
    expect(state.updateFormField).toHaveBeenCalledWith('slug', 'enterpriseb');
    expect(state.updateFormField).toHaveBeenCalledWith('weight', 70);
    expect(navigate).toHaveBeenCalledWith('/admin/customization/variants/enterprise-b');
    expect(screen.getByRole('button', { name: 'JSON Editor' })).toBeDisabled();
  });

  it('renders existing sections in order and routes section edit/create controls', () => {
    useParams.mockReturnValue({ slug: 'control' });
    const state = formState({ variant: { name: 'Control' }, form: { name: 'Control', slug: 'control', description: '', weight: 50 }, sections: [{ id: 2, order: 2, enabled: false, section_type: 'faq', updated_at: '2026-01-01T00:00:00Z' }, { id: 1, order: 1, enabled: true, section_type: 'hero', updated_at: '2026-01-01T00:00:00Z' }] });
    vi.mocked(variantHook.useVariantForm).mockReturnValue(state);
    render(<VariantEditor />);
    expect(screen.getByText('Edit Variant')).toBeInTheDocument();
    expect(screen.getByTestId('section-1')).toHaveTextContent('#1');
    expect(screen.getByTestId('section-2')).toHaveTextContent('Disabled');
    fireEvent.click(screen.getByTestId('add-section'));
    fireEvent.click(screen.getByTestId('edit-section-2'));
    expect(navigate).toHaveBeenCalledWith('/admin/customization/variants/control/sections/new');
    expect(navigate).toHaveBeenCalledWith('/admin/customization/variants/control/sections/2');
  });

  it('supports form-field, axis, tab, and back-navigation controls for an existing variant', () => {
    useParams.mockReturnValue({ slug: 'control' });
    const state = formState({
      variant: { name: 'Control' },
      error: 'The variant could not be refreshed',
      validationError: 'A name is required before saving',
      form: { name: 'Control', slug: 'control', description: '', weight: 50 },
      variantSpace: {
        axes: {
          persona: {
            variants: [
              { id: 'startup', label: 'Startup', description: 'Small teams' },
              { id: 'enterprise', label: 'Enterprise', description: 'Large organizations' },
            ],
          },
        },
      },
      axesSelection: { persona: 'startup' },
    });
    vi.mocked(variantHook.useVariantForm).mockReturnValue(state);
    render(<VariantEditor />);

    fireEvent.change(screen.getByTestId('variant-description-input'), { target: { value: 'The baseline experience' } });
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'enterprise' } });
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByRole('button', { name: 'JSON Editor' }));
    fireEvent.click(screen.getByRole('button', { name: 'Form Editor' }));

    expect(state.updateFormField).toHaveBeenCalledWith('description', 'The baseline experience');
    expect(state.updateAxesSelection).toHaveBeenCalledWith('persona', 'enterprise');
    expect(state.setActiveTab).toHaveBeenCalledWith('json');
    expect(state.setActiveTab).toHaveBeenCalledWith('form');
    expect(navigate).toHaveBeenCalledWith('/admin/customization');
    expect(screen.getByText('The variant could not be refreshed')).toBeInTheDocument();
    expect(screen.getByTestId('variant-validation-error')).toHaveTextContent('A name is required before saving');
  });

  it('renders the loading state without mounting editor controls', () => {
    vi.mocked(variantHook.useVariantForm).mockReturnValue(formState({ loading: true }));
    render(<VariantEditor />);
    expect(screen.getByText('Loading variant...')).toBeInTheDocument();
    expect(screen.queryByTestId('save-variant')).not.toBeInTheDocument();
  });

  it('delegates raw JSON editing, schema copy, issue copy, and JSON save to the form seam', async () => {
    useParams.mockReturnValue({ slug: 'control' });
    const state = formState({ isJsonTab: true, snapshotDraft: '{"variant":{}}', schemaIssues: ['variant.sections is required'], copyStatus: 'Copied', snapshotError: 'Invalid JSON' });
    vi.mocked(variantHook.useVariantForm).mockReturnValue(state);
    render(<VariantEditor />);
    fireEvent.change(screen.getByRole('textbox', { name: 'Variant JSON document' }), { target: { value: '{"variant":{"slug":"control"}}' } });
    fireEvent.click(screen.getByRole('button', { name: 'Copy variant schema' }));
    fireEvent.click(screen.getByRole('button', { name: 'Copy issues' }));
    fireEvent.click(screen.getByTestId('save-variant'));
    await waitFor(() => { expect(state.handleSaveJson).toHaveBeenCalledOnce(); });
    expect(state.setSnapshotDraft).toHaveBeenCalledWith('{"variant":{"slug":"control"}}');
    expect(state.handleCopySchema).toHaveBeenCalledOnce();
    expect(state.handleCopyIssues).toHaveBeenCalledOnce();
    expect(screen.getByText('Invalid JSON')).toBeInTheDocument();
  });
});
