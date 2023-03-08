import { ListMenuItemData } from "components/dialogs/types";
import { getYou, ListObjectType } from "utils/display";
import { BranchIcon, DeleteIcon, DonateIcon, DownvoteWideIcon, EditIcon, ReplyIcon, ReportIcon, SearchIcon, ShareIcon, BookmarkFilledIcon, BookmarkOutlineIcon, StatsIcon, SvgComponent, UpvoteWideIcon } from "@shared/icons";
import { Session } from "@shared/consts";
import { checkIfLoggedIn } from "utils/authentication";

/**
 * All available actions an object can possibly have
 */
 export enum ObjectAction {
    Bookmark = 'Bookmark',
    BookmarkUndo = 'BookmarkUndo',
    Comment = 'Comment',
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
    Bookmark = 'Bookmark',
    BookmarkUndo = 'BookmarkUndo',
    Delete = 'Delete',
    EditComplete = 'EditComplete',
    EditCancel = 'EditCanel',
    Fork = 'Fork',
    Report = 'Report',
    VoteDown = 'VoteDown',
    VoteUp = 'VoteUp',
}

/**
 * Determines which actions are available for the given object.
 * Actions follow the order: Edit, VoteUp/VoteDown, Star/StarUndo, Comment, Share, Donate, Stats, FindInPage, Fork, Copy, Report, Delete
 * @param object The object to determine actions for
 * @param session Current session. Many actions require a logged in user.
 * @param exclude Actions to exclude from the list (useful when other components on the page handle those actions, like a star button)
 */
export const getAvailableActions = (object: ListObjectType | null | undefined, session: Session | undefined, exclude: ObjectAction[] = []): ObjectAction[] => {
    if (!object) return [];
    console.log('action getavailableactions', session)
    const isLoggedIn = checkIfLoggedIn(session);
    const { canComment, canCopy, canDelete, canUpdate, canReport, canShare, canBookmark, canVote, isBookmarked, isUpvoted } = getYou(object)
    let options: ObjectAction[] = [];
    // Check edit
    if (isLoggedIn && canUpdate) {
        options.push(ObjectAction.Edit);
    }
    // Check VoteUp/VoteDown
    if (isLoggedIn && canVote) {
        options.push(isUpvoted ? ObjectAction.VoteDown : ObjectAction.VoteUp);
    }
    // Check Bookmark/BookmarkUndo
    if (isLoggedIn && canBookmark) {
        options.push(isBookmarked ? ObjectAction.BookmarkUndo : ObjectAction.Bookmark);
    }
    // Check Comment
    if (isLoggedIn && canComment) {
        options.push(ObjectAction.Comment);
    }
    // Check Share
    if (canShare) {
        options.push(ObjectAction.Share);
    }
    // Check Donate
    //TODO
    // Check Stats
    //TODO
    // Can always find in page
    options.push(ObjectAction.FindInPage);
    // Check Fork
    if (isLoggedIn && canCopy) {
        options.push(ObjectAction.Fork);
    }
    // Check Report
    if (isLoggedIn && canReport) {
        options.push(ObjectAction.Report);
    }
    // Check Delete
    if (isLoggedIn && canDelete) {
        options.push(ObjectAction.Delete);
    }
    // Omit excluded actions
    if (exclude) {
        options = options.filter((action) => !exclude.includes(action));
    }
    return options;
}

/**
 * Maps an ObjectAction to [label, Icon, iconColor, preview]
 */
 const allOptionsMap: { [key in ObjectAction]: [string, SvgComponent, string, boolean] } = ({
    [ObjectAction.Bookmark]: ['Bookmark', BookmarkOutlineIcon, "#cbae30", false],
    [ObjectAction.BookmarkUndo]: ['Unstar', BookmarkFilledIcon, "#cbae30", false],
    [ObjectAction.Comment]: ['Comment', ReplyIcon, 'default', false],
    [ObjectAction.Delete]: ['Delete', DeleteIcon, "default", false],
    [ObjectAction.Donate]: ['Donate', DonateIcon, "default", true],
    [ObjectAction.Edit]: ['Edit', EditIcon, "default", false],
    [ObjectAction.FindInPage]: ['Find...', SearchIcon, "default", false],
    [ObjectAction.Fork]: ['Fork', BranchIcon, "default", false],
    [ObjectAction.Report]: ['Report', ReportIcon, "default", false],
    [ObjectAction.Share]: ['Share', ShareIcon, "default", false],
    [ObjectAction.Stats]: ['Stats', StatsIcon, "default", true],
    [ObjectAction.VoteDown]: ['Downvote', DownvoteWideIcon, "default", false],
    [ObjectAction.VoteUp]: ['Upvote', UpvoteWideIcon, "default", false],
})

export const getActionsDisplayData = (actions: ObjectAction[]): Pick<ListMenuItemData<any>, 'Icon' | 'iconColor' | 'label' | 'preview' | 'value'>[] => {
    return actions.map((action) => {
        const [label, Icon, iconColor, preview] = allOptionsMap[action];
        return { label, Icon, iconColor, preview, value: action };
    });
}