import { BotShape, User } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps } from "../../../types.js";

type BotUpsertPropsPage = CrudPropsPage;
type BotUpsertPropsDialog = CrudPropsDialog<User>;
export type BotUpsertProps = BotUpsertPropsPage | BotUpsertPropsDialog;
export type BotFormProps = FormProps<User, BotShape>
