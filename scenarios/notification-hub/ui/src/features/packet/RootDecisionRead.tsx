import { useState, type FormEvent } from "react";
import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { AskSchema, type Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/** Credentials and the challenge are held only while this form is mounted. */
export function RootDecisionRead({ askId, onRead }: { askId: string; onRead: (ask: Ask) => void }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [challenge, setChallenge] = useState("");
  const [code, setCode] = useState("");
  const [recovery, setRecovery] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!window.isSecureContext || busy || !email.trim() || !password || (challenge && !code.trim())) return;
    setBusy(true); setError(false);
    try {
      const response = await fetch("/api/v1/identity/root-decision-read", { method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ askId, email: email.trim(), password, mfaChallenge: challenge, totpCode: recovery ? "" : code.trim(), recoveryCode: recovery ? code.trim() : "" }) });
      if (!response.ok) throw new Error("read failed");
      const data: unknown = await response.json();
      if (!data || typeof data !== "object" || !("mfaRequired" in data) || typeof data.mfaRequired !== "boolean") throw new Error("invalid response");
      if (data.mfaRequired) {
        if (!("mfaChallenge" in data) || typeof data.mfaChallenge !== "string" || !data.mfaChallenge) throw new Error("invalid challenge");
        setChallenge(data.mfaChallenge); setCode("");
      } else {
        if (!("ask" in data) || !data.ask) throw new Error("missing decision");
        const ask = fromJson(AskSchema, data.ask as JsonValue);
        if (ask.id !== askId || ask.source !== "root-rebaseline" || ask.sourceEventType !== "root.rebaseline.decision_requested.v1") throw new Error("unexpected decision");
        setPassword(""); setChallenge(""); setCode(""); onRead(ask);
      }
    } catch { setPassword(""); setChallenge(""); setCode(""); setError(true); }
    finally { setBusy(false); }
  }
  if (!window.isSecureContext) return null;
  if (!open) return <Button type="button" variant="outline" onClick={() => setOpen(true)}>{t(strings.packet.rootRead.action)}</Button>;
  return <form onSubmit={(event) => { void submit(event); }} className="flex w-full max-w-sm flex-col gap-3">
    <h2 className="font-semibold">{t(strings.packet.rootRead.title)}</h2>
    <p>{t(strings.packet.rootRead.intro)}</p>
    <label className="flex flex-col gap-1 text-sm">{t(strings.session.email)}<Input type="email" autoComplete="username" disabled={busy || Boolean(challenge)} value={email} onChange={(event) => setEmail(event.target.value)} required /></label>
    <label className="flex flex-col gap-1 text-sm">{t(strings.session.password)}<Input type="password" autoComplete="current-password" disabled={busy} value={password} onChange={(event) => setPassword(event.target.value)} required /></label>
    {challenge && <><label className="flex flex-col gap-1 text-sm">{t(recovery ? strings.session.recoveryCode : strings.session.totpCode)}<Input autoComplete="one-time-code" disabled={busy} value={code} onChange={(event) => setCode(event.target.value)} required /></label><button type="button" onClick={() => { setRecovery(!recovery); setCode(""); }}>{t(recovery ? strings.session.useTotp : strings.session.useRecovery)}</button></>}
    {error && <p role="alert">{t(strings.packet.rootRead.error)}</p>}
    <Button type="submit" variant="primary" disabled={busy}>{t(strings.packet.rootRead.action)}</Button>
    <Button type="button" variant="outline" disabled={busy} onClick={() => { setOpen(false); setEmail(""); setPassword(""); setCode(""); setChallenge(""); setError(false); }}>{t(strings.common.close)}</Button>
  </form>;
}
