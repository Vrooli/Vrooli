// TODO make sure that the report creator and object owner(s) cannot repond to reports 
// they created or own the object of
import { ReportResponseSortBy, reportResponseValidation } from "@local/shared";
import i18next from "i18next";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { ReportResponseFormat } from "../formats";
import { ModelLogic } from "../types";
import { ReportModel } from "./report";
import { ReportModelLogic, ReportResponseModelLogic } from "./types";

const __typename = "ReportResponse" as const;
const suppFields = ["you"] as const;
export const ReportResponseModel: ModelLogic<ReportResponseModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.report_response,
    display: {
        label: {
            select: () => ({
                id: true,
                report: { select: ReportModel.display.label.select() },
            }),
            get: (select, languages) => i18next.t("common:ReportResponseLabel", { report: ReportModel.display.label.get(select.report as ReportModelLogic["PrismaModel"], languages) }),
        },
    },
    format: ReportResponseFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                actionSuggested: data.actionSuggested,
                details: noNull(data.details),
                language: noNull(data.language),
                createdBy: { connect: { id: rest.userData.id } },
                ...(await shapeHelper({ relation: "report", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Report", parentRelationshipName: "responses", data, ...rest })),
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
                { report: ReportModel.search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
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
    validate: {
        isTransferable: false,
        maxObjects: 100000,
        permissionsSelect: () => ({ id: true, report: "Report" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ReportModel.validate.owner(data?.report as ReportModelLogic["PrismaModel"], userId),
        isDeleted: (data, languages) => ReportModel.validate.isDeleted(data.report as ReportModelLogic["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<ReportResponseModelLogic["PrismaSelect"]>([["report", "Report"]], ...rest),
        visibility: {
            private: { report: ReportModel.validate.visibility.private },
            public: { report: ReportModel.validate.visibility.public },
            owner: (userId) => ({ report: ReportModel.validate.visibility.owner(userId) }),
        },
    },
});
