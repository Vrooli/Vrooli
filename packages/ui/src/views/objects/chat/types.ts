import { AssistantTask } from "types";
import { ChatShape } from "utils/shape/models/chat";
import { CrudProps } from "../types";

export type ChatCrudProps = CrudProps<ChatShape> & {
    context?: string | null | undefined;
    task?: AssistantTask;
}
