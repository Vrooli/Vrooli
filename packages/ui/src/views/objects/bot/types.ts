import { BotShape, User } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps } from "../../../types.js";

type BotUpsertPropsPage = CrudPropsPage;
type BotUpsertPropsDialog = CrudPropsDialog<User>;
type BotUpsertPropsPartial = CrudPropsPartial<User>;
export type BotUpsertProps = BotUpsertPropsPage | BotUpsertPropsDialog | BotUpsertPropsPartial;
export type BotFormProps = FormProps<User, BotShape>
