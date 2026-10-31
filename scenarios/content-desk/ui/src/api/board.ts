import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

import { decodeApiError } from "./client";

const REST_API_BASE = resolveApiBase({ appendSuffix: true });

export interface BoardLaunchTarget {
  scenario: string;
  node_id: string;
  status: string;
  release_rank: number;
  readiness_goal_reported: boolean;
  readiness_goal_exists: boolean | null;
  readiness_goal_closed: boolean | null;
  readiness_approved_commit: string;
  blocked_by: Array<{ name: string; status: string }>;
  next_action: string;
}

export interface BoardOfferReadiness {
  ladder_entries: number;
  launch_scenarios: string[];
  launch_targets: BoardLaunchTarget[];
  unknown_goal_state: number;
}

export interface BoardEnvelope {
  program: string;
  version: string;
  status: string;
  phase: string;
  signals: {
    offer_read_status?: string;
    offer_readiness?: BoardOfferReadiness;
    [key: string]: unknown;
  };
  errors?: Array<{ class: string; detail: string; where: string }>;
  evidence?: string[];
}

/**
 * Read the governed board envelope from the Content Desk browser edge. The
 * edge executes the content-desk.board-read declared program, so this returns
 * the same body a CLI agent reads. A missing dependency is a typed ApiError,
 * never an empty board.
 */
export async function readBoard(signal?: AbortSignal): Promise<BoardEnvelope> {
  const response = await fetch(buildApiUrl("/board", { baseUrl: REST_API_BASE }), {
    headers: { Accept: "application/json" },
    cache: "no-store",
    signal,
  });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as BoardEnvelope;
}
