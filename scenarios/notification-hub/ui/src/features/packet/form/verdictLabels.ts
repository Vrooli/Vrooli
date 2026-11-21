import { strings } from "../../../consts/strings";
import type { SectionVerdict } from "../model/questions";

/** Catalog keys of the verdict names, written out so each one has a visible call site. */
export const VERDICT_LABEL = {
  approve: strings.packet.form.verdicts.approve,
  change: strings.packet.form.verdicts.change,
  drop: strings.packet.form.verdicts.drop,
} as const satisfies Record<SectionVerdict, string>;
