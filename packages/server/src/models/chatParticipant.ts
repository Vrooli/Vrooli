import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'ChatParticipant' as const;
const suppFields = ['you'] as const;
export const ChatParticipantModel: ModelLogic<any, any> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat_participants,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})