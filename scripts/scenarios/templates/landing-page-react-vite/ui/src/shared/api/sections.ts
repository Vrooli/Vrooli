import { apiCall } from './common';
import type { ContentSection } from './types';

export function getSections(variantId: number) {
  return apiCall<{ sections: ContentSection[] }>(`/public/variants/${variantId}/sections`);
}

export function getAdminSections(variantId: number) {
  return apiCall<{ sections: ContentSection[] }>(`/variants/${variantId}/sections`);
}

export function getSection(sectionId: number) {
  return apiCall<ContentSection>(`/sections/${sectionId}`);
}

export function updateSection(sectionId: number, content: Record<string, unknown>) {
  return apiCall<{ success: boolean; message: string }>(`/sections/${sectionId}`, {
    method: 'PATCH',
    body: JSON.stringify({ content }),
  });
}

export function patchSection(sectionId: number, payload: Partial<Pick<ContentSection, 'order' | 'enabled' | 'section_type'>>) {
  return apiCall<{ success: boolean }>(`/sections/${sectionId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}

export function createSection(section: Omit<ContentSection, 'id' | 'created_at' | 'updated_at'>) {
  return apiCall<ContentSection>('/sections', {
    method: 'POST',
    body: JSON.stringify(section),
  });
}

export function deleteSection(sectionId: number) {
  return apiCall<{ success: boolean }>(`/sections/${sectionId}`, {
    method: 'DELETE',
  });
}
