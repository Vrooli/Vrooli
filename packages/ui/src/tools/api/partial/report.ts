import { Report, ReportYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const reportYou: GqlPartial<ReportYou> = {
    __typename: 'ReportYou',
    common: {
        canDelete: true,
        canEdit: true,
        canRespond: true,
    },
    full: {},
    list: {},
}

export const report: GqlPartial<Report> = {
    __typename: 'Report',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        details: true,
        language: true,
        reason: true,
        responsesCount: true,
        you: () => rel(reportYou, 'full'),
    },
    full: {
        responses: async () => rel((await import('./reportResponse')).reportResponse, 'full', { omit: 'report' }),
    },
    list: {},
}