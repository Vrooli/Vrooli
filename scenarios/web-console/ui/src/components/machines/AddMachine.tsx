import { useEffect, useState, type JSX } from "react";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Check, Copy, Loader2, Radio, Server, Wifi } from "lucide-react";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import { machineTestID } from "./testids";
import type { IssuedCode, JoinRequest } from "../../api/machines";
import { getOnboarding, preflightOnboarding, startOnboarding } from "../../api/machines";
import { OnboardingState, type OnboardingStepEvent } from "@vrooli/proto-types/vrooli-bridge/v1/onboard/onboard_pb";
import { humanAge, humanCountdown } from "./age";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import PasswordInput from "@vrooli/react-component-library/PasswordInput/1";
import FormWizard from "@vrooli/react-component-library/FormWizard/1";

/**
 * Screen 02 — adding a machine.
 *
 * Three doors, ordered by how likely they are. Most people are adding a laptop
 * on the same network, and that case should need no address and no typing: the
 * machine asks, and the owner approves. The code is the fallback that works
 * from anywhere, and a server over SSH is the control plane's job.
 */

type Door = "network" | "code" | "server";

const setupPresets = [
  { id: "minimal", title: "Minimal connected node", summary: "Bridge agent only; no resources or scenarios are installed." },
  { id: "production", title: "Managed production node", summary: "Service posture with guarded updates and the production setup profile." },
  { id: "development", title: "Development node", summary: "Interactive development posture with the development toolchain." },
] as const;

interface AddMachineProps {
  requests: JoinRequest[];
  code: IssuedCode | null;
  issuing: boolean;
  onIssueCode: () => void;
  onReview: (request: JoinRequest) => void;
  onBack: () => void;
  /**
   * The control plane's own interface, resolved by the server against this
   * browser's origin. Empty when it could not be located — the handoff link is
   * then not rendered at all, because a link that cannot work is worse than an
   * absent one.
   */
  controlPlaneConsoleUrl: string;
  onOnboardingFinished?: (nodeID: string) => void;
}

const doorIcon: Record<Door, JSX.Element> = {
  network: <Wifi className="h-4 w-4" aria-hidden />,
  code: <Radio className="h-4 w-4" aria-hidden />,
  server: <Server className="h-4 w-4" aria-hidden />,
};

/** Group a long high-entropy code so a person can read it back a chunk at a time. */
function groupCode(code: string): string {
  return (code.match(/.{1,4}/g) ?? [code]).join(" ");
}

function JoinCode({ code, issuing, onIssueCode }: { code: IssuedCode | null; issuing: boolean; onIssueCode: () => void }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const [remaining, setRemaining] = useState(code?.expiresInSeconds ?? 0);

  useEffect(() => { setRemaining(code?.expiresInSeconds ?? 0); }, [code]);
  useEffect(() => {
    if (!code || remaining <= 0) return;
    const timer = window.setInterval(() => { setRemaining((value) => Math.max(0, value - 1)); }, 1000);
    return () => { window.clearInterval(timer); };
  }, [code, remaining]);
  useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => { setCopied(false); }, 2000);
    return () => { window.clearTimeout(timer); };
  }, [copied]);

  if (!code) {
    return (
      <Button variant="outline" size="sm" data-testid="machines-issue-code" onClick={onIssueCode} disabled={issuing}>
        {issuing ? <Loader2 className="me-1.5 h-4 w-4 animate-spin" aria-hidden /> : null}
        {issuing ? t(strings.machines.issuingCode) : t(strings.machines.issueCode)}
      </Button>
    );
  }

  const expired = remaining <= 0;
  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-3">
        <code
          data-testid="machines-join-code"
          className={`select-all break-all rounded-lg border px-3 py-2 font-mono text-sm tracking-[0.12em] ${expired ? "border-wc-default bg-wc-surface-base/60 text-wc-text-faint line-through" : "border-wc-accent/40 bg-wc-accent/10 text-wc-text-primary"}`}
        >
          {groupCode(code.code)}
        </code>
        <button
          type="button"
          data-testid="machines-copy-code"
          onClick={() => {
            // Clipboard access is absent in insecure contexts, so the button
            // must simply do nothing rather than throw.
            if (typeof navigator.clipboard === "undefined") return;
            void navigator.clipboard.writeText(code.code).then(() => { setCopied(true); }).catch(() => { setCopied(false); });
          }}
          className="inline-flex min-h-11 items-center gap-1.5 rounded-lg border border-wc-default px-3 text-xs font-medium text-wc-text-secondary transition hover:border-wc-accent hover:text-wc-text-primary"
        >
          {copied ? <Check className="h-3.5 w-3.5" aria-hidden /> : <Copy className="h-3.5 w-3.5" aria-hidden />}
          {copied ? t(strings.machines.copied) : t(strings.machines.copy)}
        </button>
      </div>
      <p className="text-xs text-wc-text-faint">
        {expired ? t(strings.machines.codeExpired) : t(strings.machines.codeSingleUse, { expiry: humanCountdown(remaining) })}
      </p>
    </div>
  );
}

function ListeningList({ requests, onReview }: { requests: JoinRequest[]; onReview: (request: JoinRequest) => void }) {
  const { t } = useTranslation();
  return (
    <section className="space-y-2">
      <h4 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
        {t(strings.machines.listening)}
      </h4>
      {requests.length === 0 ? (
        <p data-testid="machines-listening-empty" className="rounded-xl border border-dashed border-wc-default bg-wc-surface-base/40 px-4 py-5 text-center text-xs leading-5 text-wc-text-faint">
          {t(strings.machines.listeningEmpty)}
        </p>
      ) : (
        <ul className="space-y-2">
          {requests.map((request) => (
            <li
              key={request.id}
              data-testid={`machines-listening-${machineTestID(request.id)}`}
              className="flex flex-wrap items-center gap-3 rounded-xl border border-wc-default bg-wc-surface-input px-4 py-3"
            >
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium text-wc-text-primary">{request.name}</span>
                <span className="mt-0.5 block truncate text-xs text-wc-text-faint">
                  {[request.os, request.arch, request.endpoint, t(strings.machines.askedToJoin, { age: humanAge(request.requestedAgeSeconds) })]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
              </span>
              <Button size="sm" data-testid={`machines-listening-review-${machineTestID(request.id)}`} onClick={() => { onReview(request); }}>
                {t(strings.machines.review)}
              </Button>
            </li>
          ))}
        </ul>
      )}
      <p className="text-xs leading-5 text-wc-text-faint">{t(strings.machines.onlyYourAccount)}</p>
    </section>
  );
}

async function watchOnboarding(opID: string, onUpdate: (events: OnboardingStepEvent[], state: string) => void) {
  const deadline = Date.now() + 5 * 60 * 1000;
  let lastSignature = "";
  while (Date.now() < deadline) {
    const status = await getOnboarding(opID);
    const events = status.events ?? [];
    const rawState = status.op?.state;
    const state = typeof rawState === "number" ? OnboardingState[rawState] ?? String(rawState) : String(rawState ?? "");
    const signature = `${state}:${events.length}:${events.at(-1)?.stepId ?? ""}:${events.at(-1)?.status ?? ""}`;
    if (signature !== lastSignature) {
      lastSignature = signature;
      onUpdate(events, state || "started");
    }
    if (["SUCCEEDED", "FAILED", "CANCELLED", "ONBOARDING_STATE_SUCCEEDED", "ONBOARDING_STATE_FAILED", "ONBOARDING_STATE_CANCELLED"].includes(state)) {
      const failure = status.op?.failureReason ? `: ${status.op.failureReason}` : "";
      const detail = events.at(-1)?.detail ? ` — ${events.at(-1)?.detail}` : "";
      onUpdate(events, `${state}${failure}${detail}`);
      return status.op?.nodeId ?? "";
    }
    await new Promise((resolve) => window.setTimeout(resolve, 1000));
  }
  throw new Error("Onboarding is still running after the five-minute display deadline. Reopen this machine to resume from its durable step history.");
}

export default function AddMachine({ requests, code, issuing, onIssueCode, onReview, onBack, onOnboardingFinished }: AddMachineProps) {
  const { t } = useTranslation();
  const [door, setDoor] = useState<Door>("network");
  const [host, setHost] = useState("");
  const [port, setPort] = useState("22");
  const [user, setUser] = useState("");
  const [password, setPassword] = useState("");
  const [machineID, setMachineID] = useState("");
  const [setupPreset, setSetupPreset] = useState("minimal");
  const [onboardingMessage, setOnboardingMessage] = useState("");
  const [onboardingBusy, setOnboardingBusy] = useState(false);
  const [onboardingEvents, setOnboardingEvents] = useState<OnboardingStepEvent[]>([]);

  const doors: { id: Door; label: string }[] = [
    { id: "network", label: t(strings.machines.doorNetwork) },
    { id: "code", label: t(strings.machines.doorCode) },
    { id: "server", label: t(strings.machines.doorServer) },
  ];

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mx-auto flex w-full max-w-4xl items-center justify-between gap-3 px-5 pt-5">
        <div className="flex items-center gap-2">
          <IconButton
            data-testid="machines-add-back"
            onClick={onBack}
            aria-label={t(strings.machines.back)}
            shape="rounded"
          >
            <ArrowLeft aria-hidden />
          </IconButton>
          <h2 className="text-lg font-semibold text-wc-text-primary">{t(strings.machines.addTitle)}</h2>
        </div>
      </div>

      <div role="tablist" aria-label={t(strings.machines.addTitle)} className="mx-auto mt-4 flex w-full max-w-4xl gap-1 px-5">
        {doors.map((entry) => (
          <button
            key={entry.id}
            role="tab"
            type="button"
            aria-selected={door === entry.id}
            data-testid={`machines-door-${entry.id}`}
            onClick={() => { setDoor(entry.id); }}
            className={`inline-flex min-h-11 flex-1 items-center justify-center gap-1.5 rounded-lg border px-3 text-xs font-medium transition ${
              door === entry.id
                ? "border-wc-accent bg-wc-accent/10 text-wc-text-primary"
                : "border-wc-default bg-wc-surface-input text-wc-text-secondary hover:border-wc-accent/50"
            }`}
          >
            {doorIcon[entry.id]}
            <span className="truncate">{entry.label}</span>
          </button>
        ))}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-4xl space-y-5 px-5 pb-5 pt-5">
        {door === "network" && (
          <>
            <p className="text-sm leading-6 text-wc-text-muted">{t(strings.machines.networkBody)}</p>
            <div className="rounded-xl border border-wc-default bg-wc-surface-base/40 p-4">
              <h4 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
                {t(strings.machines.networkFallback)}
              </h4>
              <p className="mb-3 mt-1 text-xs leading-5 text-wc-text-muted">{t(strings.machines.networkFallbackBody)}</p>
              <JoinCode code={code} issuing={issuing} onIssueCode={onIssueCode} />
            </div>
            <ListeningList requests={requests} onReview={onReview} />
          </>
        )}

        {door === "code" && (
          <>
            <p className="text-sm leading-6 text-wc-text-muted">{t(strings.machines.networkFallbackBody)}</p>
            <JoinCode code={code} issuing={issuing} onIssueCode={onIssueCode} />
            <ListeningList requests={requests} onReview={onReview} />
          </>
        )}

        {door === "server" && (
          <FormWizard draftKey="machines-onboard-connection" steps={[{ id: "connection", title: "Connection details", content: (
          <div className="space-y-4">
            <p className="text-sm leading-6 text-wc-text-muted">{t(strings.machines.serverBody)}</p>
            <div className="rounded-xl border border-wc-default bg-wc-surface-base/40 p-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="grid gap-1 text-xs">Address<input data-testid="machines-onboard-host" value={host} onChange={(event) => { setHost(event.target.value); }} className="rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 text-sm" placeholder="server.example" /></label>
                <label className="grid gap-1 text-xs">Port<input data-testid="machines-onboard-port" type="number" min="1" max="65535" value={port} onChange={(event) => { setPort(event.target.value); }} className="rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 text-sm" /></label>
                <label className="grid gap-1 text-xs">Login<input data-testid="machines-onboard-user" value={user} onChange={(event) => { setUser(event.target.value); }} className="rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 text-sm" placeholder="root" /></label>
                <PasswordInput name="machines-onboard-password" label="Password" autoComplete="current-password" value={password} onChange={setPassword} className="text-xs sm:col-span-2" />
                <label className="grid gap-1 text-xs sm:col-span-2">Setup preset<select data-testid="machines-onboard-preset" value={setupPreset} onChange={(event) => { setSetupPreset(event.target.value); }} className="rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 text-sm">{setupPresets.map((preset) => <option key={preset.id} value={preset.id}>{preset.title}</option>)}</select><span className="text-xs text-wc-text-faint">{setupPresets.find((preset) => preset.id === setupPreset)?.summary}</span></label>
              </div>
              <Button className="mt-4" data-testid="machines-onboard-start" disabled={onboardingBusy || !host.trim() || !user.trim()} onClick={() => { setOnboardingBusy(true); setOnboardingEvents([]); setOnboardingMessage("Checking SSH target…"); const sshPort = Number(port); void preflightOnboarding({ host, port: sshPort, user, machineId: machineID }).then((result) => { if (result.passwordRequired && !password) throw new Error("This target requires the SSH password."); if (!result.machineId) throw new Error("The control plane did not return a machine identity for this target."); setMachineID(result.machineId); return startOnboarding({ machineId: result.machineId, host, port: sshPort, user, sshPassword: password, setupPreset }); }).then(async (result) => { setOnboardingMessage(`Onboarding started: ${result.opId}. Waiting for named steps…`); const nodeID = await watchOnboarding(result.opId, (events, state) => { setOnboardingEvents(events); const latest = events.at(-1); setOnboardingMessage(state.startsWith("ONBOARDING_STATE_") ? `Onboarding ${result.opId}: ${state}` : latest ? `Onboarding ${result.opId}: ${latest.stepId} · ${latest.status}` : `Onboarding ${result.opId}: ${state}`); }); if (nodeID) onOnboardingFinished?.(nodeID); }).catch((error: unknown) => { setOnboardingMessage(error instanceof Error ? error.message : "Onboarding could not start."); }).finally(() => { setOnboardingBusy(false); }); }}>{onboardingBusy ? "Onboarding…" : "Start onboarding"}</Button>
              {onboardingMessage && <p className="mt-3 text-xs text-wc-text-muted" role="status">{onboardingMessage}</p>}
              {onboardingEvents.length > 0 && <ol className="mt-3 space-y-1" data-testid="machines-onboard-progress" aria-label="Onboarding progress">{onboardingEvents.map((event, index) => <li key={`${event.stepId}-${index}`} className="flex items-center justify-between gap-3 rounded border border-wc-default px-2 py-1 text-xs"><span>{event.stepId}</span><span className="text-wc-text-muted">{event.status}</span></li>)}</ol>}
            </div>
          </div>
          ) }]} />
        )}
        </div>
      </div>
    </div>
  );
}
