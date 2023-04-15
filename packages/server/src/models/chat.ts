import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'Chat' as const;
const suppFields = ['you'] as const;
export const ChatModel: ModelLogic<any, any> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})