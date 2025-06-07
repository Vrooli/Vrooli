import { MaxObjects, RunIOSortBy, runIOValidation } from "@vrooli/shared";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { RunIOFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type RunIOModelInfo, type RunIOModelLogic, type RunModelInfo, type RunModelLogic } from "./types.js";

const __typename = "RunIO" as const;
export const RunIOModel: RunIOModelLogic = ({
    __typename,
    dbTable: "run_routine_io",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                run: { select: ModelMap.get<RunModelLogic>("Run").display().label.select() },
            }),
            get: (select, languages) => {
                return ModelMap.get<RunModelLogic>("Run").display().label.get(select.run as RunModelInfo["DbModel"], languages);
            },
        },
    }),
    format: RunIOFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: BigInt(data.id),
                    data: data.data,
                    nodeInputName: data.nodeInputName,
                    nodeName: data.nodeName,
                    run: await shapeHelper({ relation: "run", relTypes: ["Connect"], isOneToOne: true, objectType: "Run", parentRelationshipName: "io", data, ...rest }),
                };
            },
            update: async ({ data }) => {
                return {
                    data: data.data,
                };
            },
        },
        yup: runIOValidation,
    },
    search: {
        defaultSort: RunIOSortBy.DateUpdatedDesc,
        sortBy: RunIOSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            runIds: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ run: ModelMap.get<RunModelLogic>("Run").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            run: "Run",
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ["data"],
        owner: (data, userId) => ModelMap.get<RunModelLogic>("Run").validate().owner(data?.run as RunModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunIOModelInfo["DbSelect"]>([["run", "Run"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    run: useVisibility("Run", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    run: useVisibility("Run", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    run: useVisibility("Run", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    run: useVisibility("Run", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    run: useVisibility("Run", "Public", data),
                };
            },
        },
    }),
});
