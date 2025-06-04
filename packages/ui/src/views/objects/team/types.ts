import { type Team, type TeamShape } from "@local/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

type TeamUpsertPropsPage = CrudPropsPage;
type TeamUpsertPropsDialog = CrudPropsDialog<Team>;
type TeamUpsertPropsPartial = CrudPropsPartial<Team>;
export type TeamUpsertProps = TeamUpsertPropsPage | TeamUpsertPropsDialog | TeamUpsertPropsPartial;
export type TeamFormProps = FormProps<Team, TeamShape>
export type TeamViewProps = ObjectViewProps<Team>
