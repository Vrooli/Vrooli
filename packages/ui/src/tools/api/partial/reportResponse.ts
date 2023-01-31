import { ReportResponse, ReportResponseYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const reportResponseYou: GqlPartial<ReportResponseYou> = {
    __typename: 'ReportResponseYou',
    common: {
        canDelete: true,
        canEdit: true,
    },
    full: {},
    list: {},
}


export const reportResponse: GqlPartial<ReportResponse> = {
    __typename: 'ReportResponse',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        actionSuggested: true,
        details: true,
        language: true,
        report: async () => rel((await import('./report')).report, 'nav', { omit: 'responses' }),
        you: () => rel(reportResponseYou, 'full'),
    },
    full: {},
    list: {},
}