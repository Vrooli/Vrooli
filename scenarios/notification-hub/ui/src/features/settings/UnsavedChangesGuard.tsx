import { useEffect } from "react";
import { useBlocker } from "react-router-dom";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Asks before a draft is lost to navigation: a link, the back button or a
 * reload. Mount it next to any editor whose state lives only on the page.
 */
export function UnsavedChangesGuard({ when }: { when: boolean }) {
  const { t } = useTranslation();
  const blocker = useBlocker(({ currentLocation, nextLocation }) => when && currentLocation.pathname !== nextLocation.pathname);
  useEffect(() => {
    if (!when) return undefined;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [when]);
  return (
    <AlertDialog
      open={blocker.state === "blocked"}
      title={t(strings.settings.unsaved.title)}
      description={t(strings.settings.unsaved.body)}
      destructive
      confirmLabel={t(strings.settings.unsaved.leave)}
      cancelLabel={t(strings.settings.unsaved.stay)}
      closeLabel={t(strings.settings.unsaved.stay)}
      onConfirm={() => blocker.proceed?.()}
      onCancel={() => blocker.reset?.()}
    />
  );
}
