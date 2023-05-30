import { PrismaType } from "../../../types";

export enum SearchSortOption {
    New = "New",
    Top = "Top",
}

export enum SearchTimePeriod {
    Day = "Day",
    Week = "Week",
    Month = "Month",
    Year = "Year",
    AllTime = "AllTime"
}

export interface QueryEmbeddingsProps {
    limit?: number;
    offset?: number;
    prisma: PrismaType;
    searchString: string;
    sortOption: SearchSortOption;
    thresholdBookmarks?: number;
    thresholdDistance?: number;
    thresholdScore?: number;
    thresholdViews?: number;
    timePeriod?: SearchTimePeriod;
}

export interface QueryEmbeddingsHelperProps extends Omit<QueryEmbeddingsProps, "searchString" | "sortOption" | "timePeriod"> {
    dateLimit: number;
    embedding: string;
}

/**
 * Converts a SearchTimePeriod to the number of seconds since the epoch.
 */
export function getDateLimit(period?: SearchTimePeriod): number {
    let dateLimit: Date = new Date();
    switch (period) {
        case SearchTimePeriod.Day:
            dateLimit.setDate(dateLimit.getDate() - 1);
            break;
        case SearchTimePeriod.Week:
            dateLimit.setDate(dateLimit.getDate() - 7);
            break;
        case SearchTimePeriod.Month:
            dateLimit.setMonth(dateLimit.getMonth() - 1);
            break;
        case SearchTimePeriod.Year:
            dateLimit.setFullYear(dateLimit.getFullYear() - 1);
            break;
        case SearchTimePeriod.AllTime:
        default:
            dateLimit = new Date(0);
            break;
    }
    return Math.floor(dateLimit.getTime() / 1000); // Convert to seconds
}
