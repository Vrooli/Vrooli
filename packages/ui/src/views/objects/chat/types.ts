import { Chat } from "@local/shared";
import { AssistantTask } from "types";
import { CrudProps } from "../types";

export type ChatCrudProps = CrudProps<Chat> & {
    context?: string | null | undefined;
    task?: AssistantTask;
}
