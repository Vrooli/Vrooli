// TODO make sure that the report creator and object owner(s) cannot repond to reports 
// they created or own the object of
import { Prisma } from "@prisma/client";
import { ReportResponse, ReportResponseCreateInput, ReportResponseSearchInput, ReportResponseSortBy, ReportResponseUpdateInput, ReportResponseYou} from "@shared/consts";
import { reportResponseValidation } from "@shared/validation";
import i18next from "i18next";
import { noNull, selPad, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { ReportModel } from "./report";
import { ModelLogic } from "./types";

const __typename = 'ReportResponse' as const;
type Permissions = Pick<ReportResponseYou, 'canDelete' | 'canUpdate'>;
const suppFields = [] as const;
export const ReportResponseModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReportResponseCreateInput,
    GqlUpdate: ReportResponseUpdateInput,
    GqlModel: ReportResponse,
    GqlSearch: ReportResponseSearchInput,
    GqlSort: ReportResponseSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.report_responseUpsertArgs['create'],
    PrismaUpdate: Prisma.report_responseUpsertArgs['update'],
    PrismaModel: Prisma.report_responseGetPayload<SelectWrap<Prisma.report_responseSelect>>,
    PrismaSelect: Prisma.report_responseSelect,
    PrismaWhere: Prisma.report_responseWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.report_response,
    display: {
        select: () => ({
            id: true,
            report: selPad(ReportModel.display.select),
        }),
        label: (select, languages) => i18next.t('common:ReportResponseLabel', { report: ReportModel.display.label(select.report as any, languages) }),
    },
    format: {
        gqlRelMap: { 
            __typename,
            report: 'Report',
        },
        prismaRelMap: { 
            __typename,
            report: 'Report',
        },
        hiddenFields: ['createdById'], // Always hide report creator
        supplemental: {
            graphqlFields: suppFields,
            dbFields: ['createdById'],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    }
                }
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
                ...(await shapeHelper({ relation: 'report', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Report', parentRelationshipName: 'responses', data, ...rest })),
            }),
            update: async ({ data }) => ({
                actionSuggested: noNull(data.actionSuggested),
                details: noNull(data.details),
                language: noNull(data.language),
            }),
        },
        yup: reportResponseValidation,
    },
    search: {} as any,
    validate: {} as any,
})