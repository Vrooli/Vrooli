import { create } from "@bufbuild/protobuf";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { projectsClient } from "../../api/projects";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import type { SourceMedia } from "../../api/projects";
import { VideoAudioPolicy, VideoTimingSchema, type VideoTiming } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";

type Props = {
  projectId: string;
  source: SourceMedia;
  timing: VideoTiming;
  disabled: boolean;
  onChange: (timing: VideoTiming) => void;
};

function formatPts(pts: bigint, numerator: bigint, denominator: bigint): string {
  if (denominator <= 0n || numerator <= 0n) return `PTS ${pts}`;
  const millis = pts * numerator * 1000n / denominator;
  const sign = millis < 0n ? "−" : "";
  const absolute = millis < 0n ? -millis : millis;
  return `${sign}${absolute / 1000n}.${(absolute % 1000n).toString().padStart(3, "0")}s`;
}

export function VideoTimingEditor({ projectId, source, timing, disabled, onChange }: Props) {
  const { t } = useTranslation();
  const clock = useInfiniteQuery({
    queryKey: ["source-clock", projectId, source.id, timing.clockSha256],
    initialPageParam: "",
    queryFn: ({ pageParam }) => projectsClient.getSourceMediaClockPage({
      projectId, mediaId: source.id, expectedClockSha256: timing.clockSha256, pageSize: 256, pageToken: pageParam,
    }),
    getNextPageParam: (page) => page.nextPageToken || undefined,
    enabled: Boolean(timing.clockSha256),
    retry: false,
  });
  const pages = clock.data?.pages ?? [];
  const points = pages.flatMap((page) => page.points);
  const clockMismatch = pages.some((page) => page.clockSha256 !== timing.clockSha256 ||
    page.timeBaseNumerator !== timing.timeBaseNum || page.timeBaseDenominator !== timing.timeBaseDen);
  const audioFacts = source.inspectionReceipt?.audio;
  const audioNotice = !audioFacts || audioFacts.version !== 1 ? strings.shot.videoAudioUnreported :
    audioFacts.streams.length === 0 ? strings.shot.videoAudioNone : strings.shot.videoAudioPresent;
  const starts = points.filter((point) => point.pts < timing.trimEndPts);
  const ends = points.filter((point) => point.pts > timing.trimStartPts).map((point) => ({ ordinal: point.ordinal, pts: point.pts }));
  if (timing.sourceEndPts > timing.trimStartPts && !ends.some((point) => point.pts === timing.sourceEndPts)) {
    ends.push({ ordinal: Number.MAX_SAFE_INTEGER, pts: timing.sourceEndPts });
  }
  const update = (patch: Partial<VideoTiming>) => onChange(create(VideoTimingSchema, { ...timing, ...patch, sampleBinding: undefined, sampleBaseRecipeVersion: 0 }));
  const field = (label: string, value: string, options: Array<{ value: string; label: string }>, onValue: (value: string) => void) => (
    <label className="shot-field"><span>{label}</span><select value={value} disabled={disabled || clockMismatch || clock.isLoading} onChange={(event) => onValue(event.target.value)}>
      <option value="">{t(strings.shot.videoChooseFrame)}</option>
      {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
    </select></label>
  );
  const startOptions = starts.map((point) => ({ value: point.pts.toString(), label: `${t(strings.shot.videoFrame, { frame: point.ordinal + 1 })} · ${formatPts(point.pts, timing.timeBaseNum, timing.timeBaseDen)}` }));
  if (!startOptions.some((option) => option.value === timing.trimStartPts.toString())) startOptions.push({ value: timing.trimStartPts.toString(), label: `${t(strings.shot.videoSavedPoint)} · ${formatPts(timing.trimStartPts, timing.timeBaseNum, timing.timeBaseDen)}` });
  const endOptions = ends.map((point) => ({ value: point.pts.toString(), label: `${point.ordinal === Number.MAX_SAFE_INTEGER ? t(strings.shot.videoEndOfSource) : t(strings.shot.videoAfterFrame, { frame: point.ordinal })} · ${formatPts(point.pts, timing.timeBaseNum, timing.timeBaseDen)}` }));
  if (!endOptions.some((option) => option.value === timing.trimEndPts.toString())) endOptions.push({ value: timing.trimEndPts.toString(), label: `${t(strings.shot.videoSavedPoint)} · ${formatPts(timing.trimEndPts, timing.timeBaseNum, timing.timeBaseDen)}` });
  const playbackValue = `${timing.playbackRateNum}/${timing.playbackRateDen}`;
  const customPlayback = playbackValue !== "1/1" && playbackValue !== "2/1";
  const durationNumerator = (timing.trimEndPts - timing.trimStartPts) * timing.timeBaseNum * BigInt(timing.playbackRateDen) * BigInt(timing.outputRateNum);
  const durationDenominator = timing.timeBaseDen * BigInt(timing.playbackRateNum) * BigInt(timing.outputRateDen || 1);
  const frameCount = durationDenominator > 0n ? (durationNumerator + durationDenominator - 1n) / durationDenominator : 301n;
  const secondsNumerator = (timing.trimEndPts - timing.trimStartPts) * timing.timeBaseNum * BigInt(timing.playbackRateDen);
  const secondsDenominator = timing.timeBaseDen * BigInt(timing.playbackRateNum);
  const clipTooLong = secondsDenominator <= 0n || secondsNumerator > 10n * secondsDenominator || frameCount > 300n;
  return <section className="shot-video-timing" data-testid={selectors.shot.videoTiming} aria-label={t(strings.shot.videoTiming)}>
    <p className="shot-help">{t(strings.shot.videoTimingHelp)}</p>
    <p className="shot-help" role="status">{t(audioNotice)}</p>
    {timing.audioPolicy === VideoAudioPolicy.PASSTHROUGH && <Button variant="outline" className="min-h-11" disabled={disabled} onClick={() => update({ audioPolicy: VideoAudioPolicy.SILENT })}>{t(strings.shot.videoUseSilentOutput)}</Button>}
    {clock.error && <div role="alert"><p>{errorMessage(clock.error, t)}</p><Button variant="outline" className="min-h-11" onClick={() => void clock.refetch()}>{t(strings.media.retry)}</Button></div>}
    {clockMismatch && <p role="alert">{t(strings.shot.videoClockMismatch)}</p>}
    {clock.isLoading && <p role="status">{t(strings.shot.videoClockLoading)}</p>}
    <div className="shot-fields shot-fields--pair">
      {field(t(strings.shot.videoTrimStart), timing.trimStartPts.toString(), startOptions, (value) => { if (value) update({ trimStartPts: BigInt(value) }); })}
      {field(t(strings.shot.videoTrimEnd), timing.trimEndPts.toString(), endOptions, (value) => { if (value) update({ trimEndPts: BigInt(value) }); })}
    </div>
    <details><summary>{t(strings.shot.videoClockDetails)}</summary><p>{t(strings.shot.videoClockDetail, { timeBase: `${timing.timeBaseNum}/${timing.timeBaseDen}`, startPts: timing.trimStartPts.toString(), endPts: timing.trimEndPts.toString() })}</p></details>
    {clock.hasNextPage && <Button variant="outline" className="min-h-11" disabled={clock.isFetchingNextPage} onClick={() => void clock.fetchNextPage()}>{clock.isFetchingNextPage ? t(strings.shot.videoClockLoading) : t(strings.shot.videoLoadClock)}</Button>}
    <div className="shot-fields shot-fields--pair">
      <label className="shot-field"><span>{t(strings.shot.videoPlaybackRate)}</span><select disabled={disabled} value={playbackValue} onChange={(event) => { const [num, den] = event.target.value.split("/").map(Number); update({ playbackRateNum: num ?? 1, playbackRateDen: den ?? 1 }); }}>{customPlayback && <option value={playbackValue}>{t(strings.shot.videoSavedRate, { rate: playbackValue })}</option>}<option value="1/1">{t(strings.shot.videoRate1x)}</option><option value="2/1">{t(strings.shot.videoRate2x)}</option></select></label>
      <label className="shot-field"><span>{t(strings.shot.videoOutputRate)}</span><select disabled={disabled} value={String(timing.outputRateNum)} onChange={(event) => update({ outputRateNum: Number(event.target.value), outputRateDen: 1 })}><option value="24">{t(strings.shot.videoRate24)}</option><option value="30">{t(strings.shot.videoRate30)}</option></select></label>
    </div>
    {timing.trimEndPts <= timing.trimStartPts && <p role="alert">{t(strings.shot.videoInvalidRange)}</p>}
    {timing.trimEndPts > timing.trimStartPts && clipTooLong && <p role="alert">{t(strings.shot.videoClipTooLong)}</p>}
  </section>;
}
