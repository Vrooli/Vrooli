import {
  CandidateOrigin,
  CandidateStatus,
} from "@vrooli/proto-types/brand-manager/v1/candidates/candidates_pb";

import { strings } from "../../consts/strings";

type StatusLabelKey =
  | typeof strings.logo.statusProposed
  | typeof strings.logo.statusPicked
  | typeof strings.logo.statusRejected
  | typeof strings.logo.statusSuperseded;

/** statusLabelKey maps a candidate status to its translated label key. */
export function statusLabelKey(status: CandidateStatus): StatusLabelKey {
  switch (status) {
    case CandidateStatus.PICKED:
      return strings.logo.statusPicked;
    case CandidateStatus.REJECTED:
      return strings.logo.statusRejected;
    case CandidateStatus.SUPERSEDED:
      return strings.logo.statusSuperseded;
    default:
      return strings.logo.statusProposed;
  }
}

/** originLabel renders the candidate origin as a short technical badge. */
export function originLabel(origin: CandidateOrigin): string {
  switch (origin) {
    case CandidateOrigin.IMPORTED:
      return "imported";
    case CandidateOrigin.EDITED:
      return "edited";
    case CandidateOrigin.OBJECT_REMOVED:
      return "object removed";
    case CandidateOrigin.BACKGROUND_REMOVED:
      return "background removed";
    case CandidateOrigin.VECTORIZED:
      return "vectorized";
    default:
      return "generated";
  }
}
