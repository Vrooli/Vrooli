import { describe, expect, it } from 'vitest';
import { fromJsonString } from '@bufbuild/protobuf';
import { PublicBrandingSchema, UpdateBrandingRequestSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/branding_pb';

// Guards against resolving a stale installed proto copy: an unknown field makes
// admin saves throw and silently drops public business identity.
describe('branding proto contract', () => {
  it('accepts business identity and legal document fields', () => {
    const update = fromJsonString(UpdateBrandingRequestSchema, JSON.stringify({ legal_name: 'Northwind LLC', contact_address: '1 Main St', privacy_policy_markdown: '# P', terms_markdown: '# T', privacy_effective_date: '2026-09-16', terms_effective_date: '2026-09-16' }), { ignoreUnknownFields: false });
    expect(update.legalName).toBe('Northwind LLC');
    expect(PublicBrandingSchema.fields.map(field => field.name)).toEqual(expect.arrayContaining(['support_email', 'legal_name', 'contact_address', 'privacy_policy_markdown', 'terms_markdown']));
  });
});
