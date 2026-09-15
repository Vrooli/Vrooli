import { fromJsonString, toJsonString } from '@bufbuild/protobuf';
import { createClient, type Client } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { ProductPresentationAdminService } from '@vrooli/proto-types/landing-page-business-suite/v1/product_presentation_pb';
import {
  ProductPresentationDocumentSchema,
  type ProductPresentationDocument,
} from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import { CONNECT_API_BASE } from './common';
export { decodeProductPresentation } from '../../surfaces/public-landing/presentation/decode';

export type ProductPresentationClient = Client<typeof ProductPresentationAdminService>;
export type { PresentationEditorResponse } from '@vrooli/proto-types/landing-page-business-suite/v1/product_presentation_pb';
export type { ProductPresentationDocument, ResolvedProductPresentation } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';

/** Authenticated, non-cached Connect transport. No public config or REST fallback. */
export function createProductPresentationClient(fetchImplementation?: typeof fetch): ProductPresentationClient {
  return createClient(ProductPresentationAdminService, createScenarioConnectTransport({
    baseUrl: CONNECT_API_BASE,
    fetch: (input, init) => (fetchImplementation ?? globalThis.fetch)(input, {
      ...init, credentials: 'include', cache: 'no-store',
    }),
  }));
}

export const productPresentationClient = createProductPresentationClient();

/** JSON is the generated protobuf document shape, including typed content oneofs. */
export function parsePresentationDocument(text: string): ProductPresentationDocument {
  if (new TextEncoder().encode(text).byteLength > 8 * 1024 * 1024) {
    throw new Error('The presentation document exceeds the 8 MiB editing limit.');
  }
  return fromJsonString(ProductPresentationDocumentSchema, text, { ignoreUnknownFields: false });
}

export function formatPresentationDocument(document: ProductPresentationDocument): string {
  return toJsonString(ProductPresentationDocumentSchema, document, {
    useProtoFieldName: true, alwaysEmitImplicit: true, prettySpaces: 2,
  });
}
