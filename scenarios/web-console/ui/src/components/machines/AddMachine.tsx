import { useEffect, useState, type JSX } from "react";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Check, Copy, Loader2, Radio, Server, Wifi } from "lucide-react";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import { machineTestID } from "./testids";
import type { IssuedCode, JoinRequest } from "../../api/machines";
import { humanAge, humanCountdown } from "./age";

/**
 * Screen 02 — adding a machine.
 *
 * Three doors, ordered by how likely they are. Most people are adding a laptop
 * on the same network, and that case should need no address and no typing: the
 * machine asks, and the owner approves. The code is the fallback that works
 * from anywhere, and a server over SSH is the control plane's job.
 */

type Door = "network" | "code" | "server";

interface AddMachineProps {
  requests: JoinRequest[];
  code: IssuedCode | null;
  issuing: boolean;
  onIssueCode: () => void;
  onReview: (request: JoinRequest) => void;
  onBack: () => void;
  controlPlaneEndpoint: string;
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

export default function AddMachine({ requests, code, issuing, onIssueCode, onReview, onBack, controlPlaneEndpoint }: AddMachineProps) {
  const { t } = useTranslation();
  const [door, setDoor] = useState<Door>("network");

  const doors: { id: Door; label: string }[] = [
    { id: "network", label: t(strings.machines.doorNetwork) },
    { id: "code", label: t(strings.machines.doorCode) },
    { id: "server", label: t(strings.machines.doorServer) },
  ];

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mx-auto flex w-full max-w-4xl items-center justify-between gap-3 px-5 pt-5">
        <div className="flex items-center gap-2">
          <button
            type="button"
            data-testid="machines-add-back"
            onClick={onBack}
            aria-label={t(strings.machines.back)}
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
          </button>
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
          <div className="space-y-4">
            <p className="text-sm leading-6 text-wc-text-muted">{t(strings.machines.serverBody)}</p>
            {/* Honest handoff: a resumable, step-evented SSH onboarding already
                exists in the control plane. Reimplementing it here would be a
                second copy of the hardest flow in the fleet, so this door names
                the tool that owns it rather than pretending to replace it. */}
            <div className="rounded-xl border border-wc-default bg-wc-surface-base/40 p-4">
              <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
                {t(strings.machines.serverCommand)}
              </div>
              <code
                data-testid="machines-server-command"
                className="mt-2 block select-all overflow-x-auto rounded-lg border border-wc-default bg-wc-surface-base px-3 py-2 font-mono text-xs text-wc-text-secondary"
              >
                vrooli-bridge onboard start --host &lt;address&gt; --user &lt;login&gt;
              </code>
            </div>
            {controlPlaneEndpoint && (
              <a
                href={controlPlaneEndpoint}
                target="_blank"
                rel="noreferrer"
                data-testid="machines-open-bridge"
                className="inline-flex min-h-11 items-center rounded-full border border-wc-default px-4 text-sm font-medium text-wc-text-secondary transition hover:border-wc-accent hover:text-wc-text-primary"
              >
                {t(strings.machines.serverOpenBridge)}
              </a>
            )}
          </div>
        )}
        </div>
      </div>
    </div>
  );
}
