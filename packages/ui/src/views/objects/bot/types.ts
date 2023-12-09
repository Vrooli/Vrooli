import { User } from "@local/shared";
import { FormProps } from "forms/types";
import { BotShape } from "utils/shape/models/bot";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type BotUpsertPropsPage = CrudPropsPage;
type BotUpsertPropsDialog = CrudPropsDialog<User>;
export type BotUpsertProps = BotUpsertPropsPage | BotUpsertPropsDialog;
export type BotFormProps = FormProps<User, BotShape>
