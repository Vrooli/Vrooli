import { ReportResponse, ReportResponseYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { reportPartial } from "./report";

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
        report: () => relPartial(reportPartial, 'nav', { omit: 'responses' }),
        you: () => relPartial(reportResponseYouPartial, 'full'),
    },
}