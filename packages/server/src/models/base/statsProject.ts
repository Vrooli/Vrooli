import { MaxObjects, StatsProjectSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsProjectFormat } from "../formats";
import { ProjectModelInfo, ProjectModelLogic, StatsProjectModelInfo, StatsProjectModelLogic } from "./types";

const __typename = "StatsProject" as const;
export const StatsProjectModel: StatsProjectModelLogic = ({
    __typename,
    dbTable: "stats_project",
    display: () => ({
        label: {
            select: () => ({ id: true, project: { select: ModelMap.get<ProjectModelLogic>("Project").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<ProjectModelLogic>("Project").display().label.get(select.project as ProjectModelInfo["PrismaModel"], languages),
            }),
        },
    }),
    format: StatsProjectFormat,
    search: {
        defaultSort: StatsProjectSortBy.PeriodStartAsc,
        sortBy: StatsProjectSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ project: ModelMap.get<ProjectModelLogic>("Project").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            project: "Project",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ProjectModelLogic>("Project").validate().owner(data?.project as ProjectModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsProjectModelInfo["PrismaSelect"]>([["project", "Project"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    project: useVisibility("Project", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    project: useVisibility("Project", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    project: useVisibility("Project", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    project: useVisibility("Project", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    project: useVisibility("Project", "Public", data),
                };
            },
        },
    }),
});
