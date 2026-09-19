import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import { fetchSubstrateBoard } from "../../api/substrate";
import { type SubstrateBoard } from "./model";

/**
 * Reads the Substrate Board.
 *
 * The refetch interval is deliberately slower than the scenario's other reads.
 * The device graph is collected on a 30s cache in `system-monitor` and the
 * topology of a machine changes far more slowly than its utilization does, so
 * polling faster would add load without adding information.
 *
 * `retry: false` is also deliberate. When a substrate source does not answer,
 * that IS the reading — the board renders it as instrument state and names the
 * failure. Silently retrying would replace a fact the operator needs with a
 * spinner.
 */
export function useSubstrateBoard(): UseQueryResult<SubstrateBoard> {
  return useQuery({
    queryKey: ["substrate", "board"],
    queryFn: fetchSubstrateBoard,
    refetchInterval: 30_000,
    retry: false,
  });
}
