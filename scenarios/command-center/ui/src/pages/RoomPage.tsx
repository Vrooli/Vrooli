import { useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useParams, useSearchParams } from "react-router-dom";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { AmbientShell } from "../components/AmbientShell";
import { RoomVisual } from "../components/RoomVisual";
import { useBoardController } from "../lib/boardContext";
import { fetchRoom, hasValue, type RoomResponse } from "../lib/api";
import { nextMeasuredBeat } from "../lib/hero";
import { readingsForBeat } from "../lib/beat";

export default function RoomPage() {
  const { roomId = "mission-control" } = useParams();
  const [searchParams] = useSearchParams();
  const visualDiff = searchParams.get("visualDiff") === "1";
  const board = useBoardController();
  const { data, isLoading, error, isFetching, dataUpdatedAt } = useQuery<RoomResponse>({
    queryKey: ["room", roomId, board.samples],
    queryFn: () => fetchRoom(roomId, board.samples),
    refetchInterval: (query) => {
      const ttls = query.state.data?.readings.map((reading) => reading.ttlSeconds).filter((ttl) => ttl > 0) ?? [];
      return ttls.length ? Math.max(5, Math.min(...ttls)) * 1000 : 30_000;
    },
  });
  const room = board.rooms.find((entry) => entry.id === roomId) ?? data?.room ?? { id: roomId, title: roomId.replace(/-/g, " "), theme: "ground-control", composition: "orbital-field" };
  const beats = useMemo(() => room.beats ?? [], [room.beats]);
  // The visual harness addresses beats directly. Production continues to use
  // the controller's live cycle, while a pinned capture must not race the
  // controller's wall-clock progress during route navigation.
  const requestedVisualBeat = Number.parseInt(searchParams.get("beat") ?? "0", 10);
  const beatIndex = visualDiff && beats.length > 0
    ? Math.max(0, Math.min(beats.length - 1, Number.isFinite(requestedVisualBeat) ? requestedVisualBeat : 0))
    : board.beatIndex;
  const beat = beats[beatIndex];
  const theme = room.theme ?? "ground-control";
  const readings = useMemo(() => data?.readings ?? [], [data]);
  const renderReadings = useMemo(() => visualDiff ? readings.map((reading) => ({
    ...reading,
    value: reading.sample?.value ?? null,
    rows: reading.sample?.rows ?? [],
    ladder: reading.sample?.ladder ?? undefined,
  })) : readings, [readings, visualDiff]);
  const beatReadings = useMemo(() => readingsForBeat(beat, renderReadings), [beat, renderReadings]);
  const visible = useMemo(() => (board.samples === "hide" ? beatReadings.filter(hasValue) : beatReadings), [board.samples, beatReadings]);
  const constellations = useMemo(() => visualDiff ? data?.constellations?.map((constellation) => ({
    ...constellation,
    readings: constellation.readings.map((reading) => ({
      ...reading,
      value: reading.sample?.value ?? null,
      rows: reading.sample?.rows ?? [],
      ladder: reading.sample?.ladder ?? undefined,
    })),
  })) : data?.constellations, [data?.constellations, visualDiff]);
  // A panorama's hero counts the whole board, so every one of its own readings (the rooms' headlines) supports.
  useEffect(() => {
    if (board.samples !== "hide" || !beats.length || !beat || visible.length === 0) return;
    const measuredIDs = new Set(visible.map((reading) => reading.id));
    if (beat.hero && measuredIDs.has(beat.hero)) return;
    const next = nextMeasuredBeat(beats, measuredIDs, beatIndex);
    if (next >= 0 && next !== beatIndex) board.selectBeat(next);
  }, [beat, beats, board, board.samples, visible, beatIndex]);
  const hasSamples = visible.some((reading) => reading.sample !== null);
  const index = board.rooms.findIndex((entry) => entry.id === roomId);
  const position = index >= 0 ? `ROOM ${index + 1} OF ${board.rooms.length}` : "ROOM";
  const sources = Object.entries(data?.sources ?? {});
  const sourceState = sources.some(([, meta]) => meta.staleness_ts) ? "partial" : "ready";
  // Origin is shown only where it differs: per tile in a mixed room, once in the source strip when every reading shares a non-local origin.
  const origins = new Set(visible.map((reading) => reading.origin_env || "local"));
  const sharedOrigin = origins.size === 1 && !origins.has("local") ? visible[0]?.origin_display || visible[0]?.origin_env : null;

  return (
    <AmbientShell
      theme={theme}
      title={room.title}
      position={position}
      legend={hasSamples}
      status={
        <ExperienceSurface surfaceId="sources" as="div" data-testid="room-sources" className="cc-sources" state={sourceState} aria-label="Source availability">
          {sources.length === 0 ? <span className="cc-source" data-answering="none">{isLoading ? "reading sources" : "no source read"}</span> : null}
          {sources.map(([name, meta]) => {
            const status = meta.staleness_ts ? "stale" : meta.integration_status ?? "available";
            const detail = meta.staleness_ts ? `last fetch failed ${meta.staleness_ts}` : meta.integration_reason_code ?? status;
            // The dot carries the status; a source that did not answer still states why, in the source strip.
            return (
              <span key={name} className="cc-source" data-answering={status} title={`${meta.integration_id ?? name} · ${detail}`}>
                <span className="cc-source-dot" aria-hidden="true" />{name}
                <span className={status === "available" ? "cc-visually-hidden" : undefined}>: {status === "available" ? status : detail}</span>
              </span>
            );
          })}
          {sharedOrigin ? <span className="cc-source" data-origin data-testid="room-origin">{sharedOrigin}</span> : null}
          {isFetching ? <span className="cc-source cc-source-fetching" aria-label="refreshing">·</span> : null}
        </ExperienceSurface>
      }
    >
      <RoomVisual roomId={roomId} room={room} beatIndex={beatIndex} readings={renderReadings} samples={board.samples} constellations={constellations} sources={data?.sources} isLoading={isLoading} error={error} forcedTier={searchParams.get("tier")} seed={searchParams.get("seed") ?? ""} dataUpdatedAt={dataUpdatedAt} onReadingSeconds={(seconds) => board.reportReadingSeconds(roomId, seconds)} />
    </AmbientShell>
  );
}
