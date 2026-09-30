import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

function fixtureState(): string {
  return new URLSearchParams(window.location.search).get("fixture") ?? "default";
}

export function DeliveryOverview() {
  const { t } = useTranslation();
    const fixture = fixtureState();
  const terminal = fixture === "terminal-and-blocked" || fixture === "no-mac-node";
  const mirrorPassed = fixture === "mirror-passed";
  const evidence = fixture === "green-pixel" || mirrorPassed ? strings.experience.pixelGradeMirror : strings.experience.semantic;
  const disposition = terminal ? strings.experience.unsupported : mirrorPassed ? strings.experience.available : fixture === "validate-only" ? strings.experience.validateOnly : strings.experience.unavailable;
  return (
    <Card className="lg:col-span-3">
      <CardHeader><CardTitle>{t(strings.experience.deliveryEvidence)}</CardTitle></CardHeader>
      <CardContent className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
        <div role="table" data-testid={selectors.delivery.targetMatrix}><div role="rowgroup"><div role="row"><div role="cell"><strong>{t(strings.experience.targetMatrix)}</strong><p>{t(terminal ? strings.experience.terminal : strings.experience.probed)}</p></div></div></div></div>
        <div role="status" data-testid={selectors.delivery.targetDisposition}><strong>{t(strings.experience.targetDisposition)}</strong><p>{t(disposition)}</p></div>
        <div role="status" data-testid={selectors.delivery.gateVerdict}><strong>{t(strings.experience.releaseGate)}</strong><p>{t(terminal ? strings.experience.pending : mirrorPassed ? strings.experience.passedNonPromotable : strings.experience.pending)}</p></div>
        <div role="status" data-testid={selectors.delivery.readinessSummary}><strong>{t(strings.experience.releaseReadiness)}</strong><p>{t(fixture === "no-mac-node" ? strings.experience.blockedOnMacOSBridge : strings.experience.appleHardwareRequired)}</p></div>
        <div role="note" data-testid={selectors.delivery.executingNode}><strong>{t(strings.experience.executingNode)}</strong><p>{t(strings.experience.linuxHost)}</p></div>
        <div role="status" data-testid={selectors.delivery.rowPromotability}><strong>{t(strings.experience.evidenceGrade)}</strong><p>{t(evidence)}</p></div>
        <button type="button" className="min-h-11" data-testid={selectors.delivery.generateProject}>{t(strings.experience.generateProject)}</button>
      </CardContent>
    </Card>
  );
}

export function DistributionSurface() {
  const { t } = useTranslation();
    const fixture = fixtureState();
  const waiting = fixture === "verification-pending" || fixture === "review-pending";
  const noMacNode = fixture === "no-mac-node";
  const noEnrollment = fixture === "no-enrollment";
  const testflightReady = fixture === "testflight-ready";
  const channels = [strings.experience.appStore, strings.experience.testFlight, strings.experience.adHoc];
  return (
    <section data-testid={selectors.pages.distribution} className="flex flex-col gap-4" aria-labelledby="distribution-heading">
      <div><h2 id="distribution-heading" className="text-2xl font-semibold">{t(strings.experience.distribution)}</h2><p className="text-app-muted-foreground">{t(strings.experience.independentAppleChannelVerdicts)}</p></div>
      <Card role="list" data-testid={selectors.distribution.channelList}><CardHeader role="listitem"><CardTitle>{t(strings.experience.channels)}</CardTitle></CardHeader><CardContent role="listitem" className="grid gap-3 text-sm md:grid-cols-3">
        {channels.map((channel) => {
          const available = testflightReady && channel === strings.experience.testFlight;
          const status = noMacNode ? strings.experience.blockedNoMacNode : waiting ? strings.experience.waitingOnApple : available ? strings.experience.available : strings.experience.unavailable;
          const requirement = noMacNode ? strings.experience.requiresReachableMacOSBridge : noEnrollment ? strings.experience.requiresAppleDeveloperEnrollment : testflightReady ? strings.experience.requiresChannelReview : strings.experience.requiresEnrollmentAndSigning;
          const blocking = noMacNode ? strings.experience.macOSBuildHost : noEnrollment ? strings.experience.developerProgram : waiting ? strings.experience.appleReview : testflightReady && !available ? strings.experience.appStoreReview : strings.experience.developerProgram;
          const nextAction = noMacNode ? strings.experience.connectMacOSBridge : waiting ? strings.experience.waitForAppleDecision : noEnrollment ? strings.experience.enrollDeveloperProgram : available ? strings.experience.manageTestFlightTesters : strings.experience.submitAppStoreReview;
          return <article role="listitem" key={channel}><div className="flex items-center justify-between gap-2"><strong role="status" data-testid={selectors.distribution.channelAvailability}>{t(channel)}: {t(status)}</strong><StatusBadge tone={available ? "success" : "warning"}>{t(status)}</StatusBadge></div><p role="note" data-testid={selectors.distribution.channelRequirement}>{t(requirement)}</p><p role="note" data-testid={selectors.distribution.channelBlockingRung}>{t(blocking)}</p><button type="button" className="min-h-11" data-testid={selectors.distribution.channelNextAction}>{t(nextAction)}</button></article>;
        })}
      </CardContent></Card>
      <Card role="note" data-testid={selectors.distribution.artifactIdentity}><CardContent>{t(strings.experience.artifactPending)}</CardContent></Card>
      {fixture === "non-promotable-artifact" && <p data-testid={selectors.distribution.promotabilityBlock} role="status">{t(strings.experience.evidenceCannotGateRelease)}</p>}
    </section>
  );
}

export function ReadinessSurface() {
  const { t } = useTranslation();
    const fixture = fixtureState();
  const waiting = fixture === "verification-pending" || fixture === "enrollment-pending";
  const noMacNode = fixture === "no-mac-node";
  const nothingStarted = fixture === "nothing-started";
  const rungs = [strings.experience.appleID, strings.experience.developerProgram, strings.experience.appStoreConnect, strings.experience.macOSBuildHost, strings.experience.macOSBuildHost, strings.experience.testFlight];
  return (
    <section data-testid={selectors.pages.readiness} className="flex flex-col gap-4" aria-labelledby="readiness-heading">
      <div><h2 id="readiness-heading" className="text-2xl font-semibold">{t(strings.experience.releaseReadinessHeading)}</h2><p className="text-app-muted-foreground">{t(strings.experience.orderedAppleLadder)}</p></div>
      <Card role="list" data-testid={selectors.readiness.rungList}><CardContent role="listitem" className="grid gap-3 md:grid-cols-2">{rungs.map((rung, index) => { const state = noMacNode ? strings.experience.blockedNoMacNode : waiting && index === 0 ? strings.experience.waitingOnApple : index === 0 ? strings.experience.available : waiting && index === 1 ? strings.experience.waitingOnApple : nothingStarted ? strings.experience.blockedOnAppleID : strings.experience.unavailable; const owner = noMacNode ? strings.experience.macOSBridge : index < 2 ? strings.experience.owner : strings.experience.ownerAndApple; const action = noMacNode ? strings.experience.connectMacOSBridge : waiting ? strings.experience.waitForAppleDecision : index === 0 ? strings.experience.signInAppleID : strings.experience.completePrecedingRung; return <article role="listitem" key={`${rung}-${index}`} className="border-b border-app-border pb-3"><div className="flex items-center justify-between gap-2"><strong role="status" data-testid={selectors.readiness.rungState}>{t(state)}</strong><span>{t(rung)}</span></div><p role="note" data-testid={selectors.readiness.rungOwner}>{t(owner)}</p><p role="note" data-testid={selectors.readiness.rungCost}>{t(index === 1 ? strings.experience.annualProgram : strings.experience.noAdditionalCost)}</p><button type="button" className="min-h-11" data-testid={selectors.readiness.rungNextAction}>{t(action)}</button>{index === 4 && <p role="note" data-testid={selectors.readiness.rungDeadline}>{t(strings.experience.hostLease)}</p>}{index === 1 && <p role="note" data-testid={selectors.readiness.mirrorPathNote}>{t(strings.experience.semanticMirror)}</p>}</article>; })}</CardContent></Card>
    </section>
  );
}

export function RunReviewSurface() {
  const { t } = useTranslation();
    const fixture = fixtureState();
  const routeState = window.location.pathname.split("/").pop() || "";
  const state = fixture !== "default" ? fixture : routeState === "runs" ? "default" : routeState;
  const interrupted = state === "lease-lost";
  const partial = state === "partial";
  const nonPromotable = state === "mirror-run" || state === "non-promotable" || fixture === "non-promotable";
  return (
    <section data-testid={selectors.pages.runReview} className="flex flex-col gap-4" aria-labelledby="run-review-heading">
      <div><h2 id="run-review-heading" className="text-2xl font-semibold">{t(strings.experience.runReview)}</h2><p className="text-app-muted-foreground">{t(strings.experience.expectedObservedEvidence)}</p></div>
      <Card role="list" data-testid={selectors.runReview.chapterList}><CardContent role="listitem" className="space-y-3 text-sm"><p role="status" data-testid={selectors.runReview.runVerdict}>{t(interrupted ? strings.experience.incomplete : partial ? strings.experience.partial : strings.experience.passed)}</p><div className="border-t border-app-border pt-3"><p role="note" data-testid={selectors.runReview.chapterExpected}>{t(strings.experience.applicationLaunches)}</p><p role="note" data-testid={selectors.runReview.chapterObserved}>{t(partial ? strings.experience.wkwebviewUnavailable : strings.experience.semanticObservation)}</p><p role="note" data-testid={selectors.runReview.chapterStrategy}>{t(strings.experience.strategyMirrorSemantic)}</p><p role="note" data-testid={selectors.runReview.exceededBound}>{t(strings.experience.settleBound)}</p></div>{nonPromotable || interrupted ? <div role="alert" data-testid={selectors.runReview.promotabilityNotice}>{t(strings.experience.cannotGateRelease)}</div> : null}{interrupted && <p data-testid={selectors.runReview.leaseInterruption} role="alert">{t(strings.experience.leaseLostInfrastructure)}</p>}<p role="region" data-testid={selectors.runReview.recording}>{t(strings.experience.recordingUnavailable)}</p></CardContent></Card>
    </section>
  );
}

export function TargetDetailSurface() {
  const { t } = useTranslation();
    const fixture = fixtureState();
  const terminal = fixture === "terminal-linux";
  const contention = fixture === "device-leased-elsewhere";
  const targetId = window.location.pathname.split("/").pop() || "target-local";
  const missing = fixture === "no-gui-session" ? strings.experience.loggedInGuiSession : fixture === "missing-runtime-variant" ? strings.experience.x86SimulatorRuntime : terminal ? strings.experience.appleToolchain : strings.experience.macOSBridgeNode;
  return (
    <section data-testid={selectors.pages.targetDetail} className="flex flex-col gap-4" aria-labelledby="target-detail-heading"><div><h2 id="target-detail-heading" className="text-2xl font-semibold">{t(strings.experience.targetDetail)}</h2><p className="text-app-muted-foreground">{t(strings.experience.probedCapabilitySnapshot)}</p></div><Card><CardContent className="grid gap-3 text-sm md:grid-cols-2"><div role="status" data-testid={selectors.targetDetail.targetDisposition}>{t(strings.experience.disposition)}: {t(terminal ? strings.experience.unsupported : contention ? strings.experience.blockedNoMacNode : strings.experience.unavailable)}</div><div role="list" data-testid={selectors.targetDetail.capabilityList}><span role="listitem">{t(strings.experience.capabilities)}</span></div><p role="note" data-testid={selectors.targetDetail.probeTimestamp}>{t(strings.experience.probedAt)}</p><p role="note" data-testid={selectors.targetDetail.missingCapability}>{t(strings.experience.missingCapability)}: {t(missing)}</p><p role="note" data-testid={selectors.targetDetail.transport}>{t(terminal ? strings.experience.localLinuxHost : strings.experience.macOSBridge)}</p><p role="note" data-testid={selectors.targetDetail.strategyTier}>{t(terminal ? strings.experience.strategyUnsupported : strings.experience.strategyMirrorSemantic)}</p><p role="note" data-testid={selectors.targetDetail.leaseHolder}>{t(contention ? strings.experience.leaseHolderOtherConsumer : strings.experience.leaseHolderNone)}</p><p role="note" data-testid={selectors.targetDetail.deviceIdentity}>{t(strings.experience.deviceIdentity)}: {targetId}</p>{!terminal && <button type="button" className="min-h-11" data-testid={selectors.targetDetail.nextAction}>{t(fixture === "no-gui-session" ? strings.experience.logIntoGuiSession : fixture === "missing-runtime-variant" ? strings.experience.installSimulatorRuntime : strings.experience.registerInspectBridge)}</button>}</CardContent></Card></section>
  );
}
