import { StatsProjectSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsProjectFormat } from "../format/statsProject";
import { ModelLogic } from "../types";
import { ProjectModel } from "./project";
import { StatsProjectModelLogic } from "./types";

const __typename = "StatsProject" as const;
const suppFields = [] as const;
export const StatsProjectModel: ModelLogic<StatsProjectModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_project,
    display: {
        label: {
            select: () => ({ id: true, project: selPad(ProjectModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ProjectModel.display.label.get(select.project as any, languages),
            }),
        },
    },
    format: StatsProjectFormat,
    search: {
        defaultSort: StatsProjectSortBy.PeriodStartAsc,
        sortBy: StatsProjectSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ project: ProjectModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            project: "Project",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ProjectModel.validate.owner(data.project as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_projectSelect>(data, [
            ["project", "Project"],
        ], languages),
        visibility: {
            private: { project: ProjectModel.validate.visibility.private },
            public: { project: ProjectModel.validate.visibility.public },
            owner: (userId) => ({ project: ProjectModel.validate.visibility.owner(userId) }),
        },
    },
});
