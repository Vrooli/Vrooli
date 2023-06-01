import { Formatter } from "../types";

const __typename = "Report" as const;
export const ReportFormat: Formatter<ModelReportLogic> = {
    gqlRelMap: {
        __typename,
        responses: "ReportResponse",
    },
    prismaRelMap: {
        __typename,
        responses: "ReportResponse",
    },
    hiddenFields: ["createdById"], // Always hide report creator
}${ forMapper[x.createdFor]}Id`]: { id: x.createdForConnect },
                            })),
                        },
                    });
                    if (existing.length > 0)
                        throw new CustomError("0337", "MaxReportsReached", userData.languages);
                }
            },
            create: async ({ data, userData }) => {
                return {
                    id: data.id,
                    language: data.language,
                    reason: data.reason,
                    details: data.details,
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: userData.id } },
                    [forMapper[data.createdFor]]: { connect: { id: data.createdForConnect } },
                };
                return {
                    reason: data.reason ?? undefined,
                    details: data.details,
};
