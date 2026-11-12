import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { Link } from "react-router-dom";

import { strings } from "../consts/strings";
import { useResearchHome } from "../hooks/useResearchHome";
import { useTranslation } from "../i18n";
import { filePageHref, shownPath } from "../lib/paths";

/**
 * A path that wraps on a phone. Inside the research home it opens the file
 * page; anywhere else it is text with a copy button, never a dead link.
 */
export function PathText({ path, label }: { path: string; label?: string }) {
  const { t } = useTranslation();
  const home = useResearchHome();
  const href = filePageHref(path, home);
  const shown = label ?? shownPath(path, home);
  return (
    <span className="inline-flex max-w-full items-start gap-space-3xs align-top">
      {href ? (
        <Link to={href} className="rd-path text-app-primary underline-offset-2 hover:underline">{shown}</Link>
      ) : (
        <span className="rd-path">{shown}</span>
      )}
      <CopyIconButton value={path} aria-label={t(strings.common.copyPath)} copiedLabel={t(strings.common.copied)} />
    </span>
  );
}
