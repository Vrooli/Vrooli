import { Report, ReportYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { reportResponsePartial } from "./reportResponse";

export const reportYouPartial: GqlPartial<ReportYou> = {
    __typename: 'ReportYou',
    full: {
        canDelete: true,
        canEdit: true,
        canRespond: true,
    },
}

export const reportPartial: GqlPartial<Report> = {
    __typename: 'Report',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        details: true,
        language: true,
        reason: true,
        responsesCount: true,
        you: () => relPartial(reportYouPartial, 'full'),
    },
    full: {
        responses: () => relPartial(reportResponsePartial, 'full', { omit: 'report' }),
    }
}