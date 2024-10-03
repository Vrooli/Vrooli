import { MaxObjects, StatsTeamSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsTeamFormat } from "../formats";
import { StatsTeamModelInfo, StatsTeamModelLogic, TeamModelInfo, TeamModelLogic } from "./types";

const __typename = "StatsTeam" as const;
export const StatsTeamModel: StatsTeamModelLogic = ({
    __typename,
    dbTable: "stats_team",
    display: () => ({
        label: {
            select: () => ({ id: true, team: { select: ModelMap.get<TeamModelLogic>("Team").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<TeamModelLogic>("Team").display().label.get(select.team as TeamModelInfo["PrismaModel"], languages),
            }),
        },
    }),
    format: StatsTeamFormat,
    search: {
        defaultSort: StatsTeamSortBy.PeriodStartAsc,
        sortBy: StatsTeamSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ team: ModelMap.get<TeamModelLogic>("Team").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            team: "Team",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<TeamModelLogic>("Team").validate().owner(data?.team as TeamModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsTeamModelInfo["PrismaSelect"]>([["team", "Team"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    team: useVisibility("Team", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    team: useVisibility("Team", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    team: useVisibility("Team", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    team: useVisibility("Team", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    team: useVisibility("Team", "Public", data),
                };
            },
        },
    }),
});
