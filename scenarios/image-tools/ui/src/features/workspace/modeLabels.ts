import { strings } from "../../consts/strings";
import type { WorkspaceMode } from "./useWorkspace";

/** i18n key for each Workspace mode's display label. Shared by the mode
 * switcher and the inspector so the two never drift. */
export const MODE_LABEL: Record<
  WorkspaceMode,
  (typeof strings.workspace.mode)[keyof typeof strings.workspace.mode]
> = {
  edit: strings.workspace.mode.edit,
  enhance: strings.workspace.mode.enhance,
  create: strings.workspace.mode.create,
  analyze: strings.workspace.mode.analyze,
};
