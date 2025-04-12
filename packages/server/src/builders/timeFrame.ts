import { SECONDS_1_MS, TimeFrame } from "@local/shared";

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
export function timeFrameToPrisma(fieldName: string, time?: TimeFrame | null | undefined): TimeFrameWhereQuery | undefined {
    function safeTimeFrame<T extends TimeFrame>(time: T | null | undefined): T | undefined {
        if (!time || typeof time !== "object") return undefined;
        const where: TimeFrameWhereQuery = ({ [fieldName]: {} });

        if (time.before) {
            try {
                // Validate by attempting to create a Date object
                const date = new Date(time.before);
                if (!isNaN(date.getTime())) {
                    where[fieldName].lte = time.before;
                }
            } catch (e) {
                // Invalid date format, ignore
            }
        }

        if (time.after) {
            try {
                // Validate by attempting to create a Date object
                const date = new Date(time.after);
                if (!isNaN(date.getTime())) {
                    where[fieldName].gte = time.after;
                }
            } catch (e) {
                // Invalid date format, ignore
            }
        }

        if (Object.keys(where[fieldName]).length > 0) return where as T;
        return undefined;
    }
    const where = safeTimeFrame(time);
    if (!where) return undefined;
    return where;
}

/**
 * Builds a partial SQL query for date range limitations on a specified field.
 * This can be used in SQL queries to filter results based on time frames.
 * 
 * @param timeFrame - Timeframe with after and before dates to limit the query.
 * @param fieldName - The database field to which the timeframe should apply.
 * @returns A SQL string that can be inserted into a query to apply date filters.
 */
export function timeFrameToSql(fieldName: string, timeFrame: TimeFrame | undefined): string | null {
    const conditions: string[] = [];
    if (!timeFrame) return null;

    if (timeFrame.after) {
        const afterSeconds = Math.floor(new Date(timeFrame.after).getTime() / SECONDS_1_MS);
        conditions.push(`EXTRACT(EPOCH FROM t."${fieldName}") >= ${afterSeconds}`);
    }

    if (timeFrame.before) {
        const beforeSeconds = Math.floor(new Date(timeFrame.before).getTime() / SECONDS_1_MS);
        conditions.push(`EXTRACT(EPOCH FROM t."${fieldName}") <= ${beforeSeconds}`);
    }

    return conditions.length > 0 ? conditions.join(" AND ") : null;
}
