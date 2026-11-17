import { create } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import { CachedModeReadback, OperationState, ZoneApplyResultSchema, ZoneStateSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { EffectDeliveryNotice, ZoneColorEvidence } from "./ColorEvidence";

it("labels acknowledged native effect mode as mode evidence, not pulse proof", () => {
  const desired = create(ZoneStateSchema, { desiredMode: { case: "effect", value: { effectId: "Breathing", color: { red: 217, green: 154, blue: 36 }, brightnessPercent: 25 } } });
  const applied = create(ZoneApplyResultSchema, {
    desired,
    operation: OperationState.BACKEND_ACKNOWLEDGED,
    backendAcknowledged: true,
    cachedModeReadback: CachedModeReadback.MATCHES,
  });

  renderWithProviders(<>
    <EffectDeliveryNotice desired={desired} lastApply={applied}/>
    <ZoneColorEvidence title="Effect input" mode={desired.desiredMode} scaledLabel="Static RGB" offLabel="Off" noColorLabel="No color" missingLabel="Missing"/>
  </>);

  expect(screen.getByRole("status")).toHaveTextContent(strings.lightDesk.effectNativeModeConfirmed);
  expect(screen.getByText(strings.lightDesk.effectColorInput)).toBeInTheDocument();
});

it("does not describe an unacknowledged native effect as pulsing", () => {
  const desired = create(ZoneStateSchema, { desiredMode: { case: "effect", value: { effectId: "Breathing", color: { red: 217, green: 154, blue: 36 }, brightnessPercent: 25 } } });
  const applied = create(ZoneApplyResultSchema, { desired, operation: OperationState.FAILED });

  renderWithProviders(<EffectDeliveryNotice desired={desired} lastApply={applied}/>);

  expect(screen.getByRole("alert")).toHaveTextContent(strings.lightDesk.effectNativeModeUnverified);
});
