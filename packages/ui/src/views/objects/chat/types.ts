import { ChatShape } from "@local/shared";
import { CrudProps, FormProps } from "../../../types";

export type ChatCrudProps = CrudProps<ChatShape> & {
    context?: string | null | undefined;
    task?: string;
}
export type ChatFormProps = FormProps<ChatShape, ChatShape> & Pick<ChatCrudProps, "context" | "task">
