import type { MessageInitShape } from '@bufbuild/protobuf';
import { createClient } from '@connectrpc/connect';
import {
  BrandingService,
  UpdateBrandingRequestSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/branding_pb';
import type {
  SiteBranding,
  PublicBranding,
} from '@vrooli/proto-types/landing-page-react-vite/v1/branding_pb';

import { transport } from './client';

const brandingClient = createClient(BrandingService, transport);

export type BrandingUpdate = MessageInitShape<typeof UpdateBrandingRequestSchema>;

/** Fetches the full site branding record (admin). */
export async function getBranding(): Promise<SiteBranding | undefined> {
  const resp = await brandingClient.getBranding({});
  return resp.branding;
}

/** Updates site branding fields (admin). Only provided fields change. */
export async function updateBranding(data: BrandingUpdate): Promise<SiteBranding | undefined> {
  const resp = await brandingClient.updateBranding(data);
  return resp.branding;
}

/** Clears a single branding field back to its default (admin). */
export async function clearBrandingField(field: string): Promise<SiteBranding | undefined> {
  const resp = await brandingClient.clearBrandingField({ field });
  return resp.branding;
}

/** Fetches the public branding subset (no auth). */
export async function getPublicBranding(): Promise<PublicBranding | undefined> {
  const resp = await brandingClient.getPublicBranding({});
  return resp.branding;
}

export type { SiteBranding, PublicBranding };
