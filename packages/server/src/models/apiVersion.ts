import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiVersion } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { Displayer, Formatter } from "./types";

const __typename = 'ApiVersion' as const;

const suppFields = ['permissionsVersion'] as const;
const formatter = (): Formatter<ApiVersion, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        comments: 'Comment',
        directoryListings: 'ProjectVersionDirectory',
        forks: 'ApiVersion',
        reports: 'Report',
        resourceList: 'ResourceList',
        root: 'Api',
    },
    countFields: ['commentsCount', 'directoryListingsCount', 'forksCount', 'reportsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['permissionsVersion', async () => await getSingleTypePermissions(__typename, ids, prisma, userData)],
        ],
    },
})

const displayer = (): Displayer<
    Prisma.api_versionSelect,
    Prisma.api_versionGetPayload<SelectWrap<Prisma.api_versionSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => {
        // Return name if exists, or callLink host
        const name = bestLabel(select.translations, 'name', languages)
        if (name.length > 0) return name
        const url = new URL(select.callLink)
        return url.host
    },
})

export const ApiVersionModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api_version,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})