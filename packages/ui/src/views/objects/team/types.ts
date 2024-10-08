import { Team, TeamShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

type TeamUpsertPropsPage = CrudPropsPage;
type TeamUpsertPropsDialog = CrudPropsDialog<Team>;
export type TeamUpsertProps = TeamUpsertPropsPage | TeamUpsertPropsDialog;
export type TeamFormProps = FormProps<Team, TeamShape>
export type TeamViewProps = ObjectViewProps<Team>
