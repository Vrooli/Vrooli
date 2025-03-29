import { Bookmark, BookmarkFor, CommentFor, CopyResult, CopyType, DeleteType, ListObject, ReactionFor, ReportFor, Session, Success, TranslationKeyCommon, getReactionScore } from "@local/shared";
import { ListMenuItemData } from "../../components/dialogs/types.js";
import { IconInfo } from "../../icons/Icons.js";
import { checkIfLoggedIn } from "../../utils/authentication/session.js";
import { getYou } from "../../utils/display/listTools.js";

/**
 * All available actions an object can possibly have
 */
export enum ObjectAction {
    Bookmark = "Bookmark",
    BookmarkUndo = "BookmarkUndo",
    Comment = "Comment",
    Delete = "Delete",
    Donate = "Donate",
    Edit = "Edit",
    FindInPage = "FindInPage",
    Fork = "Fork",
    Report = "Report",
    Share = "Share",
    Stats = "Stats",
    VoteDown = "VoteDown",
    VoteUp = "VoteUp",
}

/**
 * Indicates that a ObjectAction has been completed. 
 * Basically any action that requires updating state or navigating to a new page.
 */
export enum ObjectActionComplete {
    Bookmark = "Bookmark",
    BookmarkUndo = "BookmarkUndo",
    Delete = "Delete",
    EditComplete = "EditComplete",
    EditCancel = "EditCanel",
    Fork = "Fork",
    Report = "Report",
    VoteDown = "VoteDown",
    VoteUp = "VoteUp",
}

// None of the start actions have payloads at the moment, 
// but this allows us to easily add them in the future.
export interface ActionStartPayloads {
    Bookmark: unknown;
    BookmarkUndo: unknown;
    Comment: unknown;
    Delete: unknown;
    Donate: unknown;
    Edit: unknown;
    FindInPage: unknown;
    Fork: unknown;
    Report: unknown;
    Share: unknown;
    Stats: unknown;
    VoteDown: unknown;
    VoteUp: unknown;
}

export interface ActionCompletePayloads {
    Bookmark: Bookmark;
    BookmarkUndo: Success;
    Delete: Success;
    EditComplete: unknown; // Not used yet
    EditCancel: unknown; // Not used yet
    Fork: CopyResult;
    Report: unknown; // Not used yet
    VoteDown: Success;
    VoteUp: Success;
}

/**
 * Determines which actions are available for the given object.
 * Actions follow the order: Edit, VoteUp/VoteDown, Bookmark/BookmarkUndo, Comment, Share, Donate, Stats, FindInPage, Fork, Copy, Report, Delete
 * @param object The object to determine actions for
 * @param session Current session. Many actions require a logged in user.
 * @param exclude Actions to exclude from the list (useful when other components on the page handle those actions, like a bookmark button)
 */
export function getAvailableActions(
    object: ListObject | null | undefined,
    session: Session | undefined,
    exclude: readonly ObjectAction[] = [],
): ObjectAction[] {
    if (!object) return [];
    const isLoggedIn = checkIfLoggedIn(session);
    const { canComment, canCopy, canDelete, canUpdate, canReport, canShare, canBookmark, canReact, isBookmarked, reaction } = getYou(object);
    let options: ObjectAction[] = [];
    // Check edit
    if (isLoggedIn && canUpdate) {
        options.push(ObjectAction.Edit);
    }
    // Check VoteUp/VoteDown
    if (isLoggedIn && canReact && object.__typename in ReactionFor) {
        options.push(getReactionScore(reaction) > 0 ? ObjectAction.VoteDown : ObjectAction.VoteUp);
    }
    // Check Bookmark/BookmarkUndo
    if (isLoggedIn && canBookmark && object.__typename in BookmarkFor) {
        options.push(isBookmarked ? ObjectAction.BookmarkUndo : ObjectAction.Bookmark);
    }
    // Check Comment
    if (isLoggedIn && canComment && object.__typename in CommentFor) {
        options.push(ObjectAction.Comment);
    }
    // Check Share
    if (canShare) {
        options.push(ObjectAction.Share);
    }
    // Check Donate
    //TODO
    // Check Stats
    //TODO ["Api", "Code", "Project", "Quiz", "Routine", "Standard", "Team", "User"].includes(object.__typename)
    // Can always find in page
    options.push(ObjectAction.FindInPage);
    // Check Fork
    if (isLoggedIn && canCopy && object.__typename in CopyType) {
        options.push(ObjectAction.Fork);
    }
    // Check Report
    if (isLoggedIn && canReport && object.__typename in ReportFor) {
        options.push(ObjectAction.Report);
    }
    // Check Delete
    if (isLoggedIn && canDelete && object.__typename in DeleteType) {
        options.push(ObjectAction.Delete);
    }
    // Omit excluded actions
    if (exclude) {
        options = options.filter((action) => !exclude.includes(action));
    }
    return options;
}

/**
 * Maps an ObjectAction to [labelKey, Icon, iconColor, preview]
 */
const allOptionsMap: { [key in ObjectAction]: [TranslationKeyCommon, IconInfo, string, boolean] } = ({
    [ObjectAction.Bookmark]: ["Bookmark", { name: "BookmarkOutline", type: "Common" }, "#cbae30", false],
    [ObjectAction.BookmarkUndo]: ["BookmarkUndo", { name: "BookmarkFilled", type: "Common" }, "#cbae30", false],
    [ObjectAction.Comment]: ["Comment", { name: "Comment", type: "Common" }, "default", false],
    [ObjectAction.Delete]: ["Delete", { name: "Delete", type: "Common" }, "default", false],
    [ObjectAction.Donate]: ["Donate", { name: "Donate", type: "Common" }, "default", true],
    [ObjectAction.Edit]: ["Edit", { name: "Edit", type: "Common" }, "default", false],
    [ObjectAction.FindInPage]: ["FindEllipsis", { name: "Search", type: "Common" }, "default", false],
    [ObjectAction.Fork]: ["Fork", { name: "Branch", type: "Routine" }, "default", false],
    [ObjectAction.Report]: ["Report", { name: "Report", type: "Common" }, "default", false],
    [ObjectAction.Share]: ["Share", { name: "Share", type: "Common" }, "default", false],
    [ObjectAction.Stats]: ["StatisticsShort", { name: "Stats", type: "Common" }, "default", true],
    [ObjectAction.VoteDown]: ["VoteDown", { name: "ArrowDown", type: "Common" }, "default", false],
    [ObjectAction.VoteUp]: ["VoteUp", { name: "ArrowUp", type: "Common" }, "default", false],
});

export function getActionsDisplayData(actions: ObjectAction[]): Pick<ListMenuItemData<ObjectAction>, "iconColor" | "iconInfo" | "labelKey" | "value">[] {
    return actions.map((action) => {
        const [labelKey, iconInfo, iconColor, preview] = allOptionsMap[action];
        return { labelKey, iconInfo, iconColor, preview, value: action };
    });
}
