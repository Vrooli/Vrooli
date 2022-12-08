import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.meetingSelect,
    Prisma.meetingGetPayload<SelectWrap<Prisma.meetingSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const MeetingModel = ({
    delegate: (prisma: PrismaType) => prisma.meeting,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Meeting' as GraphQLModelType,
    validate: {} as any,
})