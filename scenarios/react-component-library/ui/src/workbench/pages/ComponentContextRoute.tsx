import { useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { appRoutes, workspacePath } from "../../routes";
import { getWorkspace, WorkspaceProvenance } from "../../api/workspaces";
import { Banner, Skeleton, WbButton } from "../primitives";
import { loadComponentBundle } from "../data";
import { readLegacyTemplateForMigration } from "../templates/readLegacyTemplate";
import type { Template } from "../templates/spec";
import {
  createComponentWorkspace,
  createComponentWorkspaceFromTemplate,
  curatedWorkspaceIdForTemplate,
  DEFAULT_COMPONENT_WORKSPACE_ID,
} from "../workspaces/componentContext";

interface LegacyMigrationChoice {
  key: string;
  template: Template;
  componentId: string;
  storyId: string;
  componentVersion?: string;
}

/** Converts bookmarked legacy component-context URLs into durable workspaces. */
export function ComponentContextRoute() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [error, setError] = useState<string | null>(null);
  const [legacyMigration, setLegacyMigration] = useState<LegacyMigrationChoice | null>(null);
  const [importingLegacy, setImportingLegacy] = useState(false);
  const started = useRef("");
  const templateRef = params.get("template");
  const templateId = curatedWorkspaceIdForTemplate(templateRef ?? DEFAULT_COMPONENT_WORKSPACE_ID);
  const componentId = params.get("component") ?? "";
  const storyId = params.get("story") ?? "";
  const componentVersion = params.get("version") ?? undefined;

  useEffect(() => {
    const key = `${templateId}\u0000${componentId}\u0000${storyId}\u0000${componentVersion ?? ""}`;
    if (started.current === key) return;
    setLegacyMigration(null);
    setImportingLegacy(false);
    if (!componentId) {
      setError("A component ID is required to open this workspace.");
      return;
    }
    started.current = key;
    setError(null);
    void loadComponentBundle(componentId, componentVersion)
      .then((bundle) => {
        if (started.current !== key) return null;
        const selectedStory = storyId
          ? bundle.stories.find((story) => story.id === storyId)
          : bundle.stories[0];
        if (!selectedStory) {
          throw new Error(
            storyId
              ? `Story “${storyId}” is unavailable for component “${componentId}”${componentVersion ? ` at version ${componentVersion}` : ""}. No workspace was created.`
              : `Component “${componentId}” has no available stories. No workspace was created.`,
          );
        }
        return getWorkspace(templateId).then((templateWorkspace) => ({
          templateWorkspace,
          selectedStory,
        }));
      })
      .then((resolved) => {
        // A browser back/forward or a newer component-context link can replace
        // this route while the first lookup is in flight. Do not fork the old
        // template after the newer request has claimed the route.
        if (started.current !== key) return null;
        if (!resolved) return null;
        const { templateWorkspace, selectedStory } = resolved;
        if (templateWorkspace?.provenance === WorkspaceProvenance.CURATED) {
          return createComponentWorkspace({
            templateWorkspace,
            componentId,
            storyId: selectedStory.id,
            componentVersion: selectedStory.version,
          });
        }
        const legacyId = templateRef ?? templateId.replace(/^ws_curated_/, "");
        const template = readLegacyTemplateForMigration(legacyId);
        if (!template) {
          throw new Error(
            `Unable to migrate template “${templateRef ?? legacyId}”: no matching curated workspace or browser cache entry was found. No workspace was created; the original reference remains in the URL.`,
          );
        }
        // Browser-only drafts may contain the only surviving copy. Keep the
        // migration available, but do not copy local data to the server until
        // the person explicitly chooses to import it.
        setLegacyMigration({
          key,
          template,
          componentId,
          storyId: selectedStory.id,
          componentVersion: selectedStory.version,
        });
        return null;
      })
      .then((workspace) => {
        if (!workspace || started.current !== key) return;
        void navigate(workspacePath(workspace.id), { replace: true });
      })
      .catch((cause: unknown) => {
        if (started.current !== key) return;
        started.current = "";
        setError(
          cause instanceof Error ? cause.message : "Could not open the component workspace.",
        );
      });
  }, [componentId, componentVersion, navigate, storyId, templateId, templateRef]);

  const confirmLegacyImport = async () => {
    const choice = legacyMigration;
    if (!choice || choice.key !== started.current || importingLegacy) return;
    setImportingLegacy(true);
    setError(null);
    try {
      const workspace = await createComponentWorkspaceFromTemplate({
        template: choice.template,
        componentId: choice.componentId,
        storyId: choice.storyId,
        componentVersion: choice.componentVersion,
      });
      if (started.current === choice.key) {
        void navigate(workspacePath(workspace.id), { replace: true });
      }
    } catch (cause) {
      if (started.current !== choice.key) return;
      setError(cause instanceof Error ? cause.message : "Could not import the browser draft.");
      setLegacyMigration(null);
    } finally {
      setImportingLegacy(false);
    }
  };

  if (legacyMigration?.key === started.current) {
    return (
      <section className="studio-loading" aria-labelledby="legacy-import-title">
        <h2 id="legacy-import-title">{t(strings.workbench.studio.legacyImportTitle)}</h2>
        <p>
          {t(strings.workbench.studio.legacyImportBody, {
            templateName: legacyMigration.template.name,
          })}
        </p>
        <div className="wb-actions">
          <WbButton onClick={() => void confirmLegacyImport()} disabled={importingLegacy}>
            {t(
              importingLegacy
                ? strings.workbench.studio.legacyImportPending
                : strings.workbench.studio.legacyImportConfirm,
            )}
          </WbButton>
          <WbButton variant="secondary" onClick={() => void navigate(appRoutes.design)}>
            {t(strings.workbench.studio.legacyImportBack)}
          </WbButton>
        </div>
      </section>
    );
  }
  if (error) return <Banner tone="red">{error}</Banner>;
  return (
    <div className="studio-loading" aria-live="polite">
      <Skeleton height={24} />
      <Skeleton height={240} />
      <p>
        {t("workbench.studio.openingComponentWorkspace", {
          defaultValue: "Opening component workspace…",
        })}
      </p>
    </div>
  );
}
