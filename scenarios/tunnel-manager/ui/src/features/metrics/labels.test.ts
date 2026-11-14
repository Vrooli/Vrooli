/**
 * Exhaustive coverage of the probe/classification enum → label / tone maps.
 */
import { describe, expect, it } from "vitest";
import {
  ProbeKind,
  ProbeStatus,
  FailureClass,
} from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

import { strings } from "../../consts/strings";
import {
  probeKindLabel,
  probeStatusLabel,
  probeStatusTone,
  failureClassLabel,
  failureClassTone,
} from "./labels";

describe("metrics labels", () => {
  it("maps every probe kind", () => {
    expect(probeKindLabel(ProbeKind.INTERNAL)).toBe(strings.metrics.kind.internal);
    expect(probeKindLabel(ProbeKind.EXTERNAL)).toBe(strings.metrics.kind.external);
    expect(probeKindLabel(ProbeKind.UNSPECIFIED)).toBe(strings.metrics.kind.unknown);
  });

  it("maps every probe status to its label", () => {
    expect(probeStatusLabel(ProbeStatus.UP)).toBe(strings.metrics.probeStatus.up);
    expect(probeStatusLabel(ProbeStatus.DOWN)).toBe(strings.metrics.probeStatus.down);
    expect(probeStatusLabel(ProbeStatus.TIMEOUT)).toBe(strings.metrics.probeStatus.timeout);
    expect(probeStatusLabel(ProbeStatus.ERROR)).toBe(strings.metrics.probeStatus.error);
    expect(probeStatusLabel(ProbeStatus.UNSPECIFIED)).toBe(strings.metrics.probeStatus.unknown);
  });

  it("maps every probe status to its tone", () => {
    expect(probeStatusTone(ProbeStatus.UP)).toBe("success");
    expect(probeStatusTone(ProbeStatus.DOWN)).toBe("danger");
    expect(probeStatusTone(ProbeStatus.ERROR)).toBe("danger");
    expect(probeStatusTone(ProbeStatus.TIMEOUT)).toBe("warning");
    expect(probeStatusTone(ProbeStatus.UNSPECIFIED)).toBe("neutral");
  });

  it("maps every failure class to its label", () => {
    expect(failureClassLabel(FailureClass.HEALTHY)).toBe(strings.metrics.class.healthy);
    expect(failureClassLabel(FailureClass.TUNNEL_DOWN)).toBe(strings.metrics.class.tunnelDown);
    expect(failureClassLabel(FailureClass.SCENARIO_DOWN)).toBe(strings.metrics.class.scenarioDown);
    expect(failureClassLabel(FailureClass.CLOUDFLARE_OUTAGE)).toBe(strings.metrics.class.cloudflareOutage);
    expect(failureClassLabel(FailureClass.DNS_FAILURE)).toBe(strings.metrics.class.dnsFailure);
    expect(failureClassLabel(FailureClass.CONFIG_DRIFT)).toBe(strings.metrics.class.configDrift);
    expect(failureClassLabel(FailureClass.UNSPECIFIED)).toBe(strings.metrics.class.unknown);
  });

  it("maps failure-class tones (healthy / neutral / danger)", () => {
    expect(failureClassTone(FailureClass.HEALTHY)).toBe("success");
    expect(failureClassTone(FailureClass.UNSPECIFIED)).toBe("neutral");
    expect(failureClassTone(FailureClass.TUNNEL_DOWN)).toBe("danger");
    expect(failureClassTone(FailureClass.DNS_FAILURE)).toBe("danger");
  });
});
