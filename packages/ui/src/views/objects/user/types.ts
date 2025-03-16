import { ProfileShape, User } from "@local/shared";
import { FormProps, ObjectViewProps } from "../../../types.js";

export type UserFormProps = FormProps<User, ProfileShape>
export type UserViewProps = ObjectViewProps<User>
