import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ListGroup } from "./ListRow";

/** Placeholder rows shaped like list rows, announced once as loading. */
export function SkeletonRows({ rows = 3 }: { rows?: number }) {
  const { t } = useTranslation();
  return (
    <div>
      <p role="status" className="sr-only">{t(strings.common.loading)}</p>
      <ListGroup aria-hidden="true">
        {Array.from({ length: rows }, (_, index) => (
          <li key={index} className="flex flex-col gap-2 px-4 py-4">
            <Skeleton role="presentation" aria-label={undefined} className="h-4 w-3/4 motion-safe:animate-pulse" />
            <Skeleton role="presentation" aria-label={undefined} className="h-3 w-1/2 motion-safe:animate-pulse" />
          </li>
        ))}
      </ListGroup>
    </div>
  );
}
