import { FormProps } from "forms/types";
import { AssistantTask } from "types";
import { ChatShape } from "utils/shape/models/chat";
import { CrudProps } from "../types";

export type ChatCrudProps = CrudProps<ChatShape> & {
    context?: string | null | undefined;
    task?: AssistantTask;
}
export type ChatFormProps = FormProps<ChatShape, ChatShape> & Pick<ChatCrudProps, "context" | "task">
