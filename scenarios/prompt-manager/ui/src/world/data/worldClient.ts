/**
 * Connect client for WorldService. The only module that knows the wire types;
 * everything above it speaks sim types.
 */
import { create, type MessageInitShape } from '@bufbuild/protobuf'
import { createClient, type Client, type Transport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { resolvePromptManagerConnectBase } from '@/lib/api'
import {
  GetLayoutRequestSchema,
  GetWorldConfigRequestSchema,
  SetLayoutRequestSchema,
  SetWorldConfigRequestSchema,
  StreamWorldFeedRequestSchema,
  WorldService,
  type WorldConfig,
  type WorldConfigSchema,
  type WorldEvent,
  type WorldLayout,
  type WorldLayoutSchema,
} from '@vrooli/proto-types/prompt-manager/v1/world/world_pb'

export type WorldConfigInit = MessageInitShape<typeof WorldConfigSchema>
export type WorldLayoutInit = MessageInitShape<typeof WorldLayoutSchema>

export type WorldServiceClient = Client<typeof WorldService>

export function createWorldTransport(): Transport {
  return createConnectTransport({ baseUrl: resolvePromptManagerConnectBase() })
}

export function createWorldClient(transport: Transport = createWorldTransport()): WorldServiceClient {
  return createClient(WorldService, transport)
}

export interface WorldApi {
  getConfig(): Promise<WorldConfig>
  setConfig(config: WorldConfigInit): Promise<WorldConfig>
  getLayout(scene: string): Promise<WorldLayout>
  setLayout(layout: WorldLayoutInit): Promise<WorldLayout>
  streamFeed(sinceSeq: bigint, signal: AbortSignal): AsyncIterable<WorldEvent>
}

/** Thin typed wrapper so callers never touch request schemas. */
export function createWorldApi(client: WorldServiceClient = createWorldClient()): WorldApi {
  return {
    getConfig: () => client.getWorldConfig(create(GetWorldConfigRequestSchema)),
    setConfig: (config) => client.setWorldConfig(create(SetWorldConfigRequestSchema, { config })),
    getLayout: (scene) => client.getLayout(create(GetLayoutRequestSchema, { scene })),
    setLayout: (layout) => client.setLayout(create(SetLayoutRequestSchema, { layout })),
    streamFeed: (sinceSeq, signal) => client.streamWorldFeed(create(StreamWorldFeedRequestSchema, { sinceSeq }), { signal }),
  }
}
