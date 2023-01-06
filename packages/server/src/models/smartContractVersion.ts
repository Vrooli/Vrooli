import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrependString, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionSearchInput, SmartContractVersionSortBy, SmartContractVersionUpdateInput, VersionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";

const __typename = 'SmartContractVersion' as const;
type Permissions = Pick<VersionYou, 'canCopy' | 'canDelete' | 'canEdit' | 'canReport' | 'canUse' | 'canView'>;
const suppFields = ['you.canCopy', 'you.canDelete', 'you.canEdit', 'you.canReport', 'you.canUse', 'you.canView'] as const;

export const SmartContractVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: SmartContractVersionCreateInput,
    GqlUpdate: SmartContractVersionUpdateInput,
    GqlModel: SmartContractVersion,
    GqlSearch: SmartContractVersionSearchInput,
    GqlSort: SmartContractVersionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.smart_contract_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.smart_contract_versionUpsertArgs['update'],
    PrismaModel: Prisma.smart_contract_versionGetPayload<SelectWrap<Prisma.smart_contract_versionSelect>>,
    PrismaSelect: Prisma.smart_contract_versionSelect,
    PrismaWhere: Prisma.smart_contract_versionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.smart_contract_version,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages)
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'SmartContractVersion',
            pullRequest: 'PullRequest',
            reports: 'Report',
            root: 'SmartContract',
        },
        prismaRelMap: {
            __typename,
            comments: 'Comment',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'SmartContractVersion',
            pullRequest: 'PullRequest',
            reports: 'Report',
            root: 'SmartContract',
        },
        countFields: {
            commentsCount: true,
            directoryListingsCount: true,
            forksCount: true,
            reportsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
                return Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})