export type ChannelListing = {
  descriptor: { id: string; displayName: string; setup: { friction: number } };
  availability: "available" | "unavailable" | "unknown";
  reason?: string;
  implemented: boolean;
};

export async function listChannels(signal?: AbortSignal): Promise<ChannelListing[]> {
  const response = await fetch("/api/v1/channels", { signal });
  if (!response.ok) throw new Error(`Unable to load channels (${response.status})`);
  return response.json() as Promise<ChannelListing[]>;
}

export async function createBinding(input: { agentId: string; channelId: string; address: string; threadKey?: string }, signal?: AbortSignal): Promise<void> {
  const response = await fetch("/vrooli.switchboard.v1.channels.ChannelService/CreateBinding", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ agentId: input.agentId, channelId: input.channelId, address: input.address, threadKey: input.threadKey ?? "" }),
    signal,
  });
  if (!response.ok) throw new Error(`Unable to attach agent (${response.status})`);
}
