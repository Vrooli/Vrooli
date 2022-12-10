import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

const __typename = 'Post' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.postSelect,
    Prisma.postGetPayload<SelectWrap<Prisma.postSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const PostModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.post,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})