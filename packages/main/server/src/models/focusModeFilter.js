import { MaxObjects } from "@local/consts";
import { focusModeFilterValidation } from "@local/validation";
import { shapeHelper } from "../builders";
import { defaultPermissions } from "../utils";
import { FocusModeModel } from "./focusMode";
import { TagModel } from "./tag";
const __typename = "FocusModeFilter";
const suppFields = [];
export const FocusModeFilterModel = ({
    __typename,
    delegate: (prisma) => prisma.focus_mode_filter,
    display: {
        select: () => ({ id: true, tag: { select: TagModel.display.select() } }),
        label: (select, languages) => select.tag ? TagModel.display.label(select.tag, languages) : "",
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
                ...(await shapeHelper({ relation: "tag", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Tag", parentRelationshipName: "scheduleFilters", data, ...rest })),
            }),
        },
        yup: focusModeFilterValidation,
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => FocusModeModel.validate.owner(data.focusMode, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            focusMode: "FocusMode",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                focusMode: FocusModeModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=focusModeFilter.js.map