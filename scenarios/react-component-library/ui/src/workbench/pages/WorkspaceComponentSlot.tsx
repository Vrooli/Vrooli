import { FixtureFrame } from "../component/FixtureFrame";
import { useTranslation } from "../../i18n";
import { Skeleton } from "../primitives";
import { useComponentBundle } from "../data";

export function WorkspaceComponentSlot({
  componentId,
  storyId,
  componentVersion,
  height,
}: {
  componentId: string;
  storyId: string;
  componentVersion?: string;
  /** Match the authored slot footprint so the real specimen is not cropped. */
  height: number;
}) {
  const { t } = useTranslation();
  const bundle = useComponentBundle(componentId, componentVersion);
  if (bundle.isLoading) return <Skeleton height={height} />;
  if (!bundle.data) {
    return (
      <span className="tpl-slot-hint">
        {t("workbench.studio.slotUnavailable", { defaultValue: "Component unavailable" })}
      </span>
    );
  }
  const story = bundle.data.stories.find((entry) => entry.id === storyId) ?? bundle.data.stories[0];
  if (!story) {
    return (
      <span className="tpl-slot-hint">
        {t("workbench.studio.slotNoStory", { defaultValue: "No story" })}
      </span>
    );
  }
  return (
    <FixtureFrame
      assetId={componentId}
      storyId={story.id}
      version={story.version}
      height={height}
      title={story.name || story.id}
    />
  );
}
