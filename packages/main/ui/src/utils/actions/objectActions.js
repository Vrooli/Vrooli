import { BookmarkFilledIcon, BookmarkOutlineIcon, BranchIcon, DeleteIcon, DonateIcon, DownvoteWideIcon, EditIcon, ReplyIcon, ReportIcon, SearchIcon, ShareIcon, StatsIcon, UpvoteWideIcon } from "@local/icons";
import { getReactionScore } from "@local/utils";
import { checkIfLoggedIn } from "../authentication/session";
import { getYou } from "../display/listTools";
export var ObjectAction;
(function (ObjectAction) {
    ObjectAction["Bookmark"] = "Bookmark";
    ObjectAction["BookmarkUndo"] = "BookmarkUndo";
    ObjectAction["Comment"] = "Comment";
    ObjectAction["Delete"] = "Delete";
    ObjectAction["Donate"] = "Donate";
    ObjectAction["Edit"] = "Edit";
    ObjectAction["FindInPage"] = "FindInPage";
    ObjectAction["Fork"] = "Fork";
    ObjectAction["Report"] = "Report";
    ObjectAction["Share"] = "Share";
    ObjectAction["Stats"] = "Stats";
    ObjectAction["VoteDown"] = "VoteDown";
    ObjectAction["VoteUp"] = "VoteUp";
})(ObjectAction || (ObjectAction = {}));
export var ObjectActionComplete;
(function (ObjectActionComplete) {
    ObjectActionComplete["Bookmark"] = "Bookmark";
    ObjectActionComplete["BookmarkUndo"] = "BookmarkUndo";
    ObjectActionComplete["Delete"] = "Delete";
    ObjectActionComplete["EditComplete"] = "EditComplete";
    ObjectActionComplete["EditCancel"] = "EditCanel";
    ObjectActionComplete["Fork"] = "Fork";
    ObjectActionComplete["Report"] = "Report";
    ObjectActionComplete["VoteDown"] = "VoteDown";
    ObjectActionComplete["VoteUp"] = "VoteUp";
})(ObjectActionComplete || (ObjectActionComplete = {}));
export const getAvailableActions = (object, session, exclude = []) => {
    if (!object)
        return [];
    console.log("action getavailableactions", session);
    const isLoggedIn = checkIfLoggedIn(session);
    const { canComment, canCopy, canDelete, canUpdate, canReport, canShare, canBookmark, canReact, isBookmarked, reaction } = getYou(object);
    let options = [];
    if (isLoggedIn && canUpdate) {
        options.push(ObjectAction.Edit);
    }
    if (isLoggedIn && canReact) {
        options.push(getReactionScore(reaction) > 0 ? ObjectAction.VoteDown : ObjectAction.VoteUp);
    }
    if (isLoggedIn && canBookmark) {
        options.push(isBookmarked ? ObjectAction.BookmarkUndo : ObjectAction.Bookmark);
    }
    if (isLoggedIn && canComment) {
        options.push(ObjectAction.Comment);
    }
    if (canShare) {
        options.push(ObjectAction.Share);
    }
    options.push(ObjectAction.FindInPage);
    if (isLoggedIn && canCopy) {
        options.push(ObjectAction.Fork);
    }
    if (isLoggedIn && canReport) {
        options.push(ObjectAction.Report);
    }
    if (isLoggedIn && canDelete) {
        options.push(ObjectAction.Delete);
    }
    if (exclude) {
        options = options.filter((action) => !exclude.includes(action));
    }
    return options;
};
const allOptionsMap = ({
    [ObjectAction.Bookmark]: ["Bookmark", BookmarkOutlineIcon, "#cbae30", false],
    [ObjectAction.BookmarkUndo]: ["Unstar", BookmarkFilledIcon, "#cbae30", false],
    [ObjectAction.Comment]: ["Comment", ReplyIcon, "default", false],
    [ObjectAction.Delete]: ["Delete", DeleteIcon, "default", false],
    [ObjectAction.Donate]: ["Donate", DonateIcon, "default", true],
    [ObjectAction.Edit]: ["Edit", EditIcon, "default", false],
    [ObjectAction.FindInPage]: ["Find...", SearchIcon, "default", false],
    [ObjectAction.Fork]: ["Fork", BranchIcon, "default", false],
    [ObjectAction.Report]: ["Report", ReportIcon, "default", false],
    [ObjectAction.Share]: ["Share", ShareIcon, "default", false],
    [ObjectAction.Stats]: ["Stats", StatsIcon, "default", true],
    [ObjectAction.VoteDown]: ["Downvote", DownvoteWideIcon, "default", false],
    [ObjectAction.VoteUp]: ["Upvote", UpvoteWideIcon, "default", false],
});
export const getActionsDisplayData = (actions) => {
    return actions.map((action) => {
        const [label, Icon, iconColor, preview] = allOptionsMap[action];
        return { label, Icon, iconColor, preview, value: action };
    });
};
//# sourceMappingURL=objectActions.js.map