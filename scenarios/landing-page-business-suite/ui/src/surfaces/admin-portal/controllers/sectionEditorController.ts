import {
  getVariantSection,
  updateVariantSection,
  createVariantSection,
  getVariant,
  getVariantSpace,
  type ContentSection,
  type Variant,
  type VariantAxes,
  type VariantSpace,
} from '../../../shared/api';

export interface SectionEditorState {
  section: ContentSection;
  form: SectionFormFields;
}

export interface SectionFormFields {
  sectionType: ContentSection['section_type'];
  enabled: boolean;
  order: number;
  content: Record<string, unknown>;
}

function normalizeSectionContent(content: unknown): Record<string, unknown> {
  return typeof content === 'object' && content !== null && !Array.isArray(content)
    ? content as Record<string, unknown>
    : {};
}

export function buildFormFields(section: ContentSection): SectionFormFields {
  return {
    sectionType: section.section_type,
    enabled: section.enabled,
    order: section.order,
    content: normalizeSectionContent(section.content),
  };
}

export async function loadSectionEditor(variantSlug: string, sectionKey: string): Promise<SectionEditorState> {
  const section = await getVariantSection(variantSlug, sectionKey);
  return {
    section,
    form: buildFormFields(section),
  };
}

export async function persistExistingSectionContent(
  variantSlug: string,
  sectionKey: string,
  content: Record<string, unknown>,
): Promise<SectionEditorState> {
  await updateVariantSection(variantSlug, sectionKey, { content });
  return loadSectionEditor(variantSlug, sectionKey);
}

export async function createSection(
  variantSlug: string,
  section: Pick<ContentSection, 'section_type' | 'content' | 'order' | 'enabled'>,
) {
  if (!variantSlug) {
    throw new Error('Variant slug is required');
  }
  return createVariantSection(variantSlug, section);
}

export async function updateSectionOrder(variantSlug: string, sectionKey: string, order: number) {
  if (!variantSlug || !sectionKey || Number.isNaN(order)) {
    throw new Error('Variant slug, section key, and order are required');
  }
  await updateVariantSection(variantSlug, sectionKey, { order });
}

export interface VariantAxisContext {
  axisId: string;
  axisLabel: string;
  axisNote?: string;
  selectionId?: string;
  selectionLabel?: string;
  selectionDescription?: string;
  agentHints?: string[];
  examples?: Record<string, string>;
}

export interface VariantSpaceSummary {
  name: string;
  note?: string;
  agentGuidelines?: string[];
  constraintsNote?: string;
}

export interface VariantContext {
  variant: Variant;
  axes: VariantAxisContext[];
  variantSpace: VariantSpaceSummary;
}

function formatAxisLabel(axisId: string) {
  if (axisId.toLowerCase() === 'jtbd') {
    return 'JTBD';
  }
  const withSpaces = axisId.replace(/([A-Z])/g, ' $1').replace(/_/g, ' ');
  return withSpaces
    .split(' ')
    .map((word) => {
      const firstChar = word[0];
      return word && firstChar ? firstChar.toUpperCase() + word.slice(1) : '';
    })
    .join(' ')
    .trim();
}

function buildAxisContext(space: VariantSpace, axes?: VariantAxes): VariantAxisContext[] {
  return Object.entries(space.axes).map(([axisId, axisDef]) => {
    const selectionId = axes?.[axisId];
    const selection = axisDef.variants.find((variant) => variant.id === selectionId);
    return {
      axisId,
      axisLabel: formatAxisLabel(axisId),
      axisNote: axisDef._note,
      selectionId,
      selectionLabel: selection?.label ?? selectionId,
      selectionDescription: selection?.description,
      agentHints: selection?.agentHints,
      examples: selection?.examples,
    };
  });
}

export async function loadVariantContext(slug: string): Promise<VariantContext> {
  if (!slug) {
    throw new Error('Variant slug is required');
  }

  const [variant, space] = await Promise.all([getVariant(slug), getVariantSpace()]);

  return {
    variant,
    axes: buildAxisContext(space, variant.axes),
    variantSpace: {
      name: space._name,
      note: space._note,
      agentGuidelines: space._agentGuidelines,
      constraintsNote: space.constraints?._note,
    },
  };
}
