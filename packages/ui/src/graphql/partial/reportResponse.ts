import { ReportResponseYou } from "@shared/consts";
import { GqlPartial } from "types";

export const reportResponseYouPartial: GqlPartial<ReportResponseYou> = {
    __typename: 'ReportResponseYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}

export const listReportResponseFields = ['ReportResponse', `{
    id
}`] as const;
export const reportResponseFields = ['ReportResponse', `{
    id
}`] as const;