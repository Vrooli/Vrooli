import { MaxObjects, RoleSortBy, getTranslation, roleValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { RoleFormat } from "../formats";
import { RoleModelInfo, RoleModelLogic, TeamModelInfo, TeamModelLogic } from "./types";

const __typename = "Role" as const;
export const RoleModel: RoleModelLogic = ({
    __typename,
    dbTable: "role",
    dbTranslationTable: "role_translation",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                name: true,
                translations: { select: { language: true, name: true } },
            }),
            get: (select, languages) => {
                // Prefer translated name over default name
                const translated = getTranslation(select, languages).name ?? "";
                if (translated.length > 0) return translated;
                return select.name;
            },
        },
    }),
    format: RoleFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                name: data.name,
                permissions: data.permissions,
                members: await shapeHelper({ relation: "members", relTypes: ["Connect"], isOneToOne: false, objectType: "Member", parentRelationshipName: "roles", data, ...rest }),
                team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "roles", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                name: noNull(data.name),
                permissions: noNull(data.permissions),
                members: await shapeHelper({ relation: "members", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Member", parentRelationshipName: "roles", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        yup: roleValidation,
    },
    search: {
        defaultSort: RoleSortBy.MembersDesc,
        sortBy: RoleSortBy,
        searchFields: {
            createdTimeFrame: true,
            translationLanguages: true,
            teamId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                "transDescriptionWrapped",
                "transNameWrapped",
            ],
        }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, team: "Team" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<TeamModelLogic>("Team").validate().owner(data?.team as TeamModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<TeamModelLogic>("Team").validate().isDeleted(data.team as TeamModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<RoleModelInfo["PrismaSelect"]>([["team", "Team"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    team: useVisibility("Team", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    team: useVisibility("RoutineVersion", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    team: useVisibility("RoutineVersion", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    team: useVisibility("RoutineVersion", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    team: useVisibility("RoutineVersion", "Public", data),
                };
            },
        },
    }),
});
