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
      // A compact control stays beside its text whenever the two fit on one line.
      const text = row.querySelector<HTMLElement>("[data-rcl-settings-row-hint]") ?? row;
      control.dataset.measuredTrailing = String(
        control.getBoundingClientRect().top < text.getBoundingClientRect().bottom,
      );
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

function NavigableSpecimen({ width }: { width: number }) {
  const rootRef = useRef<HTMLDivElement>(null);
  const [selected, setSelected] = useState("none");
  const [measurement, setMeasurement] = useState("measuring");

  useLayoutEffect(() => {
    const frame = requestAnimationFrame(() => {
      const root = rootRef.current;
      const row = root?.querySelector<HTMLElement>("[data-rcl-settings-row][data-selectable]");
      const action = root?.querySelector<HTMLElement>("[data-testid='settings-list-row-action']");
      if (!root || !row || !action) return;
      const box = action.getBoundingClientRect();
      const hit = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2);
      setMeasurement(
        `selectable=${root.querySelectorAll("[data-rcl-settings-row-select]").length}; leading=${root.querySelectorAll("[data-rcl-settings-row-leading]").length}; chevrons=${root.querySelectorAll("[data-rcl-settings-row-chevron]").length}; action-on-top=${String(hit === action || action.contains(hit))}`,
      );
    });
    return () => cancelAnimationFrame(frame);
  }, [width]);

  const icon = (
    <span
      aria-hidden="true"
      style={{
        inlineSize: 32,
        blockSize: 32,
        borderRadius: 8,
        background: "var(--color-surface-muted)",
        display: "inline-block",
      }}
    />
  );
  return (
    <div ref={rootRef}>
      <SettingsList style={{ width }}>
        <SettingsList.Group label="Services">
          <SettingsList.Row
            label="Voice"
            hint="Dictation and read-aloud"
            leading={icon}
            onSelect={() => {
              setSelected("voice");
            }}
          >
            <span>Ready</span>
          </SettingsList.Row>
          <SettingsList.Row label="Grok" hint="Not installed" leading={icon}>
            <button
              type="button"
              data-testid="settings-list-row-action"
              onClick={() => {
                setSelected("install");
              }}
            >
              Install
            </button>
          </SettingsList.Row>
          <SettingsList.Row
            label="Remote terminals"
            hint="Off"
            leading={icon}
            onSelect={() => {
              setSelected("remote");
            }}
          >
            <button
              type="button"
              onClick={() => {
                setSelected("start");
              }}
            >
              Start
            </button>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>
      <output data-testid="settings-list-measurement">{measurement}</output>
      <output data-testid="settings-list-selected">{selected}</output>
    </div>
  );
}

export const Navigable390 = () => <NavigableSpecimen width={390} />;
export const Navigable1100 = () => <NavigableSpecimen width={1100} />;
