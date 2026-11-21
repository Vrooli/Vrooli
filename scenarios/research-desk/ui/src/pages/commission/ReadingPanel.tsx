import type { TFunction } from "i18next";
import { Alert } from "@vrooli/react-component-library/Alert/1";

import type { GetCommissionResponse, ReadSet } from "../../api/activity";
import { FileList } from "../../components/Changes";
import { Fact } from "../../components/Fact";
import { Facts, Section } from "../../components/Section";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { kb, tokens } from "../../lib/format";

/**
 * What each agent must read to start: the supervisor at start and at
 * each unit, and every owner. Sizes in KB and estimated tokens; flags in the
 * API's own words when a set is over its ceiling or the charter has grown.
 */
export function ReadingPanel({ data }: { data: GetCommissionResponse }) {
  const { t } = useTranslation();
  if (data.readSets.length === 0) return <p className="text-body-sm">{t(strings.reading.none)}</p>;
  return (
    <div className="flex flex-col gap-space-lg">
      <Facts>
        <Fact label={t(strings.reading.ceilings)} evidence={data.ceilings}>
          {data.ceilings?.known ? t(strings.reading.ceilingsKnown) : t(strings.reading.ceilingsUnknown)}
        </Fact>
      </Facts>
      <ReadSets sets={data.readSets} />
    </div>
  );
}

/** Each read set with its size, a bar against the largest, its flags and its files; also used by Preflight for owners. */
export function ReadSets({ sets }: { sets: ReadSet[] }) {
  const largest = Math.max(...sets.map((set) => Number(set.bytes)), 1);
  return (
    <>
      {sets.map((set) => (
        <ReadSetView key={set.role} set={set} largest={largest} />
      ))}
    </>
  );
}

function roleLabel(role: string, t: TFunction): string {
  if (role === "scheduler-start") return t(strings.reading.schedulerStart);
  if (role === "scheduler-unit") return t(strings.reading.schedulerUnit);
  if (role.startsWith("owner-start:")) return t(strings.reading.ownerStart, { topic: role.slice("owner-start:".length) });
  return role;
}

function ReadSetView({ set, largest }: { set: ReadSet; largest: number }) {
  const { t } = useTranslation();
  const share = Math.max(2, Math.round((Number(set.bytes) / largest) * 100));
  return (
    <Section id={`read-${set.role.replace(/[^a-z0-9-]/gi, "-")}`} title={roleLabel(set.role, t)}>
      <p className="text-body">
        {t(strings.reading.size, { kb: kb(set.bytes), tokens: tokens(set.bytes) })}
      </p>
      <div className={set.flags.length > 0 ? "rd-bar rd-bar-over" : "rd-bar"} aria-hidden="true">
        <span style={{ inlineSize: `${share}%` }} />
      </div>
      {set.flags.map((flag) => (
        <Alert key={flag} tone="warning" title={flag} />
      ))}
      <details className="rd-fold">
        <summary>{t(strings.reading.files, { count: set.files.length })}</summary>
        <FileList files={set.files} />
      </details>
    </Section>
  );
}
