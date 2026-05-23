import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { DomainInventory } from "./DomainInventory";
import { ManifestValidationReport } from "./ManifestValidationReport";
import {
  useGetManifest,
  useValidateManifest,
} from "./controllers/useManifestController";

export interface ManifestViewProps {
  scenario: string;
}

export function ManifestView({ scenario }: ManifestViewProps) {
  const { t } = useTranslation();
  const manifest = useGetManifest(scenario);
  const validate = useValidateManifest(scenario);

  if (manifest.isPending) {
    return (
      <div data-testid={selectors.features.manifest.view.loading}>
        <LoadingState label={t(strings.pages.targetManifest.loading)} />
      </div>
    );
  }
  if (manifest.isError) {
    return (
      <div data-testid={selectors.features.manifest.view.error}>
        <ErrorState
          title={t(strings.pages.targetManifest.errorTitle)}
          message={manifest.error instanceof Error ? manifest.error.message : String(manifest.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void manifest.refetch();
          }}
        />
      </div>
    );
  }

  const def = manifest.data.manifest;
  const diagnostics = validate.data?.diagnostics ?? [];
  const valid = validate.data?.valid ?? true;

  return (
    <div data-testid={selectors.features.manifest.view.root} className="flex flex-col gap-4">
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="default"
          size="sm"
          data-testid={selectors.features.manifest.view.validateButton}
          onClick={() => validate.mutate()}
          disabled={validate.isPending}
        >
          {validate.isPending
            ? t(strings.pages.targetManifest.validating)
            : t(strings.pages.targetManifest.validateButton)}
        </Button>
      </div>

      {validate.data ? (
        <section aria-labelledby="manifest-diag-heading" className="flex flex-col gap-2">
          <h4 id="manifest-diag-heading" className="text-lg font-semibold">
            {t(strings.pages.targetManifest.diagnosticsHeading)}
          </h4>
          <ManifestValidationReport diagnostics={diagnostics} valid={valid} />
        </section>
      ) : null}

      <section aria-labelledby="manifest-domains-heading" className="flex flex-col gap-2">
        <h4 id="manifest-domains-heading" className="text-lg font-semibold">
          {t(strings.pages.targetManifest.domainsHeading)}
        </h4>
        {def ? (
          <DomainInventory domains={def.domains} />
        ) : (
          <EmptyState title={t(strings.pages.targetManifest.empty)} />
        )}
      </section>
    </div>
  );
}
