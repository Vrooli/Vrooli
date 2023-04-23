import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = "ChatInvite" as const;
const suppFields = ["you"] as const;
export const ChatInviteModel: ModelLogic<any, any> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat_invite,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
});
