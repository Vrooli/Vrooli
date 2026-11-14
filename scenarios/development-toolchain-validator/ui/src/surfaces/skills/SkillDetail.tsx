import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { skillCatalogClient } from "../../api/skillCatalog";
import { errorMessage } from "../../lib/errorMessage";
import { ROUTES } from "../../routes.generated";
import { Button } from "../../shared/ui/primitives/Button";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { MetadataList } from "../../shared/ui/composites/MetadataList";

/**
 * Skill detail surface — single skill view with version + content hash.
 * Per-golden convergence timelines land when the report-by-skill query
 * surfaces them (P1).
 */
export function SkillDetail() {
  const { t } = useTranslation();
  const params = useParams<{ id: string }>();
  const id = params.id ?? "";
  const navigate = useNavigate();

  const skillQuery = useQuery({
    queryKey: ["skill", id] as const,
    queryFn: () => skillCatalogClient.getSkill({ id }),
    enabled: id.length > 0,
  });

  if (skillQuery.isLoading) {
    return (
      <LoadingSkeleton
        data-testid={selectors.skills.loading}
        variant="card"
        count={2}
      />
    );
  }

  if (skillQuery.error) {
    return (
      <p
        data-testid={selectors.skills.error}
        className="text-sm text-status-failure"
      >
        {errorMessage(skillQuery.error, t)}
      </p>
    );
  }

  const skill = skillQuery.data?.skill;
  if (!skill) {
    return (
      <EmptyState
        testId={selectors.skills.empty}
        title={t(strings.skills.empty)}
        description={t(strings.skills.emptyDescription)}
        action={
          <Button
            size="sm"
            variant="outline"
            onClick={() => void navigate(ROUTES.skillsIndex)}
          >
            {t(strings.skills.backToIndex)}
          </Button>
        }
      />
    );
  }

  return (
    <section
      data-testid={selectors.skills.detail}
      aria-labelledby={selectors.skills.detailHeading}
      className="flex flex-col gap-5"
    >
      <PanelHeader
        title={
          <span
            data-testid={selectors.skills.detailHeading}
            id={selectors.skills.detailHeading}
          >
            {t(strings.skills.detailHeading, { id: skill.id })}
          </span>
        }
        actions={
          <Button
            data-testid={selectors.skills.detailBack}
            size="sm"
            variant="ghost"
            onClick={() => void navigate(ROUTES.skillsIndex)}
          >
            <ArrowLeft className="h-4 w-4" />
            {t(strings.skills.backToIndex)}
          </Button>
        }
      />
      <MetadataList
        items={[
          { label: t(strings.goldens.versionLabel), value: skill.version },
          {
            label: t(strings.skills.contentHashLabel),
            value: skill.contentHash,
            mono: true,
          },
        ]}
      />
    </section>
  );
}
