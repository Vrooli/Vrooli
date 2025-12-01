import {
  createVariant,
  getAdminSections,
  getVariant,
  getVariantSpace,
  updateVariant,
  type ContentSection,
  type Variant,
  type VariantAxes,
  type VariantSpace,
} from '../../../shared/api';

export interface VariantEditorData {
  variant: Variant;
  sections: ContentSection[];
}

export interface VariantFormState {
  name: string;
  slug: string;
  description: string;
  weight: number;
}

export async function loadVariantEditorData(slug: string): Promise<VariantEditorData> {
  const variant = await getVariant(slug);
  if (!variant.id) {
    throw new Error('Variant payload missing ID');
  }

  const { sections } = await getAdminSections(variant.id);
  return { variant, sections };
}

export function loadVariantSpaceDefinition() {
  return getVariantSpace();
}

export function buildAxesSelection(space: VariantSpace, existing?: VariantAxes): VariantAxes {
  const selection: VariantAxes = {};
  Object.entries(space.axes).forEach(([axisId, axisDef]) => {
    const fallbackValue = axisDef.variants[0]?.id ?? '';
    const candidate = existing?.[axisId] ?? fallbackValue;
    if (candidate) {
      selection[axisId] = candidate;
    }
  });
  return selection;
}

export function hydrateFormFromVariant(variant: Variant): VariantFormState {
  return {
    name: variant.name ?? '',
    slug: variant.slug ?? '',
    description: variant.description ?? '',
    weight: variant.weight ?? 50,
  };
}

export function normalizeForm(form: VariantFormState) {
  return {
    name: form.name.trim(),
    slug: sanitizeSlugInput(form.slug.trim()),
    description: form.description.trim(),
    weight: form.weight,
  };
}

export function sanitizeSlugInput(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9-]/g, '');
}

export function validateVariantForm(params: {
  form: VariantFormState;
  variantSpace?: VariantSpace | null;
  axesSelection: VariantAxes;
  requireSlug: boolean;
}): string | null {
  const normalized = normalizeForm(params.form);
  if (!normalized.name) {
    return 'Name is required';
  }
  if (params.requireSlug && !normalized.slug) {
    return 'Slug is required';
  }
  if (!params.variantSpace) {
    return 'Variant axes registry not loaded yet';
  }

  const missingAxis = Object.keys(params.variantSpace.axes).find(
    (axisId) => !params.axesSelection[axisId],
  );
  if (missingAxis) {
    return `Select a value for the ${missingAxis} axis`;
  }

  return null;
}

export async function persistVariant(params: {
  isNew: boolean;
  slugFromRoute?: string;
  form: VariantFormState;
  axesSelection: VariantAxes;
}) {
  const normalized = normalizeForm(params.form);
  if (params.isNew) {
    return createVariant({
      name: normalized.name,
      slug: normalized.slug,
      description: normalized.description || undefined,
      weight: normalized.weight,
      axes: params.axesSelection,
    });
  }

  if (!params.slugFromRoute) {
    throw new Error('Variant slug missing');
  }

  await updateVariant(params.slugFromRoute, {
    name: normalized.name,
    description: normalized.description || undefined,
    weight: normalized.weight,
    axes: params.axesSelection,
  });

  return undefined;
}
