import { type ProfileShape, type User } from "@local/shared";
import { type FormProps, type ObjectViewProps } from "../../../types.js";

export type UserFormProps = FormProps<User, ProfileShape>
export type UserViewProps = ObjectViewProps<User>
