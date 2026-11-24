import { useState, type FormEvent } from "react";
import { Code, ConnectError } from "@connectrpc/connect";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { conversationsClient } from "../../api/notifications";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/** A decision-bound alternative to the normal verified-identity answer path.
 * Secrets and the MFA challenge exist only in this mounted component. */
export function RootPasswordAnswer({ ask, email: initialEmail, onCancel, onRecorded, onChanged, onUncertain }: {
  ask: Ask; email: string; onCancel: () => void; onRecorded: () => void; onChanged: () => void; onUncertain?: () => void;
}) {
  const { t } = useTranslation();
  const [answer, setAnswer] = useState("");
  const [email, setEmail] = useState(initialEmail);
  const [password, setPassword] = useState("");
  const [challenge, setChallenge] = useState("");
  const [code, setCode] = useState("");
  const [recovery, setRecovery] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [uncertain, setUncertain] = useState(false);
  const [recorded, setRecorded] = useState(false);

  const check = async () => {
    setBusy(true);
    try {
      const current = (await conversationsClient.getAsk({ askId: ask.id })).ask;
      if (current?.state === "answered" && current.answer === answer) {
        setRecorded(true); onRecorded();
      } else if (current && (current.state === "pending" || current.state === "escalated") && current.revisionCount === ask.revisionCount) {
        setUncertain(false); setError(t(strings.packet.rootPassword.open));
      } else {
        setError(t(strings.packet.rootPassword.changed));
      }
    } catch {
      setError(t(strings.packet.rootPassword.checkFailed));
    } finally { setBusy(false); }
  };
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!window.isSecureContext || busy || uncertain || !answer || !email.trim() || !password || (challenge && !code.trim())) return;
    setBusy(true); setError("");
    try {
      const result = await conversationsClient.answerRootWithPassword({ askId: ask.id, revisionCount: ask.revisionCount, answer, email: email.trim(), password, mfaChallenge: challenge, totpCode: recovery ? "" : code.trim(), recoveryCode: recovery ? code.trim() : "" });
      setCode("");
      if (result.mfaRequired && result.mfaChallenge) {
        setChallenge(result.mfaChallenge);
      } else if (result.answeredAt) {
        setPassword(""); setChallenge(""); setRecorded(true); onRecorded();
      } else {
        setPassword(""); setChallenge(""); setUncertain(true);
        setError(t(strings.packet.rootPassword.unclear)); onUncertain?.();
      }
    } catch (cause) {
      setCode("");
      if (cause instanceof ConnectError && cause.code === Code.Unauthenticated) {
        setPassword(""); setChallenge(""); setError(t(strings.packet.rootPassword.invalid));
      } else if (cause instanceof ConnectError && cause.code === Code.FailedPrecondition) {
        setPassword(""); setChallenge(""); setError(t(strings.packet.rootPassword.unavailable)); onChanged();
      } else {
        setPassword(""); setChallenge(""); setUncertain(true);
        setError(t(strings.packet.rootPassword.uncertain)); onUncertain?.();
      }
    } finally { setBusy(false); }
  };
  if (!window.isSecureContext) return <div className="flex flex-col gap-3"><p role="alert">{t(strings.session.secureRequired)}</p><Button type="button" variant="outline" onClick={onCancel}>{t(strings.packet.rootPassword.cancel)}</Button></div>;
  if (recorded) return <p role="status">{t(strings.packet.rootPassword.recorded)}</p>;
  return (
    <form onSubmit={(event) => void submit(event)} className="flex flex-col gap-4 rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="root-password-heading">
      <h2 id="root-password-heading" className="text-lg font-semibold">{t(strings.packet.rootPassword.title)}</h2>
      <p className="text-sm text-app-muted-foreground">{t(strings.packet.rootPassword.intro)}</p>
      <fieldset disabled={busy || uncertain || Boolean(challenge)} className="flex flex-col gap-2">
        <legend className="font-medium">{t(strings.packet.rootPassword.choice)}</legend>
        {ask.options.filter((option) => option.key === "approve" || option.key === "reject").map((option) => <label key={option.key} className="flex items-start gap-2"><input type="radio" name="root-answer" value={option.key} checked={answer === option.key} onChange={() => setAnswer(option.key)} />{option.label}</label>)}
      </fieldset>
      <label htmlFor="root-email" className="flex flex-col gap-1 text-sm">{t(strings.packet.rootPassword.email)}<Input id="root-email" type="email" autoComplete="username" value={email} disabled={busy || uncertain || Boolean(challenge)} onChange={(event) => setEmail(event.target.value)} required /></label>
      <label htmlFor="root-password" className="flex flex-col gap-1 text-sm">{t(strings.packet.rootPassword.password)}<Input id="root-password" type="password" autoComplete="current-password" value={password} disabled={busy || uncertain} onChange={(event) => setPassword(event.target.value)} required /></label>
      {challenge && <>
        <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={recovery} onChange={(event) => { setRecovery(event.target.checked); setCode(""); }} />{t(strings.packet.rootPassword.recovery)}</label>
        <label className="flex flex-col gap-1 text-sm">{recovery ? t(strings.packet.rootPassword.recoveryCode) : t(strings.packet.rootPassword.authenticatorCode)}<Input autoComplete="one-time-code" value={code} disabled={busy || uncertain} onChange={(event) => setCode(event.target.value)} required /></label>
      </>}
      {error && <p role="alert" className="text-sm text-app-danger">{error}</p>}
      {uncertain ? <Button type="button" disabled={busy} onClick={() => void check()}>{t(strings.packet.rootPassword.check)}</Button> : <Button type="submit" variant="primary" disabled={busy || !answer || !email.trim() || !password || Boolean(challenge && !code.trim())}>{busy ? t(strings.packet.rootPassword.checking) : challenge ? t(strings.packet.rootPassword.verifyCode) : t(strings.packet.rootPassword.verify)}</Button>}
      <Button type="button" variant="outline" disabled={busy} onClick={onCancel}>{t(strings.packet.rootPassword.cancel)}</Button>
    </form>
  );
}
