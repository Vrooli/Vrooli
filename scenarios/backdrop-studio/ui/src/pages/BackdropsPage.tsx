import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { Link } from "react-router-dom";

/**
 * Released backdrops.
 *
 * The release service resolves a backdrop by id and serves its bytes, but
 * exposes no list: releases are addressed by the consumer that requested them,
 * not browsed. Rather than invent a listing this API cannot serve — or show an
 * empty table that reads as a bug — the page says what is true and points at
 * the surface where a release is actually made.
 *
 * When `ReleaseService` gains a list RPC this becomes a real index; until then
 * an honest empty state is the correct build.
 */
export function BackdropsPage() {
  const { t } = useTranslation();
  return (
    <ExperienceSurface
      surfaceId="backdrops"
      state="empty"
      data-testid={selectors.pages.backdrops}
      aria-labelledby="backdrops-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="backdrops-heading" className="text-2xl font-semibold">
          {t(strings.pages.backdrops.title)}
        </h2>
        <p className="max-w-3xl text-app-muted-foreground">
          {t(strings.pages.backdrops.description)}
        </p>
      </header>
      <EmptyState
        title={t(strings.pages.backdrops.title)}
        description={t(strings.pages.backdrops.empty)}
        action={
          <Link
            to="/catalog"
            className="inline-flex min-h-11 items-center rounded-control border border-app-border px-4 text-sm font-medium"
          >
            {t(strings.pages.style.backToCatalog)}
          </Link>
        }
      />
    </ExperienceSurface>
  );
}
