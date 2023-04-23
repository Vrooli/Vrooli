import { reportResponseValidation } from "@local/validation";
import i18next from "i18next";
import { noNull, selPad, shapeHelper } from "../builders";
import { getSingleTypePermissions } from "../validators";
import { ReportModel } from "./report";
const __typename = "ReportResponse";
const suppFields = [];
export const ReportResponseModel = ({
    __typename,
    delegate: (prisma) => prisma.report_response,
    display: {
        select: () => ({
            id: true,
            report: selPad(ReportModel.display.select),
        }),
        label: (select, languages) => i18next.t("common:ReportResponseLabel", { report: ReportModel.display.label(select.report, languages) }),
    },
    format: {
        gqlRelMap: {
            __typename,
            report: "Report",
        },
        prismaRelMap: {
            __typename,
            report: "Report",
        },
        hiddenFields: ["createdById"],
        supplemental: {
            graphqlFields: suppFields,
            dbFields: ["createdById"],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
        countFields: {
            responsesCount: true,
        },
    },
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
    search: {},
    validate: {},
});
//# sourceMappingURL=reportResponse.js.map