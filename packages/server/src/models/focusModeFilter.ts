import { FocusModeFilter, FocusModeFilterCreateInput, focusModeFilterValidation, FocusModeSearchInput, FocusModeSortBy, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { FocusModeModel } from "./focusMode";
import { TagModel } from "./tag";
import { ModelLogic } from "./types";

const __typename = "FocusModeFilter" as const;
const suppFields = [] as const;
export const FocusModeFilterModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: FocusModeFilterCreateInput,
    GqlUpdate: undefined,
    GqlModel: FocusModeFilter,
    GqlPermission: {},
    GqlSearch: FocusModeSearchInput,
    GqlSort: FocusModeSortBy,
    PrismaCreate: Prisma.focus_mode_filterUpsertArgs["create"],
    PrismaUpdate: Prisma.focus_mode_filterUpsertArgs["update"],
    PrismaModel: Prisma.focus_mode_filterGetPayload<SelectWrap<Prisma.focus_mode_filterSelect>>,
    PrismaSelect: Prisma.focus_mode_filterSelect,
    PrismaWhere: Prisma.focus_mode_filterWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.focus_mode_filter,
    display: {
        select: () => ({ id: true, tag: { select: TagModel.display.select() } }),
        label: (select, languages) => select.tag ? TagModel.display.label(select.tag as any, languages) : "",
    },
    format: {
        gqlRelMap: {
            __typename,
            focusMode: "FocusMode",
            tag: "Tag",
        },
        prismaRelMap: {
            __typename,
            focusMode: "FocusMode",
            tag: "Tag",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                filterType: data.filterType,
                // ...(await shapeHelper({ relation: "focusMode", relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'FocusMode', parentRelationshipName: 'filters', data, ...rest })),
                // Can't use tagShapeHelper because in this case there isn't a join table between them
                ...(await shapeHelper({ relation: "tag", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Tag", parentRelationshipName: "scheduleFilters", data, ...rest })),
            }) as any,
        },
        yup: focusModeFilterValidation,
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => FocusModeModel.validate!.owner(data.focusMode as any, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            focusMode: "FocusMode",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                focusMode: FocusModeModel.validate!.visibility.owner(userId),
            }),
        },
    },
});
