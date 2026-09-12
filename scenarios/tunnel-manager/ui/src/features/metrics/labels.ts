import {
  ProbeKind,
  ProbeStatus,
  FailureClass,
} from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

import { strings } from "../../consts/strings";
import type { BadgeTone } from "../../components/ui/StatusBadge";

type KindKey = (typeof strings.metrics.kind)[keyof typeof strings.metrics.kind];
type ProbeStatusKey = (typeof strings.metrics.probeStatus)[keyof typeof strings.metrics.probeStatus];
type ClassKey = (typeof strings.metrics.class)[keyof typeof strings.metrics.class];

/** Probe kind (internal local-port vs external public-URL) → translation key. */
export function probeKindLabel(kind: ProbeKind): KindKey {
  switch (kind) {
    case ProbeKind.INTERNAL:
      return strings.metrics.kind.internal;
    case ProbeKind.EXTERNAL:
      return strings.metrics.kind.external;
    default:
      return strings.metrics.kind.unknown;
  }
}

/** Probe outcome → translation key. */
export function probeStatusLabel(status: ProbeStatus): ProbeStatusKey {
  switch (status) {
    case ProbeStatus.UP:
      return strings.metrics.probeStatus.up;
    case ProbeStatus.DOWN:
      return strings.metrics.probeStatus.down;
    case ProbeStatus.TIMEOUT:
      return strings.metrics.probeStatus.timeout;
    case ProbeStatus.ERROR:
      return strings.metrics.probeStatus.error;
    default:
      return strings.metrics.probeStatus.unknown;
  }
}

/** Probe outcome → badge tone. */
export function probeStatusTone(status: ProbeStatus): BadgeTone {
  switch (status) {
    case ProbeStatus.UP:
      return "success";
    case ProbeStatus.DOWN:
    case ProbeStatus.ERROR:
      return "danger";
    case ProbeStatus.TIMEOUT:
      return "warning";
    default:
      return "neutral";
  }
}

/** Failure classification → translation key. */
export function failureClassLabel(cls: FailureClass): ClassKey {
  switch (cls) {
    case FailureClass.HEALTHY:
      return strings.metrics.class.healthy;
    case FailureClass.TUNNEL_DOWN:
      return strings.metrics.class.tunnelDown;
    case FailureClass.SCENARIO_DOWN:
      return strings.metrics.class.scenarioDown;
    case FailureClass.CLOUDFLARE_OUTAGE:
      return strings.metrics.class.cloudflareOutage;
    case FailureClass.DNS_FAILURE:
      return strings.metrics.class.dnsFailure;
    case FailureClass.CONFIG_DRIFT:
      return strings.metrics.class.configDrift;
    default:
      return strings.metrics.class.unknown;
  }
}

/** Failure classification → badge tone. */
export function failureClassTone(cls: FailureClass): BadgeTone {
  switch (cls) {
    case FailureClass.HEALTHY:
      return "success";
    case FailureClass.UNSPECIFIED:
      return "neutral";
    default:
      return "danger";
  }
}
