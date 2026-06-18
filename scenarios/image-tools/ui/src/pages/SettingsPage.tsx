import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ModelDefaultsCard } from "../features/models/ModelDefaultsCard";
import { useSettings } from "../features/settings/SettingsProvider";
import {
  FONT_SCALES,
  HANDEDNESS_CHOICES,
  REDUCED_MOTION_CHOICES,
  TEXT_DIRECTION_CHOICES,
  type FontScale,
  type Handedness,
  type ReducedMotion,
  type TextDirection,
} from "../features/settings/useSettings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

// `satisfies` (not `: Record<…, string>`) so the literal key-path types are
// preserved — `t()` is typed against the union of catalog key paths, and a
// widened `string` would not satisfy it.
const FONT_SCALE_LABEL = {
  small: strings.settings.fontScale.small,
  default: strings.settings.fontScale.default,
  large: strings.settings.fontScale.large,
  xlarge: strings.settings.fontScale.xlarge,
} satisfies Record<FontScale, string>;

const REDUCED_MOTION_LABEL = {
  system: strings.settings.reducedMotion.system,
  always: strings.settings.reducedMotion.always,
  never: strings.settings.reducedMotion.never,
} satisfies Record<ReducedMotion, string>;

const TEXT_DIRECTION_LABEL = {
  auto: strings.settings.textDirection.auto,
  ltr: strings.settings.textDirection.ltr,
  rtl: strings.settings.textDirection.rtl,
} satisfies Record<TextDirection, string>;

const HANDEDNESS_LABEL = {
  left: strings.settings.handedness.left,
  right: strings.settings.handedness.right,
} satisfies Record<Handedness, string>;

interface RadioGroupProps<T extends string> {
  /** Already-translated group label (drives `aria-label`). */
  label: string;
  options: readonly T[];
  value: T;
  onChange: (value: T) => void;
  /** Already-translated label per option. */
  labelFor: (option: T) => string;
  /** Stable per-option test id. */
  testIdFor: (option: T) => string;
}

/**
 * The shared radio-group pattern already used for theme/locale below, lifted to
 * one component so each new display preference renders identically. Copy is
 * passed in pre-translated so the control stays i18n-clean.
 */
function SettingsRadioGroup<T extends string>({
  label,
  options,
  value,
  onChange,
  labelFor,
  testIdFor,
}: RadioGroupProps<T>) {
  return (
    <div role="radiogroup" aria-label={label} className="flex flex-wrap gap-2">
      {options.map((option) => (
        <button
          key={option}
          type="button"
          role="radio"
          aria-checked={value === option}
          onClick={() => onChange(option)}
          data-testid={testIdFor(option)}
          className={
            value === option
              ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
              : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
          }
        >
          {labelFor(option)}
        </button>
      ))}
    </div>
  );
}

/**
 * Settings page. Surfaces the locale and theme selectors as a real page (in
 * addition to the compact controls in the top bar), plus client-persisted
 * display & accessibility preferences (text size, motion, text direction,
 * handedness) that take effect immediately via `<html>` attributes.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const { settings, setSettings } = useSettings();

  return (
    <section
      data-testid={selectors.pages.settings}
      aria-labelledby="settings-heading"
      className="flex flex-col gap-6"
    >
      <h2 id="settings-heading" className="text-2xl font-semibold">
        {t(strings.pages.settings.title)}
      </h2>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.themeHeading)}
        </h3>
        <SettingsRadioGroup
          label={t(strings.theme.switcherLabel)}
          options={THEME_CHOICES}
          value={choice}
          onChange={setTheme}
          labelFor={(c) => t(strings.theme.choice[c])}
          testIdFor={(c) => selectors.settingsPage.themeOption({ choice: c })}
        />
      </div>

      <div className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.localeHeading)}
        </h3>
        <div role="radiogroup" aria-label={t(strings.locale.switcherLabel)} className="flex gap-2">
          {SUPPORTED_LOCALES.map((lng) => (
            <button
              key={lng}
              type="button"
              role="radio"
              aria-checked={currentLocale === lng}
              onClick={() => void setLocale(lng)}
              data-testid={selectors.settingsPage.localeOption({ code: lng })}
              className={
                currentLocale === lng
                  ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {getLocaleConfig(lng).nativeLabel}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-4">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.settings.displayHeading)}
        </h3>

        <div className="flex flex-col gap-2">
          <span className="text-sm text-app-foreground">{t(strings.settings.fontScale.label)}</span>
          <SettingsRadioGroup
            label={t(strings.settings.fontScale.label)}
            options={FONT_SCALES}
            value={settings.fontScale}
            onChange={(fontScale) => setSettings({ fontScale })}
            labelFor={(c) => t(FONT_SCALE_LABEL[c])}
            testIdFor={(c) => selectors.settingsPage.fontScaleOption({ choice: c })}
          />
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm text-app-foreground">{t(strings.settings.reducedMotion.label)}</span>
          <p className="text-xs text-app-muted-foreground">
            {t(strings.settings.reducedMotion.description)}
          </p>
          <SettingsRadioGroup
            label={t(strings.settings.reducedMotion.label)}
            options={REDUCED_MOTION_CHOICES}
            value={settings.reducedMotion}
            onChange={(reducedMotion) => setSettings({ reducedMotion })}
            labelFor={(c) => t(REDUCED_MOTION_LABEL[c])}
            testIdFor={(c) => selectors.settingsPage.reducedMotionOption({ choice: c })}
          />
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm text-app-foreground">{t(strings.settings.textDirection.label)}</span>
          <SettingsRadioGroup
            label={t(strings.settings.textDirection.label)}
            options={TEXT_DIRECTION_CHOICES}
            value={settings.textDirection}
            onChange={(textDirection) => setSettings({ textDirection })}
            labelFor={(c) => t(TEXT_DIRECTION_LABEL[c])}
            testIdFor={(c) => selectors.settingsPage.textDirectionOption({ choice: c })}
          />
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm text-app-foreground">{t(strings.settings.handedness.label)}</span>
          <p className="text-xs text-app-muted-foreground">
            {t(strings.settings.handedness.description)}
          </p>
          <SettingsRadioGroup
            label={t(strings.settings.handedness.label)}
            options={HANDEDNESS_CHOICES}
            value={settings.handedness}
            onChange={(handedness) => setSettings({ handedness })}
            labelFor={(c) => t(HANDEDNESS_LABEL[c])}
            testIdFor={(c) => selectors.settingsPage.handednessOption({ choice: c })}
          />
        </div>
      </div>

      <ModelDefaultsCard />
    </section>
  );
}
