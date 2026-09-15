import { describe, expect, it, vi } from 'vitest';
import { fromBinary, fromJsonString, toBinary, toJsonString } from '@bufbuild/protobuf';
import { PresentationEditorResponseSchema, SavePresentationDraftRequestSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/product_presentation_pb';
import { createProductPresentationClient, formatPresentationDocument, parsePresentationDocument } from '../../../shared/api/productPresentation';
import { documentFixture, snapshot } from './testFixtures';

describe('generated presentation administration client', () => {
  it('uses generated Connect requests with session credentials, no-store, and lossless uint64 generation', async () => {
    let requestURL = '';
    let requestInit: RequestInit | undefined;
    let generation: bigint | undefined;
    let title: string | undefined;
    const fetcher = vi.fn<typeof fetch>().mockImplementation(async (input, init) => {
      const request = new Request(input, init);
      requestURL = request.url; requestInit = init;
      const binary = request.headers.get('content-type')?.includes('proto');
      const decoded = binary
        ? fromBinary(SavePresentationDraftRequestSchema, new Uint8Array(await request.arrayBuffer()))
        : fromJsonString(SavePresentationDraftRequestSchema, await request.text());
      generation = decoded.expectedGeneration;
      title = decoded.document?.pages[0]?.title;
      return new Response(binary ? toBinary(PresentationEditorResponseSchema, snapshot()) : toJsonString(PresentationEditorResponseSchema, snapshot()), {
        headers: { 'Content-Type': binary ? 'application/proto' : 'application/json' },
      });
    });
    const client = createProductPresentationClient(fetcher);
    const response = await client.saveDraft({ variantSlug: 'control', expectedGeneration: 9007199254740993n, document: documentFixture() });
    expect(requestURL).toContain('/landing_page_business_suite.v1.ProductPresentationAdminService/SaveDraft');
    expect(requestInit?.credentials).toBe('include');
    expect(requestInit?.cache).toBe('no-store');
    expect(generation).toBe(9007199254740993n);
    expect(response.state?.generation).toBe(9007199254740993n);
    expect(title).toBe('Configured first page');
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
  it('round-trips the entire generated document, typed content, and explicit order', () => {
    const document = documentFixture();
    const text = formatPresentationDocument(document);
    expect(text).toContain('"schema_version"');
    expect(text).toContain('"faq"');
    expect(parsePresentationDocument(text)).toEqual(document);
    expect(parsePresentationDocument(text).pages.map(page => page.id)).toEqual(['z-page', 'a-page']);
  });
  it('rejects unknown nested fields rather than silently stripping configuration', () => {
    expect(() => parsePresentationDocument('{"schema_version":1,"bundle":{"name":"Configured","secret_extension":"lost"}}')).toThrow();
  });
  it('rejects oversized documents before sending them', () => {
    expect(() => parsePresentationDocument(' '.repeat(8 * 1024 * 1024 + 1))).toThrow('8 MiB');
  });
});
