import { useEffect, useState } from "react";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { renderLegacyComposition } from "../workspaces/legacyCompositionPreview";

type PreviewState =
  | { status: "loading" }
  | { status: "ready"; html: string }
  | { status: "error"; message: string };

/** Read-only bridge for migrated legacy states. It never writes workspace data. */
export function LegacyCompositionPreview({ argsJson }: { argsJson: string }) {
  const { t } = useTranslation();
  const [state, setState] = useState<PreviewState>({ status: "loading" });

  useEffect(() => {
    let active = true;
    setState({ status: "loading" });
    renderLegacyComposition(argsJson).then(
      (html) => {
        if (active) setState({ status: "ready", html });
      },
      (error: unknown) => {
        if (active) {
          setState({
            status: "error",
            message:
              error instanceof Error ? error.message : t(strings.workbench.studio.legacyPreviewFailed),
          });
        }
      },
    );
    return () => {
      active = false;
    };
  }, [argsJson, t]);

  if (state.status === "loading") {
    return (
      <div role="status">
        {t("workbench.studio.legacyPreviewLoading", {
          defaultValue: "Rendering saved composition…",
        })}
      </div>
    );
  }
  if (state.status === "error") return <div role="alert">{state.message}</div>;
  return (
    <iframe
      title={t("workbench.studio.legacyPreviewTitle", {
        defaultValue: "Read-only composition preview",
      })}
      sandbox="allow-scripts"
      referrerPolicy="no-referrer"
      srcDoc={state.html}
      style={{ border: 0, width: "100%", height: "100%", background: "white" }}
    />
  );
}
