import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.1.0";
import { selectors } from "../../consts/selectors";

function fixtureState(): string {
  return new URLSearchParams(window.location.search).get("fixture") ?? "default";
}

export function DeliveryOverview() {
  const fixture = fixtureState();
  const unavailable = fixture === "partial" || fixture === "gate-failed";
  const evidence = fixture === "green-pixel" ? "pixel-grade" : "semantic";
  return (
    <Card className="lg:col-span-3">
      <CardHeader><CardTitle>Delivery evidence</CardTitle></CardHeader>
      <CardContent className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
        <div role="table" data-testid={selectors.delivery.targetMatrix}><div role="rowgroup"><div role="row"><div role="cell"><strong>Target matrix</strong><p>{unavailable ? "partial" : "probed"}</p></div></div></div></div>
        <div role="status" data-testid={selectors.delivery.targetDisposition}><strong>Target disposition</strong><p>{unavailable ? "unavailable" : fixture === "validate-only" ? "validate-only" : "available"}</p></div>
        <div role="status" data-testid={selectors.delivery.gateVerdict}><strong>Release gate</strong><p>{fixture === "gate-failed" ? "failed" : "pending"}</p></div>
        <div role="status" data-testid={selectors.delivery.readinessSummary}><strong>Release readiness</strong><p>{unavailable ? "blocked" : "unknown"}</p></div>
        <div role="note" data-testid={selectors.delivery.executingNode}><strong>Executing node</strong><p>local host</p></div>
        <div role="status" data-testid={selectors.delivery.rowPromotability}><strong>Evidence grade</strong><p>{evidence}</p></div>
        <button type="button" className="min-h-11" data-testid={selectors.delivery.generateProject}>Generate project</button>
      </CardContent>
    </Card>
  );
}

export function DistributionSurface() {
  const fixture = fixtureState();
  const waiting = fixture === "verification-pending";
  const artifactBlocked = fixture === "target-api-below-floor";
  const unverified = fixture === "unverified-developer";
  const playReady = fixture === "play-ready";
  const availability = (channel: string) => channel === "ADB internal"
    ? "available"
    : waiting ? "verification-pending"
      : unverified || artifactBlocked ? "blocked"
        : playReady ? "available" : "unavailable";
  return (
    <section data-testid={selectors.pages.distribution} className="flex flex-col gap-4" aria-labelledby="distribution-heading">
      <div><h2 id="distribution-heading" className="text-2xl font-semibold">Distribution</h2><p className="text-app-muted-foreground">Independent channel verdicts from the delivery backend.</p></div>
      <Card role="list" data-testid={selectors.distribution.channelList}>
        <CardHeader role="listitem"><CardTitle>Channels</CardTitle></CardHeader>
        <CardContent role="listitem" className="grid gap-3 text-sm md:grid-cols-3">
          {(["Play", "Verified sideload", "ADB internal"] as const).map((channel) => (
            <article role="listitem" key={channel}>
              <div className="flex items-center justify-between gap-2"><strong role="status" data-testid={selectors.distribution.channelAvailability}>{channel}: {availability(channel)}</strong><StatusBadge tone={channel === "ADB internal" || playReady ? "success" : "warning"}>{availability(channel)}</StatusBadge></div>
              <p role="note" data-testid={selectors.distribution.channelRequirement}>requires verified developer and a compliant artifact</p>
              <p role="note" data-testid={selectors.distribution.channelBlockingRung}>{waiting ? "waiting on Google" : artifactBlocked ? "target API floor" : unverified ? "developer verification" : playReady ? "production review" : "developer verification"}</p>
              <button type="button" className="min-h-11" data-testid={selectors.distribution.channelNextAction}>{waiting ? "wait for Google decision" : playReady ? "submit production review" : "complete the named readiness rung"}</button>
              <p role="note" data-testid={selectors.distribution.channelDeadline}>target API deadline: 31 August 2026</p>
            </article>
          ))}
        </CardContent>
      </Card>
      <Card role="note" data-testid={selectors.distribution.artifactIdentity}><CardContent>Artifact: pending · Android package · semantic evidence</CardContent></Card>
      {fixture === "non-promotable-artifact" && <p data-testid={selectors.distribution.promotabilityBlock} role="status">This evidence cannot gate a release.</p>}
    </section>
  );
}

export function ReadinessSurface() {
  const fixture = fixtureState();
  const waiting = fixture === "verification-pending";
  const rungs = ["registration", "developer-verification", "signing", "target-api", "internal-testing", "production-review"];
  return (
    <section data-testid={selectors.pages.readiness} className="flex flex-col gap-4" aria-labelledby="readiness-heading">
      <div><h2 id="readiness-heading" className="text-2xl font-semibold">Release readiness</h2><p className="text-app-muted-foreground">The ordered Google readiness ladder.</p></div>
      <Card role="list" data-testid={selectors.readiness.rungList}><CardContent role="listitem" className="grid gap-3 md:grid-cols-2">
        {rungs.map((rung, index) => <article role="listitem" key={rung} className="border-b border-app-border pb-3"><div className="flex items-center justify-between gap-2"><strong role="status" data-testid={selectors.readiness.rungState}>{index === 0 ? "ready" : waiting && index === 1 ? "verification-pending" : "unavailable"}</strong><span>{rung}</span></div><p role="note" data-testid={selectors.readiness.rungOwner}>{index < 2 ? "owner" : "ramp or owner"}</p><p role="note" data-testid={selectors.readiness.rungCost}>{index === 0 ? "$25 registration" : "no additional cost"}</p><button type="button" className="min-h-11" data-testid={selectors.readiness.rungNextAction}>{index === 0 ? "register with Google Play" : waiting && index === 1 ? "wait for Google decision" : "complete the preceding rung"}</button>{index === 1 && <p role="note" data-testid={selectors.readiness.rungDeadline}>verification deadline: 30 September 2026</p>}</article>)}
      </CardContent></Card>
    </section>
  );
}

export function RunReviewSurface() {
  const fixture = fixtureState();
  const routeState = window.location.pathname.split("/").pop() || "";
  const state = fixture !== "default" ? fixture : routeState === "runs" ? "default" : routeState;
  const interrupted = state === "lease-lost";
  const partial = state === "partial";
  const nonPromotable = state === "partial" || state === "recording-unusable" || state === "non-promotable";
  return (
    <section data-testid={selectors.pages.runReview} className="flex flex-col gap-4" aria-labelledby="run-review-heading">
      <div><h2 id="run-review-heading" className="text-2xl font-semibold">Run review</h2><p className="text-app-muted-foreground">Expected and observed evidence for each chapter.</p></div>
      <Card role="list" data-testid={selectors.runReview.chapterList}><CardContent role="listitem" className="space-y-3 text-sm">
        <p role="status" data-testid={selectors.runReview.runVerdict}>{interrupted ? "incomplete" : partial ? "partial" : "pending"}</p>
        <div className="border-t border-app-border pt-3"><p role="note" data-testid={selectors.runReview.chapterExpected}>expected: application launches</p><p role="note" data-testid={selectors.runReview.chapterObserved}>observed: {partial ? "web content unavailable" : "pending"}</p><p role="note" data-testid={selectors.runReview.chapterStrategy}>strategy: android emulator · semantic</p><p role="note" data-testid={selectors.runReview.exceededBound}>settle bound: 30 seconds</p><p role="note" data-testid={selectors.runReview.missingOffset}>offset: not available</p></div>
        {interrupted && <p data-testid={selectors.runReview.leaseInterruption} role="alert">Device lease was lost; this is an infrastructure interruption, not a product failure.</p>}
        <p role="region" data-testid={selectors.runReview.recording}>recording: unavailable</p>
        {nonPromotable || interrupted ? <p role="alert" data-testid={selectors.runReview.promotabilityNotice}>This evidence cannot gate a release.</p> : null}
      </CardContent></Card>
    </section>
  );
}

export function TargetDetailSurface() {
  const fixture = fixtureState();
  const terminal = fixture === "terminal-linux";
  const contention = fixture === "device-leased-elsewhere";
  const targetId = window.location.pathname.split("/").pop() || "target-local";
  const missing = fixture === "no-gui-session"
    ? "GUI session"
    : fixture === "missing-runtime-variant"
      ? "x86_64 simulator runtime"
      : fixture === "missing-toolchain"
        ? "Android SDK toolchain"
      : fixture === "no-acceleration"
          ? "/dev/kvm hardware acceleration"
          : fixture === "physical-unpaired"
            ? "USB debugging authorization"
            : fixture === "physical-wireless-expired"
              ? "wireless debugging pairing"
          : terminal
            ? "Apple toolchain"
            : "Android emulator toolchain";
  const nextAction = fixture === "no-gui-session"
    ? "log in to the GUI session"
    : fixture === "missing-runtime-variant"
      ? "install the simulator runtime"
      : fixture === "missing-toolchain"
        ? "install the Android SDK toolchain"
        : fixture === "no-acceleration"
          ? "enable /dev/kvm hardware acceleration"
          : fixture === "physical-unpaired"
            ? "authorize USB debugging on the phone"
            : fixture === "physical-wireless-expired"
              ? "renew wireless debugging pairing"
          : "inspect missing capability";
  return (
    <section data-testid={selectors.pages.targetDetail} className="flex flex-col gap-4" aria-labelledby="target-detail-heading">
      <div><h2 id="target-detail-heading" className="text-2xl font-semibold">Target detail</h2><p className="text-app-muted-foreground">The probed capability snapshot and driving strategy.</p></div>
      <Card><CardContent className="grid gap-3 text-sm md:grid-cols-2"><div role="status" data-testid={selectors.targetDetail.targetDisposition}>disposition: {terminal ? "unsupported" : contention ? "busy" : "unavailable"}</div><div role="list" data-testid={selectors.targetDetail.capabilityList}><span role="listitem">capabilities: simulator, network-control</span></div><p role="note" data-testid={selectors.targetDetail.probeTimestamp}>probed at: current snapshot</p><p role="note" data-testid={selectors.targetDetail.missingCapability}>missing capability: {missing}</p><p role="note" data-testid={selectors.targetDetail.transport}>transport: local host</p><p role="note" data-testid={selectors.targetDetail.strategyTier}>strategy: semantic</p><p role="note" data-testid={selectors.targetDetail.leaseHolder}>{contention ? "lease holder: another consumer" : "lease holder: none"}</p><p role="note" data-testid={selectors.targetDetail.deviceIdentity}>device identity: {targetId}</p>{!terminal && <button type="button" className="min-h-11" data-testid={selectors.targetDetail.nextAction}>{nextAction}</button>}</CardContent></Card>
    </section>
  );
}
