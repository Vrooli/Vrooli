import { BookmarkFor, CommentFor, CommonKey, CopyType, DeleteType, getReactionScore, ReactionFor, ReportFor, Session } from "@local/shared";
import { ListMenuItemData } from "components/dialogs/types";
import { BookmarkFilledIcon, BookmarkOutlineIcon, BranchIcon, DeleteIcon, DonateIcon, DownvoteWideIcon, EditIcon, ReplyIcon, ReportIcon, SearchIcon, ShareIcon, StatsIcon, UpvoteWideIcon } from "icons";
import { SvgComponent } from "types";
import { checkIfLoggedIn } from "utils/authentication/session";
import { getYou, ListObject } from "utils/display/listTools";

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

/**
 * Determines which actions are available for the given object.
 * Actions follow the order: Edit, VoteUp/VoteDown, Bookmark/BookmarkUndo, Comment, Share, Donate, Stats, FindInPage, Fork, Copy, Report, Delete
 * @param object The object to determine actions for
 * @param session Current session. Many actions require a logged in user.
 * @param exclude Actions to exclude from the list (useful when other components on the page handle those actions, like a bookmark button)
 */
export const getAvailableActions = (object: ListObject | null | undefined, session: Session | undefined, exclude: ObjectAction[] = []): ObjectAction[] => {
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
    //TODO ["Api", "Organization", "Project", "Quiz", "Routine", "SmartContract", "Standard", "User"].includes(object.__typename)
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
};

/**
 * Maps an ObjectAction to [labelKey, Icon, iconColor, preview]
 */
const allOptionsMap: { [key in ObjectAction]: [CommonKey, SvgComponent, string, boolean] } = ({
    [ObjectAction.Bookmark]: ["Bookmark", BookmarkOutlineIcon, "#cbae30", false],
    [ObjectAction.BookmarkUndo]: ["BookmarkUndo", BookmarkFilledIcon, "#cbae30", false],
    [ObjectAction.Comment]: ["Comment", ReplyIcon, "default", false],
    [ObjectAction.Delete]: ["Delete", DeleteIcon, "default", false],
    [ObjectAction.Donate]: ["Donate", DonateIcon, "default", true],
    [ObjectAction.Edit]: ["Edit", EditIcon, "default", false],
    [ObjectAction.FindInPage]: ["FindEllipsis", SearchIcon, "default", false],
    [ObjectAction.Fork]: ["Fork", BranchIcon, "default", false],
    [ObjectAction.Report]: ["Report", ReportIcon, "default", false],
    [ObjectAction.Share]: ["Share", ShareIcon, "default", false],
    [ObjectAction.Stats]: ["StatisticsShort", StatsIcon, "default", true],
    [ObjectAction.VoteDown]: ["VoteDown", DownvoteWideIcon, "default", false],
    [ObjectAction.VoteUp]: ["VoteUp", UpvoteWideIcon, "default", false],
});

export const getActionsDisplayData = (actions: ObjectAction[]): Pick<ListMenuItemData<ObjectAction>, "Icon" | "iconColor" | "labelKey" | "value">[] => {
    return actions.map((action) => {
        const [labelKey, Icon, iconColor, preview] = allOptionsMap[action];
        return { labelKey, Icon, iconColor, preview, value: action };
    });
};
