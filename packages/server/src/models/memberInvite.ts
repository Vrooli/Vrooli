import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { MemberModel } from "./member";
import { Displayer } from "./types";

const __typename = 'MemberInvite' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.member_inviteSelect,
    Prisma.member_inviteGetPayload<SelectWrap<Prisma.member_inviteSelect>>
> => ({
    select: () => ({ id: true, member: { select: MemberModel.display.select() } }),
    // Label is the member label
    label: (select, languages) => MemberModel.display.label(select.member as any, languages),
})

export const MemberInviteModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.member_invite,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})