import { BannerRegion, INSTANT_DAMPING, type MaybeBanner } from "./Banner";

const connectionLost: MaybeBanner = {
  id: "connection-lost",
  testId: "feedback.banner.connection",
  tone: "danger",
  priority: 90,
  title: "Unable to reach the API",
  description: "Retrying automatically.",
  actions: [
    {
      id: "retry",
      label: "Retry now",
      primary: true,
      onSelect: () => undefined,
    },
  ],
};

const transcribing: MaybeBanner = {
  id: "transcribing",
  testId: "feedback.banner.transcribing",
  tone: "info",
  priority: 25,
  title: "Transcribing audio",
  spin: true,
};

const staleMic: MaybeBanner = {
  id: "stale-mic",
  testId: "feedback.banner.stale-mic",
  tone: "warning",
  priority: 42,
  title: "Microphone still held",
};

/** One condition: the full notice, nothing collapsed. */
export function Single() {
  return (
    <BannerRegion
      banners={[connectionLost]}
      damping={INSTANT_DAMPING}
      testId="feedback.banner"
    />
  );
}

/**
 * Three conditions at once. The danger notice takes the one full slot and the
 * other two collapse — the whole point of the region is that N conditions cost
 * one banner's height plus a line, not N banners.
 */
export function Stacked() {
  return (
    <BannerRegion
      banners={[transcribing, connectionLost, staleMic]}
      damping={INSTANT_DAMPING}
      testId="feedback.banner"
    />
  );
}
