import { Panel } from "../../../components/ui/panel";
import { Table, TBody, TD, TH, THead, TR } from "../../../components/ui/table";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import type { ModelRow } from "../../../services/models";
import { CAPABILITIES, capabilityLabel } from "../modelDisplay";

interface Props {
  models: ModelRow[];
}

/**
 * FootprintPanel answers the operator question the registry already knows:
 * which model each capability will actually run, and how much weight disk it
 * needs. It deliberately shows only resolved defaults; alternatives remain in
 * the registry table, where their larger sizes are visible but not implied.
 */
export function FootprintPanel({ models }: Props) {
  const { t } = useTranslation();
  const defaults = CAPABILITIES.flatMap((capability) =>
    models.filter((model) => model.isDefault && model.capability === capability),
  );
  if (defaults.length === 0) {
    return null;
  }

  return (
    <Panel
      title={t(strings.models.footprintTitle)}
      description={t(strings.models.footprintDescription)}
      bodyless
      aria-label={t(strings.models.footprintTitle)}
    >
      <Table>
        <THead>
          <TR>
            <TH className="w-1/3">{t(strings.models.colCapability)}</TH>
            <TH className="w-1/2">{t(strings.models.colModel)}</TH>
            <TH>{t(strings.models.colSize)}</TH>
          </TR>
        </THead>
        <TBody>
          {defaults.map((model) => (
            <TR key={`${model.capability}:${model.id}`}>
              <TD>{capabilityLabel(model.capability, t)}</TD>
              <TD className="font-mono text-xs">{model.id}</TD>
              <TD className="text-xs text-app-muted-foreground">
                {model.sizeMbApprox > 0
                  ? t(strings.models.sizeMb, { mb: Math.round(model.sizeMbApprox) })
                  : t(strings.models.notEstablished)}
              </TD>
            </TR>
          ))}
        </TBody>
      </Table>
    </Panel>
  );
}
