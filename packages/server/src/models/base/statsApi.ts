import { StatsApiSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsApiFormat } from "../formats";
import { ModelLogic } from "../types";
import { ApiModel } from "./api";
import { ApiModelLogic, StatsApiModelLogic } from "./types";

const __typename = "StatsApi" as const;
const suppFields = [] as const;
export const StatsApiModel: ModelLogic<StatsApiModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_api,
    display: {
        label: {
            select: () => ({ id: true, api: { select: ApiModel.display.label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ApiModel.display.label.get(select.api as ApiModelLogic["PrismaModel"], languages),
            }),
        },
    },
    format: StatsApiFormat,
    search: {
        defaultSort: StatsApiSortBy.PeriodStartAsc,
        sortBy: StatsApiSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ api: ApiModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            api: "Api",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ApiModel.validate.owner(data?.api as ApiModelLogic["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_apiSelect>(data, [
            ["api", "Api"],
        ], languages),
        visibility: {
            private: { api: ApiModel.validate.visibility.private },
            public: { api: ApiModel.validate.visibility.public },
            owner: (userId) => ({ api: ApiModel.validate.visibility.owner(userId) }),
        },
    },
});
