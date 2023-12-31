import { ProjectVersionDirectory, Session } from "@local/shared";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { DirectoryItem, DirectoryListSortBy } from "./types";

export const initializeDirectoryList = (
    directory: ProjectVersionDirectory | null | undefined,
    sortBy: DirectoryListSortBy,
    session: Session | null | undefined,
): DirectoryItem[] => {
    if (!directory) return [];
    const items = [
        ...directory.childApiVersions,
        ...directory.childNoteVersions,
        ...directory.childOrganizations,
        ...directory.childProjectVersions,
        ...directory.childRoutineVersions,
        ...directory.childSmartContractVersions,
        ...directory.childStandardVersions,
    ];
    const userLanguages = getUserLanguages(session);

    return items.sort((a, b) => {
        // Extracting titles for name-based sorting
        const { title: aTitle } = getDisplay(a, userLanguages);
        const { title: bTitle } = getDisplay(b, userLanguages);

        // Helper function to safely parse dates and return a timestamp
        const parseDate = (dateString: string) => {
            const date = new Date(dateString);
            return isNaN(date.getTime()) ? 0 : date.getTime(); // Return 0 if date is invalid
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
