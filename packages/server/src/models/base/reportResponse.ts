// TODO make sure that the report creator and object owner(s) cannot repond to reports 
// they created or own the object of
import { ReportResponseSortBy, reportResponseValidation } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { ReportResponseFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ReportModelInfo, ReportModelLogic, ReportResponseModelInfo, ReportResponseModelLogic } from "./types";

const __typename = "ReportResponse" as const;
export const ReportResponseModel: ReportResponseModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.report_response,
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
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: 100000,
        permissionsSelect: () => ({ id: true, report: "Report" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ReportModelLogic>("Report").validate().owner(data?.report as ReportModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<ReportModelLogic>("Report").validate().isDeleted(data.report as ReportModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<ReportResponseModelInfo["PrismaSelect"]>([["report", "Report"]], ...rest),
        visibility: {
            private: { report: ModelMap.get<ReportModelLogic>("Report").validate().visibility.private },
            public: { report: ModelMap.get<ReportModelLogic>("Report").validate().visibility.public },
            owner: (userId) => ({ report: ModelMap.get<ReportModelLogic>("Report").validate().visibility.owner(userId) }),
        },
    }),
});
