import { strings } from "../../consts/strings";
import { VerdictGrade } from "../../api/models";

/** The translation key that names a grade in plain words. */
export type VerdictLabelKey = (typeof strings.models.verdict)[
  | "recommended"
  | "works"
  | "experimental"
  | "untested"
  | "notRecommended"
  | "broken"];

/** How a grade reads at a glance. Meaning rides on the word, never colour alone. */
export interface GradeView {
  label: VerdictLabelKey;
  /** Token-paired chip classes. */
  className: string;
}

const VIEW: Record<VerdictGrade, GradeView> = {
  [VerdictGrade.RECOMMENDED]: { label: strings.models.verdict.recommended, className: "border border-app-success/40 bg-app-success/10 text-app-success" },
  [VerdictGrade.WORKS]: { label: strings.models.verdict.works, className: "border border-app-info/40 bg-app-info/10 text-app-info" },
  [VerdictGrade.EXPERIMENTAL]: { label: strings.models.verdict.experimental, className: "border border-app-warning/40 bg-app-warning/10 text-app-warning" },
  [VerdictGrade.UNTESTED]: { label: strings.models.verdict.untested, className: "border border-app-border bg-app-surface-muted text-app-muted-foreground" },
  [VerdictGrade.NOT_RECOMMENDED]: { label: strings.models.verdict.notRecommended, className: "border border-app-danger/40 bg-app-danger/10 text-app-danger" },
  [VerdictGrade.BROKEN]: { label: strings.models.verdict.broken, className: "border border-app-danger/60 bg-app-danger/20 font-semibold text-app-danger" },
  // A response from an older server carries no grade: say "untested", never blank.
  [VerdictGrade.UNSPECIFIED]: { label: strings.models.verdict.untested, className: "border border-app-border bg-app-surface-muted text-app-muted-foreground" },
};

/** gradeView returns the label key and classes for a grade. */
export const gradeView = (grade: VerdictGrade | undefined): GradeView =>
  // A grade a newer server adds is not in the table: it reads as untested too.
  (VIEW as Partial<Record<VerdictGrade, GradeView>>)[grade ?? VerdictGrade.UNSPECIFIED] ?? VIEW[VerdictGrade.UNSPECIFIED];
