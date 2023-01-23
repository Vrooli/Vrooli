import { ReportResponse, ReportResponseYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const reportResponseYouPartial: GqlPartial<ReportResponseYou> = {
    __typename: 'ReportResponseYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}


export const reportResponsePartial: GqlPartial<ReportResponse> = {
    __typename: 'ReportResponse',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        actionSuggested: true,
        details: true,
        language: true,
        report: () => relPartial(require('./report').reportPartial, 'nav', { omit: 'responses' }),
        you: () => relPartial(reportResponseYouPartial, 'full'),
    },
}