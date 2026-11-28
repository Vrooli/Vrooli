import { create } from "@bufbuild/protobuf";
import {
  GetPrecommitCandidateCheckRequestSchema,
  GetPrecommitConfigRequestSchema,
  CancelPrecommitCandidateCheckRequestSchema,
  StartPrecommitCandidateCheckRequestSchema,
  type PrecommitCandidateCheckResponse,
} from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { repoClient } from "../../lib/connect";

export type PrecommitCandidateCheck = PrecommitCandidateCheckResponse;

export async function startPrecommitCandidateCheck(repositoryId: string, retryTerminal = false): Promise<PrecommitCandidateCheck> {
  const response = await repoClient.startPrecommitCandidateCheck(create(StartPrecommitCandidateCheckRequestSchema, { repositoryId, retryTerminal }));
  return response;
}

export async function getPrecommitCandidateCheck(repositoryId: string, jobId: string): Promise<PrecommitCandidateCheck> {
  return repoClient.getPrecommitCandidateCheck(create(GetPrecommitCandidateCheckRequestSchema, { repositoryId, jobId }));
}

export async function cancelPrecommitCandidateCheck(repositoryId: string, jobId: string): Promise<PrecommitCandidateCheck> {
  return repoClient.cancelPrecommitCandidateCheck(create(CancelPrecommitCandidateCheckRequestSchema, { repositoryId, jobId }));
}

export async function precommitChecksEnabled(repositoryId: string): Promise<boolean> {
  const config = await repoClient.getPrecommitConfig(create(GetPrecommitConfigRequestSchema, { repositoryId }));
  return config.enabled && config.runBeforeCommit;
}

