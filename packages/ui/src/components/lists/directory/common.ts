import { ProjectVersionDirectory, Session, TimeFrame } from "@local/shared";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { DirectoryItem } from "./types";

export enum DirectoryListSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    NameAsc = "NameAsc",
    NameDesc = "NameDesc",
}

export const initializeDirectoryList = (
    directory: ProjectVersionDirectory | null | undefined,
    sortBy: DirectoryListSortBy | `${DirectoryListSortBy}`,
    timeFrame: TimeFrame | undefined,
    session: Session | null | undefined,
): DirectoryItem[] => {
    if (!directory) return [];
    let items = [
        ...directory.childApiVersions,
        ...directory.childCodeVersions,
        ...directory.childNoteVersions,
        ...directory.childProjectVersions,
        ...directory.childRoutineVersions,
        ...directory.childStandardVersions,
        ...directory.childTeams,
    ];
    const userLanguages = getUserLanguages(session);

    // Helper function to safely parse dates and return a timestamp
    const parseDate = (dateString: string) => {
        const date = new Date(dateString);
        return !isFinite(date.getTime()) ? 0 : date.getTime(); // Return 0 if date is invalid
    };

    // Filter items based on time frame
    if (timeFrame?.after || timeFrame?.before) {
        const afterTime = timeFrame?.after ? parseDate(timeFrame.after) : -Infinity;
        const beforeTime = timeFrame?.before ? parseDate(timeFrame.before) : Infinity;

        items = items.filter(item => {
            const itemTime = parseDate(item.created_at); // Assuming filtering by creation date
            return itemTime > afterTime && itemTime < beforeTime;
        });
    }

    return items.sort((a, b) => {
        // Extracting titles for name-based sorting
        const { title: aTitle } = getDisplay(a, userLanguages);
        const { title: bTitle } = getDisplay(b, userLanguages);

        // Helper function to safely parse dates and return a timestamp
        const parseDate = (dateString: string) => {
            const date = new Date(dateString);
            return !isFinite(date.getTime()) ? 0 : date.getTime(); // Return 0 if date is invalid
        };

        switch (sortBy) {
            case "DateCreatedAsc":
                return parseDate(a.created_at) - parseDate(b.created_at);
            case "DateCreatedDesc":
                return parseDate(b.created_at) - parseDate(a.created_at);
            case "DateUpdatedAsc":
                return parseDate(a.updated_at) - parseDate(b.updated_at);
            case "DateUpdatedDesc":
                return parseDate(b.updated_at) - parseDate(a.updated_at);
            case "NameAsc":
                return aTitle.localeCompare(bTitle);
            case "NameDesc":
                return bTitle.localeCompare(aTitle);
            default:
                // Default fallback to name ascending if no valid sort by is provided
                return aTitle.localeCompare(bTitle);
        }
    });
};
