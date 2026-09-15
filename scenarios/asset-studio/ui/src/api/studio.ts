import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import {
	CreateIdentityRequestSchema,
	CreateRenderRequestSchema,
	JudgeConformanceRequestSchema,
	ListIdentitiesRequestSchema,
	ReleaseAssetRequestSchema,
	ResolveSpecRequestSchema,
	SelectCandidateRequestSchema,
	StudioService,
	type AssetReference,
  type Identity,
} from "@vrooli/proto-types/asset-studio/v1/studio/studio_pb";
import { transport } from "./client";

const client = createClient(StudioService, transport);

export async function listIdentities(): Promise<Identity[]> {
  return (await client.listIdentities(create(ListIdentitiesRequestSchema))).identities;
}

export async function resolveProductSpec(identityId: string, productName: string): Promise<{ specId: string; payload: string }> {
  const response = await client.resolveSpec(create(ResolveSpecRequestSchema, { template: "{{product}} product still, consistent with its versioned identity", fields: { product: productName }, identityVersionIds: [identityId] }));
  return { specId: response.specId, payload: response.resolvedPayload };
}
export async function createRender(specId: string, estimatedCost: number): Promise<AssetReference[]> {
  return (await client.createRender(create(CreateRenderRequestSchema, { specId, estimatedCost, candidateCount: 2 })).then((response) => response)).candidates;
}
export async function selectCandidate(assetId: string): Promise<AssetReference> {
  const response = await client.selectCandidate(create(SelectCandidateRequestSchema, { assetId }));
  if (!response.selected) throw new Error("Asset Studio did not return the selected candidate");
  return response.selected;
}
export async function judgeConformance(assetId: string, identityId: string): Promise<void> {
  await client.judgeConformance(create(JudgeConformanceRequestSchema, { assetId, identityVersionId: identityId, actorId: "operator-ui", actorKind: "operator", passed: true, basis: "prose-only" }));
}
export async function releaseAsset(assetId: string): Promise<AssetReference> {
  const response = await client.releaseAsset(create(ReleaseAssetRequestSchema, { assetId }));
  if (!response.asset) throw new Error("Asset Studio did not return the released asset");
  return response.asset;
}

export async function createProductIdentity(input: { name: string; form: string; finish: string }): Promise<Identity> {
  const response = await client.createIdentity(create(CreateIdentityRequestSchema, {
    identity: { name: input.name, kind: "product", traits: { form: input.form, finish: input.finish }, credentialClaims: "" },
    actorId: "operator-ui",
    actorKind: "operator",
  }));
  if (!response.identity) throw new Error("Asset Studio did not return the created identity");
  return response.identity;
}
