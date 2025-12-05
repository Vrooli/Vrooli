import { apiCall } from './common';

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

export function getDocsTree() {
  return apiCall<DocEntry[]>('/admin/docs/tree');
}

export function getDocContent(path: string) {
  return apiCall<DocContent>(`/admin/docs/content?path=${encodeURIComponent(path)}`);
}
