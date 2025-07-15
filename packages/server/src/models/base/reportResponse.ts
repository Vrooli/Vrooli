// TODO make sure that the report creator and object owner(s) cannot repond to reports 
// they created or own the object of
import { MaxObjects, ReportResponseSortBy, reportResponseValidation } from "@vrooli/shared";
import i18next from "i18next";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { defaultPermissions, getSingleTypePermissions } from "../../validators/permissions.js";
import { ReportResponseFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { type ReportModelInfo, type ReportModelLogic, type ReportResponseModelInfo, type ReportResponseModelLogic } from "./types.js";

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
            get: (select, languages) => i18next.t("common:ReportResponseLabel", { report: ModelMap.get<ReportModelLogic>("Report").display().label.get(select.report as ReportModelInfo["DbModel"], languages) }),
        },
    }),
    format: ReportResponseFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: BigInt(data.id),
                actionSuggested: data.actionSuggested,
                details: noNull(data.details),
                language: noNull(data.language),
                createdBy: { connect: { id: BigInt(rest.userData.id) } },
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
            suppFields: SuppFields[__typename],
            dbFields: ["createdById"],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ReportResponseModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
        owner: (data, userId) => ModelMap.get<ReportModelLogic>("Report").validate().owner(data?.report as ReportModelInfo["DbModel"], userId),
        isDeleted: (data) => ModelMap.get<ReportModelLogic>("Report").validate().isDeleted(data.report as ReportModelInfo["DbModel"]),
        isPublic: (data, getParentInfo?) => oneIsPublic<ReportResponseModelInfo["DbSelect"]>([["report", "Report"]], data, getParentInfo),
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
