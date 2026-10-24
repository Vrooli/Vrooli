import { createClient } from "@connectrpc/connect";
import { CommitmentsService, type Commitment } from "@vrooli/proto-types/personal-planner/v1/commitments/commitments_pb";

import { transport } from "./client";

export type { Commitment };

const commitmentsClient = createClient(CommitmentsService, transport);
export async function fetchCommitments(): Promise<Commitment[]> { const response = await commitmentsClient.listCommitments({}); return response.commitments; }
export async function createCommitment(input: { result: string; definitionOfDone?: string; promisedBoundary: string; timezone?: string; beneficiary?: string; assumptions?: string; scopeExclusions?: string; state?: string }): Promise<Commitment> { const response = await commitmentsClient.createCommitment({ ...input, definitionOfDone: input.definitionOfDone ?? "", timezone: input.timezone ?? "", beneficiary: input.beneficiary ?? "", assumptions: input.assumptions ?? "", scopeExclusions: input.scopeExclusions ?? "", state: input.state ?? "proposed" }); if (!response.commitment) throw new Error("The server returned no commitment"); return response.commitment; }
export async function updateCommitmentState(commitment: Commitment, state: string): Promise<Commitment> { const response = await commitmentsClient.updateCommitmentState({ id: commitment.id, state, expectedRevision: commitment.revision }); if (!response.commitment) throw new Error("The server returned no commitment"); return response.commitment; }
