import type { TFunction } from "i18next";
import { strings } from "../../consts/strings";

/** A sensitivity label as the operator reads it. Labels from a newer Hub stay readable as they are. */
export function contentLabel(label: string, t: TFunction): string {
  switch (label) {
    case "public": return t(strings.reach.channel.labels.public);
    case "private": return t(strings.reach.channel.labels.private);
    case "sensitive": return t(strings.reach.channel.labels.sensitive);
    case "critical": return t(strings.reach.channel.labels.critical);
    default: return label;
  }
}
