import { FormProps } from "forms/types";
import { ChatShape } from "utils/shape/models/chat";
import { CrudProps } from "../types";

export type ChatCrudProps = CrudProps<ChatShape> & {
    context?: string | null | undefined;
    task?: string;
}
export type ChatFormProps = FormProps<ChatShape, ChatShape> & Pick<ChatCrudProps, "context" | "task">
