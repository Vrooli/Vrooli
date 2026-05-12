/**
 * ComponentDetailPage — full-width editor + preview for a component.
 *
 * Resolves the component id from the route param, fetches the
 * component record so the editor header shows the libraryId (instead
 * of the bare id), and hands the rest off to `<ComponentEditor />`.
 * Closing the editor returns the user to the components list.
 */
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";

import { componentsClient } from "../api/components";
import { ComponentEditor } from "../features/components/ComponentEditor";
import { useTranslation } from "../i18n";

export function ComponentDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();

  const { data, isLoading, error } = useQuery({
    queryKey: ["components", "get", id],
    queryFn: () => componentsClient.getComponent({ id: id ?? "" }),
    enabled: !!id,
  });

  if (!id) {
    return (
      <p data-testid="component-detail-missing-id" className="text-sm text-app-danger">
        {t("componentDetail.missingId", { defaultValue: "No component id in URL." })}
      </p>
    );
  }

  if (isLoading) {
    return (
      <p data-testid="component-detail-loading" className="text-sm text-app-muted-foreground">
        {t("componentDetail.loading", { defaultValue: "Loading component…" })}
      </p>
    );
  }

  if (error || !data?.component) {
    return (
      <p data-testid="component-detail-error" className="text-sm text-app-danger">
        {t("componentDetail.notFound", { defaultValue: "Component not found." })}
      </p>
    );
  }

  return (
    <div data-testid="component-detail-page" className="flex flex-col gap-3">
      <ComponentEditor
        id={data.component.id}
        libraryId={data.component.libraryId || data.component.id}
        onClose={() => navigate("/components")}
      />
    </div>
  );
}
