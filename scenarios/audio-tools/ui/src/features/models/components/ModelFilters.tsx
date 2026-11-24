import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { Panel } from "../../../components/ui/panel";
import { Select } from "../../../components/ui/select";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import { CAPABILITIES, capabilityLabel } from "../modelDisplay";
import type { ModelFiltersState } from "../types";

interface Props {
  value: ModelFiltersState;
  onChange: (next: ModelFiltersState) => void;
}

/**
 * Server-side filter controls. Every change is forwarded to
 * ListModelsRequest by the page; this component never filters client-side.
 */
export function ModelFilters({ value, onChange }: Props) {
  const { t } = useTranslation();

  function update(patch: Partial<ModelFiltersState>) {
    onChange({ ...value, ...patch });
  }

  return (
    <Panel
      title={t(strings.models.filtersTitle)}
      aria-label={t(strings.models.filtersTitle)}
    >
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div className="flex flex-col gap-1">
          <label
            htmlFor="models-filter-capability"
            className="text-xs font-medium text-app-muted-foreground"
          >
            {t(strings.models.capabilityLabel)}
          </label>
          <Select
            id="models-filter-capability"
            value={value.capability}
            onChange={(event) => update({ capability: event.target.value })}
          >
            <option value="">{t(strings.models.allCapabilities)}</option>
            {CAPABILITIES.map((capability) => (
              <option key={capability} value={capability}>
                {capabilityLabel(capability, t)}
              </option>
            ))}
          </Select>
        </div>

        <div className="flex flex-col gap-1">
          <label
            htmlFor="models-filter-language"
            className="text-xs font-medium text-app-muted-foreground"
          >
            {t(strings.models.languageLabel)}
          </label>
          <Input
            id="models-filter-language"
            value={value.language}
            placeholder={t(strings.models.languagePlaceholder)}
            aria-describedby="models-filter-language-help"
            onChange={(event) => update({ language: event.target.value })}
          />
          <p
            id="models-filter-language-help"
            className="text-xs text-app-muted-foreground"
          >
            {t(strings.models.languageHelp)}
          </p>
        </div>

        <div className="flex items-end pb-2">
          <label
            htmlFor="models-filter-commercial"
            className="flex items-center gap-2 text-sm text-app-foreground"
          >
            <input
              id="models-filter-commercial"
              type="checkbox"
              checked={value.commercialOnly}
              onChange={(event) => update({ commercialOnly: event.target.checked })}
              className="h-4 w-4 rounded border-app-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus/50"
            />
            {t(strings.models.commercialOnlyLabel)}
          </label>
        </div>

        <div className="flex items-end pb-2">
          <label
            htmlFor="models-filter-ready"
            className="flex items-center gap-2 text-sm text-app-foreground"
          >
            <input
              id="models-filter-ready"
              type="checkbox"
              checked={value.onlyReady}
              onChange={(event) => update({ onlyReady: event.target.checked })}
              className="h-4 w-4 rounded border-app-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus/50"
            />
            {t(strings.models.onlyReadyLabel)}
          </label>
        </div>
      </div>
    </Panel>
  );
}
