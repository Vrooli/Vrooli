import { useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/** Connect an existing authenticator account to the owner identified by Access. */
export function RootEnrollment() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [challenge, setChallenge] = useState("");
  const [code, setCode] = useState("");
  const [recovery, setRecovery] = useState(false);
  const [busy, setBusy] = useState(false);
  const [confirmRevoke, setConfirmRevoke] = useState(false);
  const [revokeResult, setRevokeResult] = useState<"success" | "error" | null>(null);
  const [result, setResult] = useState<"success" | "error" | null>(null);
  const clear = () => { setEmail(""); setPassword(""); setChallenge(""); setCode(""); setRecovery(false); };
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!window.isSecureContext || busy || !email.trim() || !password || (challenge && !code.trim())) return;
    setBusy(true); setResult(null);
    try {
      const response = await fetch("/api/v1/identity/root-enrollment", {
        method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: email.trim(), password, mfaChallenge: challenge, totpCode: recovery ? "" : code.trim(), recoveryCode: recovery ? code.trim() : "" }),
      });
      if (!response.ok) throw new Error("enrollment failed");
      const data: unknown = await response.json();
      if (!data || typeof data !== "object" || !("mfaRequired" in data) || typeof data.mfaRequired !== "boolean") throw new Error("invalid response");
      if (data.mfaRequired) {
        if (!("mfaChallenge" in data) || typeof data.mfaChallenge !== "string" || !data.mfaChallenge) throw new Error("invalid challenge");
        setChallenge(data.mfaChallenge); setCode("");
      } else {
        clear(); setOpen(false); setResult("success");
        await queryClient.invalidateQueries();
      }
    } catch {
      setChallenge(""); setCode(""); setResult("error");
    } finally { setBusy(false); }
  }
  async function revoke() {
    if (!window.isSecureContext || busy) return;
    setBusy(true); setRevokeResult(null);
    try {
      const response = await fetch("/api/v1/identity/root-enrollment", { method: "DELETE", credentials: "same-origin" });
      if (!response.ok) throw new Error("revocation failed");
      setConfirmRevoke(false); setRevokeResult("success");
      void queryClient.invalidateQueries();
    } catch { setConfirmRevoke(false); setRevokeResult("error"); }
    finally { setBusy(false); }
  }
  return <section className="flex flex-col items-start gap-3 border-t border-app-border pt-4">
    <h2 className="font-semibold">{t(strings.session.enrollmentTitle)}</h2>
    <p className="text-sm text-app-muted-foreground">{t(strings.session.enrollmentIntro)}</p>
    {!window.isSecureContext ? <p role="alert">{t(strings.session.secureRequired)}</p> : !open ? <Button type="button" variant="outline" onClick={() => { setResult(null); setOpen(true); }}>{t(strings.session.enrollmentAction)}</Button> :
      <form onSubmit={(event) => { void submit(event); }} className="flex w-full max-w-sm flex-col gap-3">
        <label className="flex flex-col gap-1 text-sm">{t(strings.session.email)}<Input type="email" autoComplete="username" value={email} onChange={(event) => setEmail(event.target.value)} required /></label>
        <label className="flex flex-col gap-1 text-sm">{t(strings.session.password)}<Input type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required /></label>
        {challenge && <><label className="flex flex-col gap-1 text-sm">{t(recovery ? strings.session.recoveryCode : strings.session.totpCode)}<Input autoComplete="one-time-code" value={code} onChange={(event) => setCode(event.target.value)} required /></label><button type="button" onClick={() => { setRecovery(!recovery); setCode(""); }}>{t(recovery ? strings.session.useTotp : strings.session.useRecovery)}</button></>}
        <Button type="submit" variant="primary" disabled={busy}>{t(strings.session.enrollmentAction)}</Button>
        <Button type="button" variant="outline" disabled={busy} onClick={() => { clear(); setOpen(false); setResult(null); }}>{t(strings.common.close)}</Button>
      </form>}
    {window.isSecureContext && <>
      <Button type="button" variant="outline" disabled={busy} onClick={() => { setRevokeResult(null); setConfirmRevoke(true); }}>{t(strings.session.revokeAction)}</Button>
      <AlertDialog open={confirmRevoke} title={t(strings.session.revokeTitle)} description={t(strings.session.revokeDescription)} destructive busy={busy} confirmLabel={t(strings.session.revokeAction)} cancelLabel={t(strings.common.close)} closeLabel={t(strings.common.close)} onConfirm={() => { void revoke(); }} onCancel={() => setConfirmRevoke(false)} />
    </>}
    {revokeResult && <p role={revokeResult === "error" ? "alert" : "status"}>{t(revokeResult === "error" ? strings.session.revokeError : strings.session.revokeSuccess)}</p>}
    {result && <p role={result === "error" ? "alert" : "status"}>{t(result === "error" ? strings.session.enrollmentError : strings.session.enrollmentSuccess)}</p>}
  </section>;
}
