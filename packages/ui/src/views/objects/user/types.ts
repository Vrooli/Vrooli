import { User } from "@local/shared";
import { FormProps } from "forms/types";
import { ProfileShape } from "utils/shape/models/profile";
import { ObjectViewProps } from "views/types";

export type UserFormProps = FormProps<User, ProfileShape>
export type UserViewProps = ObjectViewProps<User>
