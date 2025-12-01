import { getSection, updateSection, type ContentSection } from '../../../shared/api';

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

export function buildFormFields(section: ContentSection): SectionFormFields {
  return {
    sectionType: section.section_type,
    enabled: section.enabled,
    order: section.order,
    content: section.content ?? {},
  };
}

export async function loadSectionEditor(sectionId: number): Promise<SectionEditorState> {
  const section = await getSection(sectionId);
  return {
    section,
    form: buildFormFields(section),
  };
}

export async function persistExistingSectionContent(
  sectionId: number,
  content: Record<string, unknown>,
): Promise<SectionEditorState> {
  await updateSection(sectionId, content);
  return loadSectionEditor(sectionId);
}
