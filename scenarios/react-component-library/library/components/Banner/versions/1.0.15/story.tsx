import { useEffect, useRef, useState } from "react";
import { chromeTheme, StatusBarFill } from "@vrooli/react-component-library/ChromeTheme/1";
import { BannerRegion, INSTANT_DAMPING, renderedBackgroundOf, type MaybeBanner } from "./Banner";

const connectionLost: MaybeBanner = {
  id: "connection-lost",
  testId: "feedback.banner.connection",
  tone: "danger",
  priority: 90,
  title: "Unable to reach the API",
  description: "Retrying automatically.",
  actions: [{ id: "retry", label: "Retry now", primary: true, onSelect: () => undefined }],
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
    <BannerRegion banners={[connectionLost]} damping={INSTANT_DAMPING} testId="feedback.banner" />
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

/**
 * The invariant that matters on a phone: the colour handed to the OS status bar
 * is the colour the notice actually rendered as.
 *
 * That is the defect this story exists for. An earlier release kept a
 * hand-written colour table beside a token-derived `color-mix` palette, the two
 * drifted, and the notch showed a visibly different blue from the banner under
 * it — while every unit test passed, because only a browser resolves
 * `color-mix`.
 *
 * It asserts the contribution rather than the painted strip on purpose. The
 * strip's colour arrives through a custom property on the document element, and
 * that propagation behaves differently inside the composite review sheet's
 * canvas iframes — so asserting it there tests the harness, not the component.
 * What must never drift again is the value this component publishes, and that
 * is read straight from the service.
 */
export function ChromeMatch() {
  const [verdict, setVerdict] = useState("PENDING");
  // Scoped to this story's own subtree. A document-wide query passes in
  // isolation and fails on the composite review sheet, where several stories
  // share a page and the first banner belongs to a different one.
  const scopeRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    // Deliberately not requestAnimationFrame: on the review sheet each story is
    // an iframe in a grid, and a tile below the fold may never be given
    // animation frames — the probe would sit on "PENDING" while the component
    // underneath it was perfectly correct.
    let settled = false;
    const check = () => {
      if (settled) return true;
      const scope = scopeRef.current;
      const region = scope?.querySelector("[data-rcl-banner-region]");
      const banner = scope?.querySelector("[data-rcl-banner]");
      const measured = region?.getAttribute("data-rcl-banner-chrome");
      // Wait for all three to agree before recording anything. The region
      // publishes a fallback first and replaces it once the rendered colour
      // resolves, so a probe that latches on the first published value records
      // the fallback and calls a correct component broken.
      if (!banner || !measured || measured === "unmeasured") return false;
      const published = chromeTheme.current()?.statusColor ?? null;
      if (published !== measured) return false;
      const bannerBg = renderedBackgroundOf(banner);
      if (!bannerBg) return false;
      settled = true;
      setVerdict(
        published === bannerBg
          ? "CHROME-MATCH"
          : `CHROME-MISMATCH published ${published} vs rendered ${bannerBg}`,
      );
      return true;
    };

    if (check()) return;
    const unsubscribe = chromeTheme.subscribe(() => {
      check();
    });
    const retries = [50, 150, 400, 800, 1600].map((delay) => window.setTimeout(check, delay));
    // A bounded last word, so a genuine failure reports what it saw instead of
    // sitting on PENDING until the capture times out with nothing to read.
    const deadline = window.setTimeout(() => {
      if (settled) return;
      settled = true;
      const scope = scopeRef.current;
      const banner = scope?.querySelector("[data-rcl-banner]");
      const region = scope?.querySelector("[data-rcl-banner-region]");
      setVerdict(
        `CHROME-MISMATCH published ${chromeTheme.current()?.statusColor ?? "none"} vs measured ${region?.getAttribute("data-rcl-banner-chrome") ?? "?"} vs rendered ${banner ? renderedBackgroundOf(banner) : "?"}`,
      );
    }, 6000);
    return () => {
      unsubscribe();
      retries.forEach((timer) => {
        window.clearTimeout(timer);
      });
      window.clearTimeout(deadline);
    };
  }, []);

  return (
    <div ref={scopeRef} style={{ ["--rcl-safe-top" as string]: "44px" }}>
      <StatusBarFill />
      <BannerRegion
        banners={[transcribing]}
        damping={INSTANT_DAMPING}
        testId="feedback.banner"
        chromePriority={1000}
      />
      <output data-testid="feedback.banner.chrome-verdict">{verdict}</output>
    </div>
  );
}
