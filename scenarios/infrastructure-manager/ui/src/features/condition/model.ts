import {
  BandVerdict as BandVerdictEnum,
  TrustVerdict as TrustVerdictEnum,
} from "@vrooli/proto-types/infrastructure-manager/v1/condition/condition_pb";

import { BAND_VERDICTS, TRUST_VERDICTS, type BandVerdict, type TrustVerdict } from "../../theme/instrument";

/**
 * Condition-verdict model helpers.
 *
 * Verdicts are resolved through the generated enum rather than by array
 * position: a positional list silently yields `undefined` for any value added
 * to the proto later, which renders as a blank token instead of a verdict.
 *
 * Both fallbacks fail toward distrust. An unrecognised trust verdict is
 * UNTRUSTED and an unrecognised band verdict is NOT_EVALUATED, because a
 * verdict this build does not understand is exactly the reading nobody should
 * be told is fine.
 *
 * They live in the feature's model rather than in its page because the board
 * grades the same readings the detail page does, and two independent
 * derivations of "is this reading trusted" is how two surfaces start
 * disagreeing about the same response.
 */
export function trustName(value: number): TrustVerdict {
  const name = (TrustVerdictEnum[value] || "").replace("TRUST_VERDICT_", "");
  return name in TRUST_VERDICTS ? (name as TrustVerdict) : "UNTRUSTED";
}

export function bandName(value: number): BandVerdict {
  const name = (BandVerdictEnum[value] || "").replace("BAND_VERDICT_", "");
  return name in BAND_VERDICTS ? (name as BandVerdict) : "NOT_EVALUATED";
}
