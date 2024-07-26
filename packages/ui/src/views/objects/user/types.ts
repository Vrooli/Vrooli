import { ProfileShape, User } from "@local/shared";
import { FormProps } from "forms/types";
import { ObjectViewProps } from "views/types";

export type UserFormProps = FormProps<User, ProfileShape>
export type UserViewProps = ObjectViewProps<User>
