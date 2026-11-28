import {
  RelevantContextKind,
  RelevantContextRepeatPolicy,
  type RelevantContextItem,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

export const contextKindLabels: Record<RelevantContextKind, string> = {
  [RelevantContextKind.UNSPECIFIED]: "Context",
  [RelevantContextKind.SKILL]: "Skill",
  [RelevantContextKind.DOC]: "Doc",
  [RelevantContextKind.COMMAND]: "Command",
  [RelevantContextKind.SEARCH]: "Search",
  [RelevantContextKind.CODE_REF]: "Code",
  [RelevantContextKind.REQ_REF]: "Requirement",
  [RelevantContextKind.NOTE]: "Note",
};

export function contextKindLabel(kind: RelevantContextKind, casing: "title" | "lower" = "title") {
  const label = contextKindLabels[kind];
  return casing === "lower" ? label.toLowerCase() : label;
}

export function repeatLabel(policy: RelevantContextRepeatPolicy) {
  switch (policy) {
    case RelevantContextRepeatPolicy.ON_RESUME:
      return "on resume";
    case RelevantContextRepeatPolicy.EVERY_PHASE:
      return "every phase";
    case RelevantContextRepeatPolicy.PHASE_ENTRY:
      return "phase entry";
    case RelevantContextRepeatPolicy.AS_NEEDED:
      return "as needed";
    case RelevantContextRepeatPolicy.ONCE_PER_EXECUTION:
      return "once";
    default:
      return "";
  }
}

export function contextCommand(item: RelevantContextItem) {
  if (item.argv.length > 0) return item.argv.join(" ");
  return item.command;
}
