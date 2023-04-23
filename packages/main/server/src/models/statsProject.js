import { StatsProjectSortBy } from "@local/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ProjectModel } from "./project";
const __typename = "StatsProject";
const suppFields = [];
export const StatsProjectModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_project,
    display: {
        select: () => ({ id: true, project: selPad(ProjectModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: ProjectModel.display.label(select.project, languages),
        }),
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            project: "Api",
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsProjectSortBy.PeriodStartAsc,
        sortBy: StatsProjectSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ project: ProjectModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            project: "Project",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ProjectModel.validate.owner(data.project, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["project", "Project"],
        ], languages),
        visibility: {
            private: { project: ProjectModel.validate.visibility.private },
            public: { project: ProjectModel.validate.visibility.public },
            owner: (userId) => ({ project: ProjectModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=statsProject.js.map