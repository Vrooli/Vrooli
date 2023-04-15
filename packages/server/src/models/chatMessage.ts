import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'ChatMessage' as const;
const suppFields = ['you'] as const;
export const ChatMessageModel: ModelLogic<any, any> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat_message,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})