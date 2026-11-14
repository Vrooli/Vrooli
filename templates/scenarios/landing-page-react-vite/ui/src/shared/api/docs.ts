import { createClient } from '@connectrpc/connect';
import { DocsService } from '@vrooli/proto-types/landing-page-react-vite/v1/docs_pb';
import type {
  DocEntry,
  GetDocContentResponse,
} from '@vrooli/proto-types/landing-page-react-vite/v1/docs_pb';

import { transport } from './client';

const docsClient = createClient(DocsService, transport);

/** Fetches the documentation tree (admin). */
export async function getDocsTree(): Promise<DocEntry[]> {
  const resp = await docsClient.getDocsTree({});
  return resp.entries;
}

/** Fetches a single documentation file's rendered content (admin). */
export function getDocContent(path: string): Promise<GetDocContentResponse> {
  return docsClient.getDocContent({ path });
}

export type { DocEntry, GetDocContentResponse };
