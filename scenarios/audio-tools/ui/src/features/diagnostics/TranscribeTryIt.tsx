import { Panel } from "../../components/ui/panel";
import { Tabs } from "../../components/ui/tabs";
import type { ProviderTrace } from "../../services/diagnostics";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { LiveTry } from "./LiveTry";
import { OneshotTry } from "./OneshotTry";
import { FileTry } from "./FileTry";

interface Props {
  onTrace: (t: ProviderTrace) => void;
}

// TranscribeTryIt is the per-capability "try it" card. Three modes live
// inside a Panel-scoped Tabs strip (modes are sub-features of one
// capability, not top-level navigation). Each mode is its own file so
// the live/oneshot/file paths can be tested in isolation.
export function TranscribeTryIt({ onTrace }: Props) {
  const { t } = useTranslation();
  return (
    <Panel
      title={t(strings.diagnostics.transcribeTitle)}
      description={t(strings.diagnostics.transcribeDescription)}
    >
      <Tabs
        ariaLabel={t(strings.diagnostics.tablistTranscribeMode)}
        items={[
          { value: "live", label: t(strings.diagnostics.tabLiveMic) },
          { value: "oneshot", label: t(strings.diagnostics.tabOneshot) },
          { value: "file", label: t(strings.diagnostics.tabFile) },
        ]}
        defaultValue="live"
      >
        {(active: string) => {
          if (active === "live") return <LiveTry onTrace={onTrace} />;
          if (active === "oneshot") return <OneshotTry onTrace={onTrace} />;
          return <FileTry onTrace={onTrace} />;
        }}
      </Tabs>
    </Panel>
  );
}

export { LiveTry, OneshotTry, FileTry };
