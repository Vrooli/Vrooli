import { TimeFrame } from "@local/shared";

type TimeFrameWhereQuery = {
    [fieldName: string]: {
        lte?: Date | string;
        gte?: Date | string;
    };
};

/**
 * Converts time frame to Prisma "where" query
 * @param time Time frame to convert
 * @param fieldName Name of time field (typically created_at or updated_at)
 * @returns Prisma "where" query if any time frame fields are defined, undefined otherwise
 */
export const timeFrameToPrisma = (fieldName: string, time?: TimeFrame | null | undefined): TimeFrameWhereQuery | undefined => {
    if (!time || (!time.before && !time.after)) return undefined;
    const where: TimeFrameWhereQuery = ({ [fieldName]: {} });
    if (time.before) where[fieldName].lte = time.before;
    if (time.after) where[fieldName].gte = time.after;
    return where;
};
