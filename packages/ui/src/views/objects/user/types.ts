import { type ProfileShape, type User } from "@vrooli/shared";
import { type FormProps, type ObjectViewProps } from "../../../types.js";

export type UserFormProps = FormProps<User, ProfileShape>
export type UserViewProps = ObjectViewProps<User>
