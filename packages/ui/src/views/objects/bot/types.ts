import { type BotShape, type User } from "@vrooli/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps } from "../../../types.js";

type BotUpsertPropsPage = CrudPropsPage;
type BotUpsertPropsDialog = CrudPropsDialog<User>;
type BotUpsertPropsPartial = CrudPropsPartial<User>;
export type BotUpsertProps = BotUpsertPropsPage | BotUpsertPropsDialog | BotUpsertPropsPartial;
export type BotFormProps = FormProps<User, BotShape>
