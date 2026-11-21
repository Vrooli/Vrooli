import { useLayoutEffect, useRef, useState } from "react";
import { SettingsList } from "./SettingsList";

function Specimen({ width }: { width: number }) {
  const rootRef = useRef<HTMLDivElement>(null);
  const [measurement, setMeasurement] = useState("measuring");

  useLayoutEffect(() => {
    const frame = requestAnimationFrame(() => {
      const group = rootRef.current?.querySelector<HTMLElement>(
        "[data-rcl-settings-group-surface]",
      );
      const row = rootRef.current?.querySelector<HTMLElement>("[data-rcl-settings-row]");
      const control = rootRef.current?.querySelector<HTMLElement>(
        "[data-rcl-settings-row-control]",
      );
      if (!group || !row || !control) return;
      const groupStyle = getComputedStyle(group);
      const rowStyle = getComputedStyle(row);
      const controlStyle = getComputedStyle(control);
      control.dataset.measuredTrailing = String(controlStyle.justifySelf === "end");
      setMeasurement(
        `width=${width}; border=${groupStyle.borderTopWidth}; min-height=${rowStyle.minHeight}`,
      );
    });
    return () => cancelAnimationFrame(frame);
  }, [width]);

  return (
    <div ref={rootRef}>
      <SettingsList style={{ width }} variant="auto">
        <SettingsList.Intro
          eyebrow="Speech recognition"
          title="Voice input"
          description="Control voice capture preferences."
        />
        <SettingsList.Group label="Capture">
          <SettingsList.Row
            label="Voice input"
            hint="Enable speech-to-text controls."
            control="compact"
          >
            <button type="button">On</button>
          </SettingsList.Row>
          <SettingsList.Row
            label="Enrollment"
            hint="Arbitrary composed content remains supported."
            control="wide"
          >
            <div data-testid="settings-list-arbitrary-child">Enrollment panel</div>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>
      <output data-testid="settings-list-measurement">{measurement}</output>
    </div>
  );
}

export const Width320 = () => <Specimen width={320} />;
export const Width390 = () => <Specimen width={390} />;
export const Width700 = () => <Specimen width={700} />;
export const Width1100 = () => <Specimen width={1100} />;
