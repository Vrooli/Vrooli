import { ReportYou } from "@shared/consts";
import { GqlPartial } from "types";

export const reportYouPartial: GqlPartial<ReportYou> = {
    __typename: 'ReportYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
        canRespond: true,
    }),
}

export const listReportFields = ['Report', `{
    id
}`] as const;
export const reportFields = ['Report', `{
    id
    language
    reason
    details
    isOwn
}`] as const;