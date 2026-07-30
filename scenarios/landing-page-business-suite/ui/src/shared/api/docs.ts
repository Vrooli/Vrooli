import { createClient } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { DocsService, type DocEntry as GeneratedDocEntry } from '@vrooli/proto-types/landing-page-business-suite/docs_pb';
import { CONNECT_API_BASE } from './common';

export interface DocEntry {
  name: string;
  path: string;
  isDir: boolean;
  children?: DocEntry[];
}

export interface DocContent {
  path: string;
  content: string;
  title: string;
}

const docsClient = createClient(DocsService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

function entryFromProto(entry: GeneratedDocEntry): DocEntry {
  return {
    name: entry.name,
    path: entry.path,
    isDir: entry.isDir,
    ...(entry.children.length > 0 ? { children: entry.children.map(entryFromProto) } : {}),
  };
}

export async function getDocsTree(): Promise<DocEntry[]> {
  const response = await docsClient.getDocsTree({});
  return response.entries.map(entryFromProto);
}

export async function getDocContent(path: string): Promise<DocContent> {
  const response = await docsClient.getDocContent({ path });
  return { path: response.path, content: response.content, title: response.title };
}
