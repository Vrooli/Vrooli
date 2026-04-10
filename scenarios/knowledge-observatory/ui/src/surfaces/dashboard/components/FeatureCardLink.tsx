// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { ReactNode } from "react";
import { routeToHash, type Route } from "../../../shared/controllers/routeController";

export type FeatureCardLinkProps = {
  route: Route;
  title: string;
  description: string;
  icon: ReactNode;
  badge?: string;
  testId?: string;
};

export function FeatureCardLink({
  route,
  title,
  description,
  icon,
  badge,
  testId,
}: FeatureCardLinkProps) {
  return (
    <a
      href={routeToHash(route)}
      data-testid={testId}
      className="ko-panel ko-panel-inset ko-panel-link p-6"
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="mb-3">{icon}</div>
          <h3 className="ko-text-lg font-semibold mb-2">{title}</h3>
          <p className="ko-text-sm ko-muted">{description}</p>
        </div>
        {badge ? <span className="ko-tag">{badge}</span> : null}
      </div>
    </a>
  );
}
