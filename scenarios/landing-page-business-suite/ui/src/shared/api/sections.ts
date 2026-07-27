import { apiCall } from './common';
import { parseOrNull } from './safeParse';
import {
  ContentSectionSchema,
  SectionsListResponseSchema,
  UpdateSectionResponseSchema,
} from './schemas/sections.schema';
import { SuccessResponseSchema } from './schemas/common.schema';
import type { ContentSection } from './types';

export function getSections(variantId: number) {
  return apiCall<{ sections: ContentSection[] }>(`/public/variants/${String(variantId)}/sections`).then((resp) => {
    const validated = parseOrNull(SectionsListResponseSchema, resp, 'SectionsListResponse');
    if (!validated) {
      return { sections: [] };
    }
    return validated;
  });
}

export function getAdminSections(variantId: number) {
  return apiCall<{ sections: ContentSection[] }>(`/variants/${String(variantId)}/sections`).then((resp) => {
    const validated = parseOrNull(SectionsListResponseSchema, resp, 'SectionsListResponse');
    if (!validated) {
      return { sections: [] };
    }
    return validated;
  });
}

export function getSection(sectionId: number) {
  return apiCall<ContentSection>(`/sections/${String(sectionId)}`).then((resp) => {
    const validated = parseOrNull(ContentSectionSchema, resp, 'ContentSection');
    if (!validated) {
      throw new Error('Invalid section response from API');
    }
    return validated;
  });
}

export function updateSection(sectionId: number, content: Record<string, unknown>) {
  return apiCall<{ success: boolean; message: string }>(`/sections/${String(sectionId)}`, {
    method: 'PATCH',
    body: JSON.stringify({ content }),
  }).then((resp) => {
    const validated = parseOrNull(UpdateSectionResponseSchema, resp, 'UpdateSectionResponse');
    if (!validated) {
      throw new Error('Invalid update section response from API');
    }
    return validated;
  });
}

export function patchSection(sectionId: number, payload: Partial<Pick<ContentSection, 'order' | 'enabled' | 'section_type'>>) {
  return apiCall<{ success: boolean }>(`/sections/${String(sectionId)}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'PatchSectionResponse');
    if (!validated) {
      throw new Error('Invalid patch section response from API');
    }
    return validated;
  });
}

export function createSection(section: Omit<ContentSection, 'id' | 'created_at' | 'updated_at'>) {
  return apiCall<ContentSection>('/sections', {
    method: 'POST',
    body: JSON.stringify(section),
  }).then((resp) => {
    const validated = parseOrNull(ContentSectionSchema, resp, 'ContentSection');
    if (!validated) {
      throw new Error('Invalid create section response from API');
    }
    return validated;
  });
}

export function deleteSection(sectionId: number) {
  return apiCall<{ success: boolean }>(`/sections/${String(sectionId)}`, {
    method: 'DELETE',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'DeleteSectionResponse');
    if (!validated) {
      throw new Error('Invalid delete section response from API');
    }
    return validated;
  });
}
