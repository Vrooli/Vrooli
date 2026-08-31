import { useCallback, useEffect, useState } from "react";
import { Check, CopyCheck, RotateCcw, Settings2, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  useWorkspaceStore,
  useEffectiveFontSize,
  type AppearanceProperty,
} from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { DEFAULT_THEME_ID } from "../consts/config";
import { cn } from "../lib/classnames";
import { strings } from "../consts/strings";
import { Button } from "./ui/button";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/0";
import AppearancePreview from "./appearance/AppearancePreview";
import HeaderColorPicker from "./appearance/HeaderColorPicker";
import ThemePicker from "./appearance/ThemePicker";
import FontSizeStepper from "./appearance/FontSizeStepper";

/** Transient success feedback lifetime. Errors stay until the next action. */
const FEEDBACK_TTL_MS = 4000;

type ApplyFeedback =
  | { kind: "applied"; count: number }
  | { kind: "defaultSaved" }
  | { kind: "error" };

export default function AppearanceModal() {
  const { t } = useTranslation();
  const appearanceModalPane = useWorkspaceStore((s) => s.appearanceModalPane);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const panes = useWorkspaceStore((s) => s.panes);
  const setPaneColor = useWorkspaceStore((s) => s.setPaneColor);
  const setPaneTheme = useWorkspaceStore((s) => s.setPaneTheme);
  const setPaneFontSize = useWorkspaceStore((s) => s.setPaneFontSize);
  const setDeviceFontSize = useWorkspaceStore((s) => s.setDeviceFontSize);
  const clearDeviceFontSize = useWorkspaceStore((s) => s.clearDeviceFontSize);
  const applyAppearance = useWorkspaceStore((s) => s.applyAppearance);
  const defaultHeaderColor = useWorkspaceStore((s) => s.defaultHeaderColor);
  const defaultThemeId = useWorkspaceStore((s) => s.defaultThemeId);
  const defaultFontSize = useWorkspaceStore((s) => s.defaultFontSize);
  const setSettingsModalOpen = useWorkspaceStore((s) => s.setSettingsModalOpen);
  const setSettingsInitialTab = useWorkspaceStore((s) => s.setSettingsInitialTab);
  const { syncPaneUpdate, syncPaneUpdates } = useWorkspaceSync();

  const [selectedProps, setSelectedProps] = useState<Record<AppearanceProperty, boolean>>({
    headerColor: true,
    themeId: true,
    fontSize: true,
  });
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [feedback, setFeedback] = useState<ApplyFeedback | null>(null);

  useEffect(() => {
    if (!feedback || feedback.kind === "error") return;
    const timer = setTimeout(() => setFeedback(null), FEEDBACK_TTL_MS);
    return () => clearTimeout(timer);
  }, [feedback]);

  const pane = panes.find((p) => p.sessionId === appearanceModalPane);
  const sessionId = appearanceModalPane ?? "";
  const currentFontSize = useEffectiveFontSize(sessionId);

  const close = useCallback(() => setAppearanceModalPane(null), [setAppearanceModalPane]);

  if (!appearanceModalPane || !pane) return null;

  const paneSessionId = pane.sessionId;
  const currentColor = pane.headerColor;
  const currentThemeId = pane.themeId ?? DEFAULT_THEME_ID;
  const otherPaneIds = panes
    .filter((p) => p.sessionId !== sessionId)
    .map((p) => p.sessionId);

  const propertyList = (Object.keys(selectedProps) as AppearanceProperty[]).filter(
    (prop) => selectedProps[prop],
  );
  const noneSelected = propertyList.length === 0;

  const propertyLabels: Record<AppearanceProperty, string> = {
    headerColor: t(strings.appearance.applySection.propHeaderColor),
    themeId: t(strings.appearance.applySection.propTheme),
    fontSize: t(strings.appearance.applySection.propFontSize),
  };

  /** Server payload carrying only the selected properties. */
  const buildPayload = () => ({
    ...(selectedProps.headerColor ? { header_color: currentColor } : {}),
    ...(selectedProps.themeId ? { theme_id: currentThemeId } : {}),
    ...(selectedProps.fontSize ? { font_size: currentFontSize } : {}),
  });

  const handleApplyToOpen = async () => {
    setConfirmOpen(false);
    applyAppearance(sessionId, {
      properties: propertyList,
      toExistingPanes: true,
      asNewPaneDefault: false,
    });
    const failed = await syncPaneUpdates(otherPaneIds, buildPayload());
    setFeedback(
      failed.length > 0
        ? { kind: "error" }
        : { kind: "applied", count: otherPaneIds.length },
    );
  };

  const handleSetDefault = () => {
    applyAppearance(sessionId, {
      properties: propertyList,
      toExistingPanes: false,
      asNewPaneDefault: true,
    });
    setFeedback({ kind: "defaultSaved" });
  };

  const handleResetToDefaults = () => {
    clearDeviceFontSize(paneSessionId);
    setPaneColor(paneSessionId, defaultHeaderColor);
    setPaneTheme(paneSessionId, defaultThemeId);
    setPaneFontSize(paneSessionId, defaultFontSize);
    syncPaneUpdate(paneSessionId, {
      header_color: defaultHeaderColor,
      theme_id: defaultThemeId,
      font_size: defaultFontSize,
    });
  };

  const openDefaultsSettings = () => {
    close();
    setSettingsInitialTab("new-pane-defaults");
    setSettingsModalOpen(true);
  };

  const propChip = (prop: AppearanceProperty) => {
    const on = selectedProps[prop];
    return (
      <button
        key={prop}
        type="button"
        data-testid={`appearance-prop-${prop}`}
        aria-pressed={on}
        className={cn(
          "flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors",
          on
            ? "border-wc-accent bg-wc-accent/10 text-wc-text-primary"
            : "border-wc-default text-wc-text-muted hover:text-wc-text-primary",
        )}
        onClick={() => setSelectedProps((s) => ({ ...s, [prop]: !s[prop] }))}
      >
        <Check
          className={cn("h-3 w-3", on ? "text-wc-accent" : "opacity-30")}
          aria-hidden="true"
        />
        {propertyLabels[prop]}
      </button>
    );
  };

  const feedbackText = feedback
    ? feedback.kind === "applied"
      ? t(strings.appearance.applySection.appliedFeedback, { count: feedback.count })
      : feedback.kind === "defaultSaved"
        ? t(strings.appearance.applySection.defaultSavedFeedback)
        : t(strings.appearance.applySection.applyError)
    : null;

  return (
    <ResponsiveDialog
      // No keyboard avoidance: colour, theme and size controls only — no text entry, so there is
      // nothing for a keyboard to cover.
      avoidKeyboard={false}
      open
      onClose={close}
      size="md"
      closeLabel={t(strings.appearance.closeAriaLabel)}
      title={t(strings.appearance.title)}
      testId="appearance-modal"
    >
      <SettingsList className="p-4">
        <section>
          <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
            {t(strings.appearance.previewHeading)}
          </h3>
          <AppearancePreview
            headerColor={currentColor}
            themeId={currentThemeId}
            fontSize={currentFontSize}
            sessionName={pane.name}
          />
        </section>

        <SettingsList.Group>
          <HeaderColorPicker
            currentColor={currentColor}
            onSelectColor={(color) => {
              setPaneColor(paneSessionId, color);
              syncPaneUpdate(paneSessionId, { header_color: color });
            }}
            testIdPrefix="appearance"
          />
          <ThemePicker
            currentThemeId={currentThemeId}
            onSelectTheme={(themeId) => {
              setPaneTheme(paneSessionId, themeId);
              syncPaneUpdate(paneSessionId, { theme_id: themeId });
            }}
            testIdPrefix="appearance"
          />
          <FontSizeStepper
            currentSize={currentFontSize}
            onChangeSize={(size) => {
              setDeviceFontSize(paneSessionId, size);
              syncPaneUpdate(paneSessionId, { font_size: size });
            }}
            testIdPrefix="appearance"
          />
          <div className="border-t border-wc-default pt-3">
            <button
              type="button"
              data-testid="appearance-reset-defaults"
              className="flex items-center gap-1.5 text-xs font-medium text-wc-text-muted hover:text-wc-text-primary"
              onClick={handleResetToDefaults}
              title={t(strings.appearance.resetToDefaultsHint)}
            >
              <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
              {t(strings.appearance.resetToDefaults)}
            </button>
          </div>
        </SettingsList.Group>

        <SettingsList.Group className="text-[11px] text-wc-text-faint">
          <div>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">
              {t(strings.appearance.applySection.heading)}
            </h3>
            <p className="mt-1 text-[11px] text-wc-text-faint">
              {t(strings.appearance.applySection.description)}
            </p>
          </div>
          <div className="flex flex-wrap gap-1.5">
            {(["headerColor", "themeId", "fontSize"] as AppearanceProperty[]).map(propChip)}
          </div>
          {noneSelected && (
            <p className="text-[11px] text-wc-text-muted">
              {t(strings.appearance.applySection.noPropsHint)}
            </p>
          )}
          <div className="space-y-2">
            {otherPaneIds.length > 0 && (
              <Button
                data-testid="appearance-apply-open"
                variant="outline"
                className="w-full"
                disabled={noneSelected}
                onClick={() => setConfirmOpen(true)}
              >
                <CopyCheck className="h-4 w-4 me-2" aria-hidden="true" />
                {t(strings.appearance.applySection.applyToOpen, {
                  count: otherPaneIds.length,
                })}
              </Button>
            )}
            <Button
              data-testid="appearance-set-default"
              variant="outline"
              className="w-full"
              disabled={noneSelected}
              onClick={handleSetDefault}
            >
              <Sparkles className="h-4 w-4 me-2" aria-hidden="true" />
              {t(strings.appearance.applySection.setDefault)}
            </Button>
          </div>
          <div aria-live="polite">
            {feedbackText && (
              <p
                data-testid="appearance-apply-feedback"
                className={cn(
                  "text-[11px]",
                  feedback?.kind === "error" ? "text-wc-error-text" : "text-wc-text-muted",
                )}
              >
                {feedbackText}
              </p>
            )}
          </div>
          <button
            type="button"
            data-testid="appearance-manage-defaults"
            className="flex items-center gap-1.5 text-[11px] font-medium text-wc-text-muted hover:text-wc-text-primary"
            onClick={openDefaultsSettings}
          >
            <Settings2 className="h-3.5 w-3.5" aria-hidden="true" />
            {t(strings.appearance.applySection.manageDefaults)}
          </button>
        </SettingsList.Group>
      </SettingsList>

      <AlertDialog
        open={confirmOpen}
        title={t(strings.appearance.applySection.confirmTitle)}
        description={t(strings.appearance.applySection.confirmBody, {
          count: otherPaneIds.length,
          properties: propertyList.map((prop) => propertyLabels[prop]).join(", "),
        })}
        cancelLabel={t(strings.appearance.applySection.confirmCancel)}
        confirmLabel={t(strings.appearance.applySection.confirmApply)}
        onCancel={() => setConfirmOpen(false)}
        onConfirm={() => {
          void handleApplyToOpen();
        }}
        testIdPrefix="appearance-apply"
      />
    </ResponsiveDialog>
  );
}
