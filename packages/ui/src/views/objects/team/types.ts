import { Team, TeamShape } from "@local/shared";
import { FormProps } from "forms/types";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type TeamUpsertPropsPage = CrudPropsPage;
type TeamUpsertPropsDialog = CrudPropsDialog<Team>;
export type TeamUpsertProps = TeamUpsertPropsPage | TeamUpsertPropsDialog;
export type TeamFormProps = FormProps<Team, TeamShape>
export type TeamViewProps = ObjectViewProps<Team>
