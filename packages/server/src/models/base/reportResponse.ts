// TODO make sure that the report creator and object owner(s) cannot repond to reports 
// they created or own the object of
import { MaxObjects, ReportResponseSortBy, reportResponseValidation } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { ReportResponseFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ReportModelInfo, ReportModelLogic, ReportResponseModelInfo, ReportResponseModelLogic } from "./types";

const __typename = "ReportResponse" as const;
export const ReportResponseModel: ReportResponseModelLogic = ({
    __typename,
    dbTable: "report_response",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                report: { select: ModelMap.get<ReportModelLogic>("Report").display().label.select() },
            }),
            get: (select, languages) => i18next.t("common:ReportResponseLabel", { report: ModelMap.get<ReportModelLogic>("Report").display().label.get(select.report as ReportModelInfo["PrismaModel"], languages) }),
        },
    }),
    format: ReportResponseFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                actionSuggested: data.actionSuggested,
                details: noNull(data.details),
                language: noNull(data.language),
                createdBy: { connect: { id: rest.userData.id } },
                report: await shapeHelper({ relation: "report", relTypes: ["Connect"], isOneToOne: true, objectType: "Report", parentRelationshipName: "responses", data, ...rest }),
            }),
            update: async ({ data }) => ({
                actionSuggested: noNull(data.actionSuggested),
                details: noNull(data.details),
                language: noNull(data.language),
            }),
        },
        yup: reportResponseValidation,
    },
    search: {
        defaultSort: ReportResponseSortBy.DateCreatedDesc,
        sortBy: ReportResponseSortBy,
        searchFields: {
            createdTimeFrame: true,
            languageIn: true,
            reportId: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "detailsWrapped",
                { report: ModelMap.get<ReportModelLogic>("Report").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            dbFields: ["createdById"],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ReportResponseModelInfo["GqlPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, report: "Report" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ReportModelLogic>("Report").validate().owner(data?.report as ReportModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<ReportModelLogic>("Report").validate().isDeleted(data.report as ReportModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<ReportResponseModelInfo["PrismaSelect"]>([["report", "Report"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    report: useVisibility("Report", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    report: useVisibility("Report", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    report: useVisibility("Report", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    report: useVisibility("Report", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    report: useVisibility("Report", "Public", data),
                };
            },
        },
    }),
});
