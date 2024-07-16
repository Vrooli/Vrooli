import { focusModeFilterValidation, MaxObjects } from "@local/shared";
import { ModelMap } from ".";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions } from "../../utils";
import { FocusModeFilterFormat } from "../formats";
import { FocusModeFilterModelLogic, FocusModeModelInfo, FocusModeModelLogic, TagModelInfo, TagModelLogic } from "./types";

const __typename = "FocusModeFilter" as const;
export const FocusModeFilterModel: FocusModeFilterModelLogic = ({
    __typename,
    dbTable: "focus_mode_filter",
    display: () => ({
        label: {
            select: () => ({ id: true, tag: { select: ModelMap.get<TagModelLogic>("Tag").display().label.select() } }),
            get: (select, languages) => select.tag ? ModelMap.get<TagModelLogic>("Tag").display().label.get(select.tag as TagModelInfo["PrismaModel"], languages) : "",
        },
    }),
    format: FocusModeFilterFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                filterType: data.filterType,
                // ...(await shapeHelper({ relation: "focusMode", relTypes: ['Connect'], isOneToOne: true,   objectType: 'FocusMode', parentRelationshipName: 'filters', data, ...rest })),
                // Can't use tagShapeHelper because in this case there isn't a join table between them
                tag: await shapeHelper({ relation: "tag", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Tag", parentRelationshipName: "scheduleFilters", data, ...rest }),
            }) as any,
        },
        yup: focusModeFilterValidation,
    },
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<FocusModeModelLogic>("FocusMode").validate().owner(data?.focusMode as FocusModeModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            focusMode: "FocusMode",
        }),
        visibility: {
            private: function getVisibilityPrivate() {
                return {};
            },
            public: function getVisibilityPublic() {
                return {};
            },
            owner: (userId) => ({
                focusMode: ModelMap.get<FocusModeModelLogic>("FocusMode").validate().visibility.owner(userId),
            }),
        },
    }),
});
