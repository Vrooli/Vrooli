import { bookmarkListValidation } from "@shared/validation";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'NodeRoutineList' as const;

const suppFields = [] as const;
export const BookmarkListModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.bookmark_list,
    display: {
        select: () => ({ id: true, label: true }),
        label: (select) => select.label,
    },
    format: {
        gqlRelMap: {
            __typename,
            bookmarks: 'Bookmark',
        },
        prismaRelMap: {
            __typename,
            bookmarks: 'Bookmark',
        },
        countFields: {
            bookmarksCount: true,
        },
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any),
            update: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any)
        },
        yup: bookmarkListValidation,
    },
    validate: {} as any,// TODO
})