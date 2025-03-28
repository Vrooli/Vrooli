import { BookmarkFor, CopyType, DeleteType, ListObject, ReportFor, Session } from "@local/shared";
import { checkIfLoggedIn } from "../../utils/authentication/session.js";
import { getYou } from "../display/listTools.js";

/**
 * All available bulk actions
 */
export enum BulkObjectAction {
    Bookmark = "Bookmark",
    BookmarkUndo = "BookmarkUndo",
    Delete = "Delete",
    Export = "Export",
    ProjectAdd = "ProjectAdd",
    Report = "Report",
}

/**
 * Indicates that a BulkObjectAction has been completed. 
 * Basically any action that requires updating state or navigating to a new page.
 */
export enum BulkObjectActionComplete {
    Bookmark = "Bookmark",
    BookmarkUndo = "BookmarkUndo",
    Delete = "Delete",
    Report = "Report",
}

/**
 * If any object has a false, permission, then the permission for the whole list is false.
 */
function getBulkYou(objects: ListObject[]) {
    const permissions = objects.map(object => getYou(object));
    return {
        canCopy: permissions.every(perm => perm.canCopy),
        canDelete: permissions.every(perm => perm.canDelete),
        canReport: permissions.every(perm => perm.canReport),
        canBookmark: permissions.every(perm => perm.canBookmark),
        isBookmarked: permissions.every(perm => perm.isBookmarked),
    };
}

/**
 * Determines which actions are available for the given objects.
 * Actions follow the order: Edit, Bookmark, ProjectAdd, Export, Report, Delete
 * @param objects All selected objects
 * @param session Current session. Many actions require a logged in user.
 */
export function getAvailableBulkActions(objects: ListObject[], session: Session | undefined): BulkObjectAction[] {
    if (objects.length <= 0) return [];
    const isLoggedIn = checkIfLoggedIn(session);
    const { canCopy, canDelete, canReport, canBookmark, isBookmarked } = getBulkYou(objects);
    const options: BulkObjectAction[] = [];
    // Check Bookmark/BookmarkUndo
    if (isLoggedIn && canBookmark && objects.some(object => object.__typename in BookmarkFor)) {
        options.push(isBookmarked ? BulkObjectAction.BookmarkUndo : BulkObjectAction.Bookmark);
    }
    // If you can see an object, you can add it to a project
    options.push(BulkObjectAction.ProjectAdd);
    // Check Export
    if (isLoggedIn && canCopy && objects.some(object => object.__typename in CopyType)) {
        options.push(BulkObjectAction.Export);
    }
    // Check Report
    if (isLoggedIn && canReport && objects.some(object => object.__typename in ReportFor)) {
        options.push(BulkObjectAction.Report);
    }
    // Check Delete
    if (isLoggedIn && canDelete && objects.some(object => object.__typename in DeleteType)) {
        options.push(BulkObjectAction.Delete);
    }
    return options;
}
