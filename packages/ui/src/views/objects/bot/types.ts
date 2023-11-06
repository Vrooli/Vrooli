import { User } from "@local/shared";
import { FormProps } from "forms/types";
import { BotShape } from "utils/shape/models/bot";
import { UpsertProps } from "../types";

export type BotUpsertProps = UpsertProps<User>
export type BotFormProps = FormProps<User, BotShape>
