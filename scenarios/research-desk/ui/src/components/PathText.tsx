import { FilePath } from "@vrooli/react-component-library/FilePath/1.0.6";

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
    <FilePath
      path={path}
      presentation="inline"
      displayText={shown}
      href={href ?? undefined}
      showCopyButton
      copyLabel={t(strings.common.copyPath)}
      copiedLabel={t(strings.common.copied)}
      className="rd-path"
    />
  );
}
