import { BotShape, User } from "@local/shared";
import { FormProps } from "forms/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type BotUpsertPropsPage = CrudPropsPage;
type BotUpsertPropsDialog = CrudPropsDialog<User>;
export type BotUpsertProps = BotUpsertPropsPage | BotUpsertPropsDialog;
export type BotFormProps = FormProps<User, BotShape>
