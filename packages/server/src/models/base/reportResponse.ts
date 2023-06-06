// TODO make sure that the report creator and object owner(s) cannot repond to reports 
// they created or own the object of
import { reportResponseValidation } from "@local/shared";
import i18next from "i18next";
import { noNull, selPad, shapeHelper } from "../../builders";
import { getSingleTypePermissions } from "../../validators";
import { ReportResponseFormat } from "../format/reportResponse";
import { ModelLogic } from "../types";
import { ReportModel } from "./report";
import { ReportResponseModelLogic } from "./types";

const __typename = "ReportResponse" as const;
const suppFields = [] as const;
export const ReportResponseModel: ModelLogic<ReportResponseModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.report_response,
    display: {
        label: {
            select: () => ({
                id: true,
                report: selPad(ReportModel.display.label.select),
            }),
            get: (select, languages) => i18next.t("common:ReportResponseLabel", { report: ReportModel.display.label.get(select.report as any, languages) }),
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
    } as any,
    validate: {} as any,
});
