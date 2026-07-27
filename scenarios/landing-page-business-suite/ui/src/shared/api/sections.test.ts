import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createSection, deleteSection, getAdminSections, getSection, getSections, patchSection, updateSection } from './sections';
import { apiCall } from './common';

vi.mock('./common', () => ({ apiCall: vi.fn() }));
const mockApiCall = vi.mocked(apiCall);

const section = {
  id: 3, variant_id: 2, section_type: 'hero' as const, content: { title: 'Welcome' },
  order: 1, enabled: true, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
};

describe('sections API', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('loads public and admin sections, falling back safely for malformed lists', async () => {
    mockApiCall.mockResolvedValueOnce({ sections: [section] }).mockResolvedValueOnce({ sections: [section] }).mockResolvedValueOnce({});
    await expect(getSections(2)).resolves.toEqual({ sections: [section] });
    await expect(getAdminSections(2)).resolves.toEqual({ sections: [section] });
    await expect(getSections(2)).resolves.toEqual({ sections: [] });
    expect(mockApiCall.mock.calls.map(([path]) => path)).toEqual(['/public/variants/2/sections', '/variants/2/sections', '/public/variants/2/sections']);
  });

  it('gets and creates validated sections while rejecting malformed payloads', async () => {
    mockApiCall.mockResolvedValueOnce(section).mockResolvedValueOnce({}).mockResolvedValueOnce(section);
    await expect(getSection(3)).resolves.toEqual(section);
    await expect(getSection(3)).rejects.toThrow('Invalid section response');
    await expect(createSection({ ...section, id: undefined, created_at: undefined, updated_at: undefined } as never)).resolves.toEqual(section);
    expect(mockApiCall.mock.calls[2]?.[0]).toBe('/sections');
  });

  it('sends update, patch, and delete requests with their validated responses', async () => {
    mockApiCall.mockResolvedValueOnce({ success: true, message: 'Saved' }).mockResolvedValueOnce({ success: true }).mockResolvedValueOnce({ success: true });
    await expect(updateSection(3, { title: 'Updated' })).resolves.toEqual({ success: true, message: 'Saved' });
    await expect(patchSection(3, { enabled: false })).resolves.toEqual({ success: true });
    await expect(deleteSection(3)).resolves.toEqual({ success: true });
    expect(mockApiCall.mock.calls.map(([, options]) => options?.method)).toEqual(['PATCH', 'PATCH', 'DELETE']);
  });

  it('fails closed when a mutation response does not satisfy its declared contract', async () => {
    mockApiCall.mockResolvedValueOnce({}).mockResolvedValueOnce({ updated_at: 42 }).mockResolvedValueOnce({ success: 'yes' });
    await expect(updateSection(3, {})).rejects.toThrow('Invalid update section response');
    await expect(patchSection(3, { order: 2 })).rejects.toThrow('Invalid patch section response');
    await expect(deleteSection(3)).rejects.toThrow('Invalid delete section response');
  });
});
