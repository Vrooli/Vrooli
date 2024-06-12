import { StatsProjectSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
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
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            project: "Project",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ProjectModelLogic>("Project").validate().owner(data?.project as ProjectModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsProjectModelInfo["PrismaSelect"]>([["project", "Project"]], ...rest),
        visibility: {
            private: { project: ModelMap.get<ProjectModelLogic>("Project").validate().visibility.private },
            public: { project: ModelMap.get<ProjectModelLogic>("Project").validate().visibility.public },
            owner: (userId) => ({ project: ModelMap.get<ProjectModelLogic>("Project").validate().visibility.owner(userId) }),
        },
    }),
});
