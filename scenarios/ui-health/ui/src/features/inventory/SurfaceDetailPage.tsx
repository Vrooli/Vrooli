import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { Badge } from "../../components/ui/Badge";
import { Card, CardBody, CardHeader, CardTitle } from "../../components/ui/Card";
import { CodeBlock } from "../../components/ui/CodeBlock";
import { EmptyState } from "../../components/ui/EmptyState";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES } from "../../routes.generated";
import {
  decodeSurfaceId,
  scanScenario,
  type ProvenanceTag,
  type SurfaceKind,
} from "../../api/inventory";
import { inventoryQueryKey } from "./useInventory";

const KIND_LABEL_KEY = {
  component: strings.pages.search.kind.component,
  page: strings.pages.search.kind.page,
  feature: strings.pages.search.kind.feature,
  hook: strings.pages.search.kind.hook,
  layout: strings.pages.search.kind.layout,
  other: strings.pages.search.kind.other,
  unspecified: strings.pages.search.kind.unspecified,
} as const satisfies Record<SurfaceKind, string>;

const PROVENANCE_LABEL_KEY = {
  custom: strings.pages.search.provenance.custom,
  "adopted-unmodified": strings.pages.search.provenance.adoptedUnmodified,
  "adopted-modified": strings.pages.search.provenance.adoptedModified,
  unknown: strings.pages.search.provenance.unknown,
  unspecified: strings.pages.search.provenance.unspecified,
} as const satisfies Record<ProvenanceTag, string>;

const PROVENANCE_TONE: Record<
  ProvenanceTag,
  "neutral" | "info" | "success" | "warn" | "error"
> = {
  custom: "info",
  "adopted-unmodified": "success",
  "adopted-modified": "warn",
  unknown: "neutral",
  unspecified: "neutral",
};

// Reuses inventoryQueryKey from api/inventory so a scan triggered from
// InventoryPage shares its cached result with this detail page.

export function SurfaceDetailPage() {
  const { t } = useTranslation();
  const { surfaceId = "" } = useParams<{ surfaceId: string }>();
  const decoded = decodeSurfaceId(surfaceId);

  const scenario = decoded?.scenario ?? "";
  const slot = decoded?.slot ?? "";

  const query_ = useQuery({
    queryKey: inventoryQueryKey(scenario),
    queryFn: () => scanScenario(scenario),
    enabled: scenario.length > 0,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  });

  if (!decoded) {
    return (
      <section
        data-testid={selectors.pages.surfaceDetail}
        aria-labelledby="surface-detail-heading"
        className="flex flex-col gap-4"
      >
        <BackLink />
        <h2 id="surface-detail-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.inventory.detail.title)}
        </h2>
        <EmptyState
          title={t(strings.pages.inventory.detail.notFound.title)}
          description={t(strings.pages.inventory.detail.notFound.description, {
            slot: surfaceId,
            scenario: "—",
          })}
          data-testid={selectors.inventory.detail.notFound}
        />
      </section>
    );
  }

  const surface = query_.data?.surfaces.find((s) => s.slot === slot);
  const provenanceList = query_.data?.provenance ?? [];
  const widgets = query_.data?.widgets ?? [];

  return (
    <section
      data-testid={selectors.pages.surfaceDetail}
      aria-labelledby="surface-detail-heading"
      className="flex flex-col gap-4"
    >
      <BackLink />
      <h2 id="surface-detail-heading" className="text-2xl font-semibold tracking-tight">
        {t(strings.pages.inventory.detail.title)}
      </h2>

      {query_.isFetching ? (
        <p
          className="flex items-center gap-2 text-sm text-app-muted-foreground"
          role="status"
          aria-live="polite"
          data-testid={selectors.inventory.detail.loading}
        >
          <Loader2 aria-hidden className="h-4 w-4 animate-spin" />
          {t(strings.pages.inventory.detail.loading)}
        </p>
      ) : null}

      {query_.error ? (
        <div
          role="alert"
          data-testid={selectors.inventory.detail.error}
          className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
        >
          {t(strings.pages.inventory.detail.error, {
            message:
              query_.error instanceof Error ? query_.error.message : String(query_.error),
          })}
        </div>
      ) : null}

      {query_.isSuccess && !surface ? (
        <EmptyState
          title={t(strings.pages.inventory.detail.notFound.title)}
          description={t(strings.pages.inventory.detail.notFound.description, {
            slot,
            scenario,
          })}
          data-testid={selectors.inventory.detail.notFound}
        />
      ) : null}

      {surface ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle>
                <span className="break-all">{surface.displayName || surface.slot || slot}</span>
              </CardTitle>
            </CardHeader>
            <CardBody>
              <dl
                className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-[max-content_1fr]"
                data-testid={selectors.inventory.detail.meta}
              >
                <DefRow label={t(strings.pages.inventory.detail.scenario)}>
                  <span className="font-mono break-all">{surface.scenario}</span>
                </DefRow>
                <DefRow label={t(strings.pages.inventory.detail.slot)}>
                  <span className="font-mono break-all">{surface.slot || "—"}</span>
                </DefRow>
                <DefRow label={t(strings.pages.inventory.detail.kind)}>
                  <Badge tone="neutral">{t(KIND_LABEL_KEY[surface.kind])}</Badge>
                </DefRow>
                <DefRow label={t(strings.pages.inventory.detail.filePath)}>
                  <span className="font-mono text-xs break-all">{surface.filePath || "—"}</span>
                </DefRow>
                {surface.description ? (
                  <DefRow label={t(strings.pages.inventory.detail.description)}>
                    <span className="text-sm">{surface.description}</span>
                  </DefRow>
                ) : null}
              </dl>
            </CardBody>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t(strings.pages.inventory.detail.provenance.heading)}</CardTitle>
            </CardHeader>
            <CardBody data-testid={selectors.inventory.detail.provenance}>
              {provenanceList.length === 0 ? (
                <p className="text-sm text-app-muted-foreground">
                  {t(strings.pages.inventory.detail.provenance.empty)}
                </p>
              ) : (
                <ul className="flex flex-col gap-3">
                  {provenanceList.map((p, idx) => (
                    <li
                      key={`${p.library}-${p.componentName}-${idx}`}
                      className="flex flex-col gap-1 rounded-control border border-app-border bg-app-surface p-3"
                    >
                      <div className="flex items-center gap-2">
                        <Badge tone={PROVENANCE_TONE[p.provenance]}>
                          {t(PROVENANCE_LABEL_KEY[p.provenance])}
                        </Badge>
                        <span className="font-mono text-xs break-all">{p.componentName || "—"}</span>
                      </div>
                      <dl className="grid grid-cols-1 gap-x-6 gap-y-1 sm:grid-cols-[max-content_1fr]">
                        <DefRow label={t(strings.pages.inventory.detail.provenance.library)}>
                          <span className="font-mono text-xs break-all">{p.library || "—"}</span>
                        </DefRow>
                        <DefRow label={t(strings.pages.inventory.detail.provenance.libraryVersion)}>
                          <span className="font-mono text-xs break-all">
                            {p.libraryVersion || "—"}
                          </span>
                        </DefRow>
                        <DefRow label={t(strings.pages.inventory.detail.provenance.adoptionId)}>
                          <span className="font-mono text-xs break-all">{p.adoptionId || "—"}</span>
                        </DefRow>
                      </dl>
                    </li>
                  ))}
                </ul>
              )}
            </CardBody>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t(strings.pages.inventory.detail.widgets.heading)}</CardTitle>
            </CardHeader>
            <CardBody data-testid={selectors.inventory.detail.widgets}>
              {widgets.length === 0 ? (
                <p className="text-sm text-app-muted-foreground">
                  {t(strings.pages.inventory.detail.widgets.empty)}
                </p>
              ) : (
                <ul className="flex flex-col gap-3">
                  {widgets.map((w) => (
                    <li
                      key={w.widgetId}
                      className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface p-3"
                    >
                      <div className="flex flex-wrap items-center gap-2 text-xs">
                        <span className="font-mono break-all text-app-foreground">{w.widgetId}</span>
                        <span aria-hidden className="text-app-muted-foreground">·</span>
                        <span className="font-mono break-all text-app-muted-foreground">
                          {w.componentName || "—"}
                        </span>
                      </div>
                      {w.propsSchemaJson ? (
                        <CodeBlock
                          language="json"
                          code={prettyJson(w.propsSchemaJson)}
                          caption={t(strings.pages.inventory.detail.widgets.propsSchema)}
                          copyLabel={t(strings.common.code.copy)}
                          copiedLabel={t(strings.common.code.copied)}
                          copyShortLabel={t(strings.common.code.copyShort)}
                        />
                      ) : null}
                    </li>
                  ))}
                </ul>
              )}
            </CardBody>
          </Card>
        </>
      ) : null}
    </section>
  );
}

function BackLink() {
  const { t } = useTranslation();
  return (
    <Link
      to={ROUTES.inventory}
      className="inline-flex items-center gap-1 self-start text-sm font-medium text-app-primary hover:underline"
      data-testid={selectors.inventory.detail.back}
    >
      <ArrowLeft aria-hidden className="h-3.5 w-3.5" />
      {t(strings.pages.inventory.detail.back)}
    </Link>
  );
}

function DefRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <>
      <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</dt>
      <dd className="text-sm text-app-foreground">{children}</dd>
    </>
  );
}

function prettyJson(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}
