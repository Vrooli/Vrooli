/**
 * DomainCard displays information about a registered health domain.
 * Shows domain name, status, capabilities, and health information.
 *
 * [REQ:LD-DOMAIN-DISCOVER] - UI for domain discovery
 * [REQ:LD-DOMAIN-HEALTH] - Visual health status display
 */
import { Heart } from "lucide-react";
import { StatusBadge } from "./StatusBadge";
import { Card } from "../ui";
import { formatRelativeTime } from "../../lib/format";
import type { Domain } from "../../lib/api";

interface DomainCardProps {
  domain: Domain;
  onClick?: () => void;
}

export function DomainCard({ domain, onClick }: DomainCardProps) {
  return (
    <Card
      interactive
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === "Enter" && onClick?.()}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-violet-500/30 to-fuchsia-500/30 flex items-center justify-center">
            <Heart className="w-5 h-5 text-violet-400" />
          </div>
          <div>
            <h3 className="font-medium text-slate-100">{domain.display_name}</h3>
            <p className="text-xs text-slate-500">{domain.name}</p>
          </div>
        </div>
        <StatusBadge status={domain.status} />
      </div>
      {domain.description && (
        <p className="mt-3 text-sm text-slate-400 line-clamp-2">{domain.description}</p>
      )}
      {domain.capabilities && domain.capabilities.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1">
          {domain.capabilities.slice(0, 3).map((cap) => (
            <span key={cap} className="text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-400">
              {cap}
            </span>
          ))}
          {domain.capabilities.length > 3 && (
            <span className="text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-400">
              +{domain.capabilities.length - 3}
            </span>
          )}
        </div>
      )}
      {domain.last_health_at && (
        <p className="mt-3 text-xs text-slate-500">
          Last check: {formatRelativeTime(domain.last_health_at)}
        </p>
      )}
    </Card>
  );
}
