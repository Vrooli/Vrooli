import { AlertTriangle, CheckCircle2, HelpCircle, RefreshCw, XOctagon } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { SidecarStatus } from "@vrooli/proto-types/typescript-code-graph/v1/health/health_pb";

import { strings } from "../../consts/strings";

type StatusKey = (typeof strings.sidecar.status)[keyof typeof strings.sidecar.status];

/**
 * Stable, locale-independent token for each sidecar state. Doubles as the
 * selector parameter and the dot's visual variant so tests target a state
 * without binding to translated copy or a color.
 */
export type SidecarStatusToken =
  | "unspecified"
  | "ready"
  | "unhealthy"
  | "restarting"
  | "permanently_unhealthy";

export interface SidecarStatusMeta {
  readonly token: SidecarStatusToken;
  /** i18n key for the human label. */
  readonly labelKey: StatusKey;
  /** Icon — pairs with the label so status is never conveyed by color alone. */
  readonly icon: LucideIcon;
  /**
   * Token-driven dot/text accent class. Color is redundant with the icon +
   * label, satisfying the non-color-only A11y contract.
   */
  readonly accentClass: string;
}

/**
 * Map the SidecarStatus proto enum to its label + icon + accent. Pure and
 * total: every enum value (including the UNSPECIFIED default and the terminal
 * PERMANENTLY_UNHEALTHY) maps to a distinct, icon-bearing entry.
 */
export function sidecarStatusMeta(status: SidecarStatus): SidecarStatusMeta {
  switch (status) {
    case SidecarStatus.READY:
      return {
        token: "ready",
        labelKey: strings.sidecar.status.ready,
        icon: CheckCircle2,
        accentClass: "text-app-success",
      };
    case SidecarStatus.UNHEALTHY:
      return {
        token: "unhealthy",
        labelKey: strings.sidecar.status.unhealthy,
        icon: AlertTriangle,
        accentClass: "text-app-warning",
      };
    case SidecarStatus.RESTARTING:
      return {
        token: "restarting",
        labelKey: strings.sidecar.status.restarting,
        icon: RefreshCw,
        accentClass: "text-app-primary",
      };
    case SidecarStatus.PERMANENTLY_UNHEALTHY:
      return {
        token: "permanently_unhealthy",
        labelKey: strings.sidecar.status.permanentlyUnhealthy,
        icon: XOctagon,
        accentClass: "text-app-danger",
      };
    default:
      return {
        token: "unspecified",
        labelKey: strings.sidecar.status.unspecified,
        icon: HelpCircle,
        accentClass: "text-app-muted-foreground",
      };
  }
}
