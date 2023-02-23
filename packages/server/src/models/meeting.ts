import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, Meeting, MeetingCreateInput, MeetingSearchInput, MeetingSortBy, MeetingUpdateInput, MeetingYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions } from "../utils";
import { ModelLogic } from "./types";
import { OrganizationModel } from "./organization";
import { getSingleTypePermissions } from "../validators";

const __typename = 'Meeting' as const;
type Permissions = Pick<MeetingYou, 'canDelete' | 'canInvite' | 'canUpdate'>;
const suppFields = ['you'] as const;
export const MeetingModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: MeetingCreateInput,
    GqlUpdate: MeetingUpdateInput,
    GqlModel: Meeting,
    GqlSearch: MeetingSearchInput,
    GqlSort: MeetingSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.meetingUpsertArgs['create'],
    PrismaUpdate: Prisma.meetingUpsertArgs['update'],
    PrismaModel: Prisma.meetingGetPayload<SelectWrap<Prisma.meetingSelect>>,
    PrismaSelect: Prisma.meetingSelect,
    PrismaWhere: Prisma.meetingWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            attendees: 'User',
            invites: 'MeetingInvite',
            labels: 'Label',
            organization: 'Organization',
            restrictedToRoles: 'Role',
        },
        prismaRelMap: {
            __typename,
            organization: 'Organization',
            restrictedToRoles: 'Role',
            attendees: 'User',
            invites: 'MeetingInvite',
            labels: 'Label',
        },
        joinMap: {
            labels: 'label',
            restrictedToRoles: 'role',
            attendees: 'user',
            invites: 'user',
        },
        countFields: {
            attendeesCount: true,
            invitesCount: true,
            labelsCount: true,
            translationsCount: true,
        },
        supplemental: {
            dbFields: ['organizationId'],
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                }
            },
        },
    },
    mutate: {} as any,
    search: {
        defaultSort: MeetingSortBy.EventStartAsc,
        sortBy: MeetingSortBy,
        searchFields: {
            createdTimeFrame: true,
            labelsIds: true,
            maxEventEnd: true,
            maxEventStart: true,
            maxRecurrEnd: true,
            maxRecurrStart: true,
            minEventEnd: true,
            minEventStart: true,
            minRecurrEnd: true,
            minRecurrStart: true,
            openToAnyoneWithInvite: true,
            organizationId: true,
            showOnOrganizationProfile: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'labelsWrapped',
                'transNameWrapped',
                'transDescriptionWrapped'
            ]
        })
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            showOnOrganizationProfile: true,
            organization: 'Organization',
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isPublic }),
            canInvite: () => isAdmin,
        }),
        owner: (data) => ({
            Organization: data.organization,
        }),
        isDeleted: () => false,
        isPublic: (data) => data.showOnOrganizationProfile === true,
        visibility: {
            private: {
                OR: [
                    { showOnOrganizationProfile: false },
                    { organization: { isPrivate: true } },
                ]
            },
            public: {
                AND: [
                    { showOnOrganizationProfile: true },
                    { organization: { isPrivate: false } },
                ]
            },
            owner: (userId) => ({
                organization: OrganizationModel.query.hasRoleQuery(userId),
            }),
        }
    },
})