import { ProfileShape, User } from "@local/shared";
import { FormProps, ObjectViewProps } from "../../../types";

export type UserFormProps = FormProps<User, ProfileShape>
export type UserViewProps = ObjectViewProps<User>
