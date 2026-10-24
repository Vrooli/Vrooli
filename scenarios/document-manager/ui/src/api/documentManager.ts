import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE } from "./client";

export interface Collection {
  id: string;
  name: string;
  default_privacy_class: number;
  federated: boolean;
}
export interface DocumentRecord {
  id: string;
  content_sha256: string;
  source_name: string;
  detected_mime: string;
  privacy_class: number;
}
export interface QueryResult { unit_id: string; document_hash: string; anchor_uri: string; score: number; }

async function call<T>(procedure: string, body: unknown): Promise<T> {
  const response = await fetch(buildApiUrl(`/vrooli.document_manager.v1.${procedure}`, { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) throw new Error(`API request failed (${response.status})`);
  return (await response.json()) as T;
}

export const listCollections = async () => {
  const response = await call<{ collections?: Collection[] }>("corpus.CorpusService/ListCollections", { limit: 100 });
  return { collections: response.collections ?? [] };
};

export const listDocuments = async () => {
  const response = await call<{ documents?: DocumentRecord[] }>("intake.IntakeService/ListDocuments", { limit: 100 });
  return { documents: response.documents ?? [] };
};

export const queryCorpus = async (text: string, collectionId?: string) => {
  const response = await call<{ results?: QueryResult[]; partial?: boolean }>("retrieval.RetrievalService/Query", { text, collection_id: collectionId ?? "", caller_max_privacy: "PRIVACY_CLASS_INTERNAL", limit: 20 });
  return { results: response.results ?? [], partial: response.partial ?? false };
};

export const getDocument = async (id: string) => {
  const response = await call<{ document?: DocumentRecord }>("intake.IntakeService/GetDocument", { id });
  return { document: response.document };
};
