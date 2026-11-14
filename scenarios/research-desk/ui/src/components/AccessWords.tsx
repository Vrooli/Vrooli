import { WebConsoleAccess } from "../api/activity";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const WORDS = {
  [WebConsoleAccess.SERVICE_CREDENTIAL]: strings.access.credential,
  [WebConsoleAccess.AVAILABILITY_POLICY]: strings.access.availability,
  [WebConsoleAccess.UNAVAILABLE]: strings.access.unavailable,
  [WebConsoleAccess.UNSPECIFIED]: strings.access.unknown,
} as const;

/**
 * How the desk reaches Web Console to start and steer sessions, in plain
 * words: with its own credential, through Web Console's local availability
 * policy (which lets any local program act), or not at all. `said` is the
 * API's own sentence, shown only when it reports no kind.
 */
export function AccessWords({ kind, said }: { kind: WebConsoleAccess; said: string }) {
  const { t } = useTranslation();
  return <span className="text-body-sm">{t(WORDS[kind], { said })}</span>;
}
