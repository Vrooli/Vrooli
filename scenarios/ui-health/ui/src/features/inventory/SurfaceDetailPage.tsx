import { useParams } from "react-router-dom";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function SurfaceDetailPage() {
  const { t } = useTranslation();
  const { surfaceId } = useParams<{ surfaceId: string }>();
  return (
    <section
      data-testid={selectors.pages.surfaceDetail}
      aria-labelledby="surface-detail-heading"
      className="flex flex-col gap-3"
    >
      <h2 id="surface-detail-heading" className="text-2xl font-semibold tracking-tight">
        {t(strings.pages.inventory.detail.title)}
      </h2>
      <p className="text-sm text-app-muted-foreground break-all">{surfaceId}</p>
    </section>
  );
}
