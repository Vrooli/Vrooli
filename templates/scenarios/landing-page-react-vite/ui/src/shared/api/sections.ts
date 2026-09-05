import type { JsonObject } from '@bufbuild/protobuf';
import { createClient } from '@connectrpc/connect';

// Section content is a google.protobuf.Struct (dynamic JSON). The UI holds it as
// a plain record; this bridges to the proto Struct init type at the boundary.
type SectionContent = Record<string, unknown>;
import { ContentService } from '@vrooli/proto-types/landing-page-react-vite/v1/content_pb';
import type { ContentSection } from '@vrooli/proto-types/landing-page-react-vite/v1/content_pb';

import { transport } from './client';

const contentClient = createClient(ContentService, transport);

/** Lists the enabled sections for a variant (public view). */
export async function getSections(variantId: bigint): Promise<ContentSection[]> {
  const resp = await contentClient.getPublicSections({ variantId });
  return resp.sections;
}

/** Lists all sections for a variant, including disabled ones (admin view). */
export async function getAdminSections(variantId: bigint): Promise<ContentSection[]> {
  const resp = await contentClient.getSections({ variantId });
  return resp.sections;
}

/** Fetches a single section by id (admin). */
export async function getSection(sectionId: bigint): Promise<ContentSection | undefined> {
  const resp = await contentClient.getSection({ id: sectionId });
  return resp.section;
}

/** Replaces a section's content payload (admin). */
export async function updateSection(
  sectionId: bigint,
  content: SectionContent,
): Promise<ContentSection | undefined> {
  const resp = await contentClient.updateSection({
    id: sectionId,
    content: content as JsonObject,
  });
  return resp.section;
}

export interface CreateSectionInput {
  variantId: bigint;
  sectionType: string;
  content: SectionContent;
  order: number;
  enabled?: boolean;
}

/** Creates a new section for a variant (admin). */
export async function createSection(input: CreateSectionInput): Promise<ContentSection | undefined> {
  const resp = await contentClient.createSection({
    variantId: input.variantId,
    sectionType: input.sectionType,
    content: input.content as JsonObject,
    order: input.order,
    enabled: input.enabled ?? true,
  });
  return resp.section;
}

/** Deletes a section by id (admin). */
export async function deleteSection(sectionId: bigint): Promise<boolean> {
  const resp = await contentClient.deleteSection({ id: sectionId });
  return resp.deleted;
}

export type { ContentSection };
