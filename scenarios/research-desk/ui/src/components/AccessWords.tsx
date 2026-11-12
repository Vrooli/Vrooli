import { webConsoleAccess } from "../api/launch";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const WORDS = {
  credential: strings.access.credential,
  availability: strings.access.availability,
  unknown: strings.access.unknown,
} as const;

/**
 * How the desk reaches Web Console to start and steer sessions, in plain
 * words (rulings F14, V8): with its own credential, or through Web Console's
 * local availability policy, which lets any local program act.
 */
export function AccessWords({ said }: { said: string }) {
  const { t } = useTranslation();
  return <span className="text-body-sm">{t(WORDS[webConsoleAccess(said)], { said })}</span>;
}
