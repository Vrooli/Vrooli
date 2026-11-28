import { create } from "@bufbuild/protobuf";
import {
  BrowserDesktopChromeSchema, CaptureArea, PhoneSystemChromeSchema, PresentationAppearance,
  PresentationKind, PresentationRGBSchema, PresentationRectSchema, ScreenFitMode,
  ScreenPresentationSchema, type ScreenPresentation,
} from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export type ScreenPresentationDraft = {
  captureArea: CaptureArea;
  kind: PresentationKind;
  fitMode: ScreenFitMode;
  matteColor: string;
  carrierText: string;
  clockText: string;
  batteryPercent: string;
  showCellularIndicator: boolean;
  showWifiIndicator: boolean;
  showHomeIndicator: boolean;
  showStatus: boolean;
  showBatteryPercent: boolean;
  phoneAppearance: PresentationAppearance;
  topFill: string;
  bottomFill: string;
  statusInk: string;
  homeInk: string;
  windowTitle: string;
  addressText: string;
  showWindowControls: boolean;
  showToolbar: boolean;
  browserAppearance: PresentationAppearance;
  windowX: string;
  windowY: string;
  windowWidth: string;
  windowHeight: string;
  showMenuBar: boolean;
  showDock: boolean;
  desktopColor: string;
  windowColor: string;
  chromeColor: string;
};

const colorToHex = (value?: { red: number; green: number; blue: number }) => value
  ? `#${[value.red, value.green, value.blue].map((channel) => channel.toString(16).padStart(2, "0")).join("")}` : "";
const colorFromHex = (value: string) => {
  const match = /^#?([\da-f]{2})([\da-f]{2})([\da-f]{2})$/i.exec(value.trim());
  return match ? create(PresentationRGBSchema, { red: Number.parseInt(match[1] ?? "", 16), green: Number.parseInt(match[2] ?? "", 16), blue: Number.parseInt(match[3] ?? "", 16) }) : undefined;
};

export function emptyScreenPresentationDraft(): ScreenPresentationDraft {
  return {
    captureArea: CaptureArea.APP_ONLY, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.CONTAIN, matteColor: "",
    carrierText: "", clockText: "9:41", batteryPercent: "84", showCellularIndicator: true, showWifiIndicator: true,
    showHomeIndicator: true, showStatus: true, showBatteryPercent: false, phoneAppearance: PresentationAppearance.AUTO,
    topFill: "", bottomFill: "", statusInk: "", homeInk: "", windowTitle: "", addressText: "",
    showWindowControls: true, showToolbar: true, browserAppearance: PresentationAppearance.LIGHT,
    windowX: "", windowY: "", windowWidth: "", windowHeight: "", showMenuBar: false, showDock: false,
    desktopColor: "", windowColor: "", chromeColor: "",
  };
}

export function screenPresentationDraftFromProto(presentation?: ScreenPresentation): ScreenPresentationDraft {
  const defaults = emptyScreenPresentationDraft();
  const phone = presentation?.phoneSystemChrome;
  const browser = presentation?.browserDesktopChrome;
  return {
    ...defaults, captureArea: presentation?.captureArea ?? defaults.captureArea,
    kind: presentation?.kind ?? defaults.kind, fitMode: presentation?.fitMode ?? defaults.fitMode,
    matteColor: colorToHex(presentation?.matte), carrierText: phone?.carrierText ?? "", clockText: phone?.clockText ?? "9:41",
    batteryPercent: String(phone?.batteryPercent ?? 84), showCellularIndicator: phone?.showCellularIndicator ?? true,
    showWifiIndicator: phone?.showWifiIndicator ?? true, showHomeIndicator: phone?.showHomeIndicator ?? true,
    showStatus: phone?.showStatus ?? true, showBatteryPercent: phone?.showBatteryPercent ?? false,
    phoneAppearance: phone?.appearance ?? defaults.phoneAppearance, topFill: colorToHex(phone?.topFill),
    bottomFill: colorToHex(phone?.bottomFill), statusInk: colorToHex(phone?.statusInk), homeInk: colorToHex(phone?.homeInk),
    windowTitle: browser?.windowTitle ?? "", addressText: browser?.addressText ?? "",
    showWindowControls: browser?.showWindowControls ?? true, showToolbar: browser?.showToolbar ?? true,
    browserAppearance: browser?.appearance ?? defaults.browserAppearance,
    windowX: browser?.window ? String(browser.window.x) : "", windowY: browser?.window ? String(browser.window.y) : "",
    windowWidth: browser?.window ? String(browser.window.width) : "", windowHeight: browser?.window ? String(browser.window.height) : "",
    showMenuBar: browser?.showMenuBar ?? false, showDock: browser?.showDock ?? false,
    desktopColor: colorToHex(browser?.desktopColor), windowColor: colorToHex(browser?.windowColor), chromeColor: colorToHex(browser?.chromeColor),
  };
}

export function screenPresentationFromDraft(sourceMediaId: string, draft: ScreenPresentationDraft, base?: ScreenPresentation): ScreenPresentation {
  const rect = [draft.windowX, draft.windowY, draft.windowWidth, draft.windowHeight];
  const window = rect.every((value) => value !== "") ? create(PresentationRectSchema, {
    x: Number(draft.windowX), y: Number(draft.windowY), width: Number(draft.windowWidth), height: Number(draft.windowHeight),
  }) : undefined;
  const phoneSystemChrome = create(PhoneSystemChromeSchema, {
    carrierText: draft.carrierText, clockText: draft.clockText, batteryPercent: Number(draft.batteryPercent || 0),
    showCellularIndicator: draft.showCellularIndicator, showWifiIndicator: draft.showWifiIndicator,
    showHomeIndicator: draft.showHomeIndicator, showStatus: draft.showStatus, showBatteryPercent: draft.showBatteryPercent,
    appearance: draft.phoneAppearance, topFill: colorFromHex(draft.topFill), bottomFill: colorFromHex(draft.bottomFill),
    statusInk: colorFromHex(draft.statusInk), homeInk: colorFromHex(draft.homeInk),
  });
  const browserDesktopChrome = create(BrowserDesktopChromeSchema, {
    windowTitle: draft.windowTitle, addressText: draft.addressText, showWindowControls: draft.showWindowControls,
    showToolbar: draft.showToolbar, appearance: draft.browserAppearance, window, showMenuBar: draft.showMenuBar,
    showDock: draft.showDock, desktopColor: colorFromHex(draft.desktopColor), windowColor: colorFromHex(draft.windowColor),
    chromeColor: colorFromHex(draft.chromeColor),
  });
  const init = {
    ...base,
    sourceMediaId, captureArea: draft.captureArea, kind: draft.kind, fitMode: draft.fitMode,
    viewportWidthPixels: base?.viewportWidthPixels ?? 0, viewportHeightPixels: base?.viewportHeightPixels ?? 0,
    matte: colorFromHex(draft.matteColor),
    phoneSystemChrome: draft.kind === PresentationKind.PHONE_SYSTEM ? phoneSystemChrome : undefined,
    browserDesktopChrome: draft.kind === PresentationKind.BROWSER_DESKTOP ? browserDesktopChrome : undefined,
  } as ScreenPresentation;
  return create(ScreenPresentationSchema, init);
}

export function screenPresentationDraftInvalid(draft: ScreenPresentationDraft) {
  return draft.kind === PresentationKind.BROWSER_DESKTOP && draft.showWindowControls && !draft.showToolbar;
}

export function ScreenPresentationEditor({ value, disabled, onChange }: {
  value: ScreenPresentationDraft; disabled: boolean; onChange: (patch: Partial<ScreenPresentationDraft>) => void;
}) {
  const { t } = useTranslation();
  const select = <K extends keyof ScreenPresentationDraft>(key: K, value: ScreenPresentationDraft[K]) => onChange({ [key]: value } as Partial<ScreenPresentationDraft>);
  const numberField = (label: string, key: keyof ScreenPresentationDraft) => <label className="shot-field" key={key}><span>{label}</span><input type="number" inputMode="decimal" value={String(value[key])} disabled={disabled} onChange={(event) => select(key, event.target.value as never)} /></label>;
  const colorField = (label: string, key: keyof ScreenPresentationDraft) => <label className="shot-field" key={key}><span>{label}</span><input type="text" inputMode="text" autoCapitalize="characters" maxLength={7} placeholder={t(strings.shot.hexPlaceholder)} value={String(value[key])} disabled={disabled} aria-invalid={String(value[key]) !== "" && !/^#?[\da-f]{6}$/i.test(String(value[key]))} onChange={(event) => select(key, event.target.value as never)} /></label>;
  const checks = (fields: Array<[keyof ScreenPresentationDraft, string]>) => fields.map(([key, label]) => <label className="shot-check" key={key}><input type="checkbox" checked={Boolean(value[key])} disabled={disabled} onChange={(event) => select(key, event.target.checked as never)} />{label}</label>);
  const invalid = screenPresentationDraftInvalid(value);
  return <div className="shot-fields">
    <label className="shot-field"><span>{t(strings.shot.captureArea)}</span><select value={value.captureArea} disabled={disabled} onChange={(event) => select("captureArea", Number(event.target.value))}><option value={CaptureArea.UNSPECIFIED}>{t(strings.shot.unspecified)}</option><option value={CaptureArea.APP_ONLY}>{t(strings.shot.appOnly)}</option><option value={CaptureArea.ALREADY_COMPOSED}>{t(strings.shot.alreadyComposed)}</option></select></label>
    <label className="shot-field"><span>{t(strings.shot.presentationKind)}</span><select value={value.kind} disabled={disabled} onChange={(event) => select("kind", Number(event.target.value))}><option value={PresentationKind.UNSPECIFIED}>{t(strings.shot.unspecified)}</option><option value={PresentationKind.PLAIN}>{t(strings.shot.plainCapture)}</option><option value={PresentationKind.PHONE_SYSTEM}>{t(strings.shot.phoneSystem)}</option><option value={PresentationKind.BROWSER_DESKTOP}>{t(strings.shot.browserDesktop)}</option></select></label>
    <label className="shot-field"><span>{t(strings.shot.fitMode)}</span><select value={value.fitMode} disabled={disabled} onChange={(event) => select("fitMode", Number(event.target.value))}><option value={ScreenFitMode.UNSPECIFIED}>{t(strings.shot.unspecified)}</option><option value={ScreenFitMode.CONTAIN}>{t(strings.shot.contain)}</option><option value={ScreenFitMode.COVER}>{t(strings.shot.cover)}</option></select></label>
    <details><summary>{t(strings.shot.advancedPresentation)}</summary><div className="shot-fields">{colorField(t(strings.shot.matteColor), "matteColor")}<p className="shot-help">{t(strings.shot.optionalHexHelp)}</p></div></details>
    {value.kind === PresentationKind.PHONE_SYSTEM && <details><summary>{t(strings.shot.phoneChrome)}</summary><div className="shot-fields">
      <label className="shot-field"><span>{t(strings.shot.appearance)}</span><select value={value.phoneAppearance} disabled={disabled} onChange={(event) => select("phoneAppearance", Number(event.target.value))}><option value={PresentationAppearance.AUTO}>{t(strings.shot.appearanceAuto)}</option><option value={PresentationAppearance.UNSPECIFIED}>{t(strings.shot.appearanceDefault)}</option><option value={PresentationAppearance.LIGHT}>{t(strings.shot.appearanceLight)}</option><option value={PresentationAppearance.DARK}>{t(strings.shot.appearanceDark)}</option></select></label>
      {checks([["showStatus", t(strings.shot.statusBar)], ["showBatteryPercent", t(strings.shot.batteryLabel)]])}
      <div className="shot-fields shot-fields--pair">{colorField(t(strings.shot.topFill), "topFill")}{colorField(t(strings.shot.bottomFill), "bottomFill")}</div>
      <div className="shot-fields shot-fields--pair">{colorField(t(strings.shot.statusInk), "statusInk")}{colorField(t(strings.shot.homeInk), "homeInk")}</div>
      <label className="shot-field"><span>{t(strings.shot.carrier)}</span><input value={value.carrierText} disabled={disabled} onChange={(event) => select("carrierText", event.target.value)} /></label>
      <label className="shot-field"><span>{t(strings.shot.clock)}</span><input value={value.clockText} disabled={disabled} onChange={(event) => select("clockText", event.target.value)} /></label>
      {numberField(t(strings.shot.battery), "batteryPercent")}{checks([["showCellularIndicator", t(strings.shot.cellular)], ["showWifiIndicator", t(strings.shot.wifi)], ["showHomeIndicator", t(strings.shot.homeIndicator)]])}
    </div></details>}
    {value.kind === PresentationKind.BROWSER_DESKTOP && <details><summary>{t(strings.shot.browserChrome)}</summary><div className="shot-fields">
      <label className="shot-field"><span>{t(strings.shot.appearance)}</span><select value={value.browserAppearance} disabled={disabled} onChange={(event) => select("browserAppearance", Number(event.target.value))}><option value={PresentationAppearance.UNSPECIFIED}>{t(strings.shot.appearanceDefault)}</option><option value={PresentationAppearance.LIGHT}>{t(strings.shot.appearanceLight)}</option><option value={PresentationAppearance.DARK}>{t(strings.shot.appearanceDark)}</option></select></label>
      <div className="shot-fields shot-fields--pair">{numberField(t(strings.shot.windowX), "windowX")}{numberField(t(strings.shot.windowY), "windowY")}{numberField(t(strings.shot.windowWidth), "windowWidth")}{numberField(t(strings.shot.windowHeight), "windowHeight")}</div><p className="shot-help">{t(strings.shot.windowRectHelp)}</p>
      <div className="shot-fields shot-fields--pair">{colorField(t(strings.shot.desktopColor), "desktopColor")}{colorField(t(strings.shot.windowColor), "windowColor")}{colorField(t(strings.shot.chromeColor), "chromeColor")}</div>
      <label className="shot-field"><span>{t(strings.shot.windowTitle)}</span><input value={value.windowTitle} disabled={disabled} onChange={(event) => select("windowTitle", event.target.value)} /></label>
      <label className="shot-field"><span>{t(strings.shot.addressText)}</span><input value={value.addressText} disabled={disabled} onChange={(event) => select("addressText", event.target.value)} /></label>
      {checks([["showWindowControls", t(strings.shot.windowControls)], ["showToolbar", t(strings.shot.toolbar)], ["showMenuBar", t(strings.shot.menuBar)], ["showDock", t(strings.shot.dock)]])}
      {invalid && <p role="alert">{t(strings.shot.browserControlsRequireToolbar)}</p>}
    </div></details>}
  </div>;
}
