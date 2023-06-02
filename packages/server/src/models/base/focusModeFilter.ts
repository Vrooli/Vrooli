import { focusModeFilterValidation, MaxObjects } from "@local/shared";
import { shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { FocusModeFilterFormat } from "../format/focusModeFilter";
import { ModelLogic } from "../types";
import { FocusModeModel } from "./focusMode";
import { TagModel } from "./tag";
import { FocusModeFilterModelLogic } from "./types";

const __typename = "FocusModeFilter" as const;
const suppFields = [] as const;
export const FocusModeFilterModel: ModelLogic<FocusModeFilterModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.focus_mode_filter,
    display: {
        label: {
            select: () => ({ id: true, tag: { select: TagModel.display.label.select() } }),
            get: (select, languages) => select.tag ? TagModel.display.label.get(select.tag as any, languages) : "",
        },
    },
    format: FocusModeFilterFormat,
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
