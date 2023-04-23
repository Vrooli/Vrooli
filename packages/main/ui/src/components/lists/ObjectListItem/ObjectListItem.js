import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { RunStatus } from "@local/consts";
import { EditIcon, OrganizationIcon, UserIcon } from "@local/icons";
import { isOfType } from "@local/utils";
import { uuid } from "@local/uuid";
import { Box, Chip, LinearProgress, ListItem, ListItemText, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis } from "../../../styles";
import { ObjectAction } from "../../../utils/actions/objectActions";
import { getBookmarkFor, getCounts, getDisplay, getYou, placeholderColor } from "../../../utils/display/listTools";
import { getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import usePress from "../../../utils/hooks/usePress";
import { useWindowSize } from "../../../utils/hooks/useWindowSize";
import { getObjectEditUrl, getObjectUrl } from "../../../utils/navigation/openObject";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
import { BookmarkButton } from "../../buttons/BookmarkButton/BookmarkButton";
import { CommentsButton } from "../../buttons/CommentsButton/CommentsButton";
import { ReportsButton } from "../../buttons/ReportsButton/ReportsButton";
import { VoteButton } from "../../buttons/VoteButton/VoteButton";
import { ObjectActionMenu } from "../../dialogs/ObjectActionMenu/ObjectActionMenu";
import { RoleList } from "../RoleList/RoleList";
import { smallHorizontalScrollbar } from "../styles";
import { TagList } from "../TagList/TagList";
import { TextLoading } from "../TextLoading/TextLoading";
export function CompletionBar(props) {
    return (_jsxs(Box, { sx: { display: "flex", alignItems: "center", pointerEvents: "none" }, children: [_jsx(Box, { sx: { width: "100%", mr: 1 }, children: _jsx(LinearProgress, { variant: props.variant, ...props, sx: { borderRadius: 1, height: 8 } }) }), _jsx(Box, { sx: { minWidth: 35 }, children: _jsx(Typography, { variant: "body2", color: "text.secondary", children: `${Math.round(props.value)}%` }) })] }));
}
export function ObjectListItem({ beforeNavigation, hideUpdateButton, index, loading, data, objectType, zIndex, }) {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const id = useMemo(() => data?.id ?? uuid(), [data]);
    const [object, setObject] = useState(data);
    useEffect(() => { setObject(data); }, [data]);
    const profileColors = useMemo(() => placeholderColor(), []);
    const { canComment, canUpdate, canReact, canBookmark, isBookmarked, reaction } = useMemo(() => getYou(data), [data]);
    const { subtitle, title } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);
    const { score } = useMemo(() => getCounts(data), [data]);
    const [anchorEl, setAnchorEl] = useState(null);
    const handleContextMenu = useCallback((target) => {
        setAnchorEl(target);
    }, []);
    const closeContextMenu = useCallback(() => setAnchorEl(null), []);
    const link = useMemo(() => data ? getObjectUrl(data) : "", [data]);
    const handleClick = useCallback((target) => {
        if (!target.id || !target.id.startsWith("list-item-"))
            return;
        if (data === null || link.length === 0)
            return;
        if (beforeNavigation) {
            const shouldContinue = beforeNavigation(data);
            if (shouldContinue === false)
                return;
        }
        setLocation(link);
    }, [link, beforeNavigation, setLocation, data]);
    const editUrl = useMemo(() => data ? getObjectEditUrl(data) : "", [data]);
    const handleEditClick = useCallback((event) => {
        event.preventDefault();
        const target = event.target;
        if (!target.id || !target.id.startsWith("edit-list-item-"))
            return;
        if (!data)
            return;
        if (beforeNavigation) {
            const shouldContinue = beforeNavigation(data);
            if (shouldContinue === false)
                return;
        }
        setLocation(editUrl);
    }, [beforeNavigation, data, editUrl, setLocation]);
    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onRightClick: handleContextMenu,
    });
    const leftColumn = useMemo(() => {
        if (isMobile && !isOfType(object, "Organization", "User"))
            return null;
        switch (object?.__typename) {
            case "Organization":
            case "User":
                const Icon = object?.__typename === "Organization" ? OrganizationIcon : UserIcon;
                return (_jsx(Box, { width: isMobile ? "40px" : "50px", minWidth: isMobile ? "40px" : "50px", height: isMobile ? "40px" : "50px", marginBottom: isMobile ? "auto" : "unset", borderRadius: '100%', bgcolor: profileColors[0], justifyContent: 'center', alignItems: 'center', sx: {
                        display: "flex",
                        pointerEvents: "none",
                    }, children: _jsx(Icon, { fill: profileColors[1], width: isMobile ? "25px" : "35px", height: isMobile ? "25px" : "35px" }) }));
            case "Project":
            case "Routine":
            case "Standard":
                return (_jsx(VoteButton, { disabled: !canReact, emoji: reaction, objectId: object?.id ?? "", voteFor: object?.__typename, score: score, onChange: (newEmoji, newScore) => { } }));
            default:
                return null;
        }
    }, [isMobile, object, profileColors, canReact, reaction, score]);
    const actionButtons = useMemo(() => {
        const commentableObjects = ["Project", "Routine", "Standard"];
        const reportsCount = getCounts(object).reports;
        const { bookmarkFor, starForId } = getBookmarkFor(object);
        return (_jsxs(Stack, { direction: isMobile ? "row" : "column", spacing: 1, sx: {
                pointerEvents: "none",
                justifyContent: isMobile ? "right" : "center",
                alignItems: isMobile ? "center" : "start",
            }, children: [!hideUpdateButton && canUpdate &&
                    _jsx(Box, { id: `edit-list-item-button-${id}`, component: "a", href: editUrl, onClick: handleEditClick, sx: {
                            display: "flex",
                            justifyContent: "center",
                            alignItems: "center",
                            cursor: "pointer",
                            pointerEvents: "all",
                            paddingBottom: isMobile ? "0px" : "4px",
                        }, children: _jsx(EditIcon, { id: `edit-list-item-icon${id}`, fill: palette.secondary.main }) }), isMobile && isOfType(object, "Api", "ApiVersion", "Comment", "Issue", "Note", "NoteVersion", "Post", "Project", "ProjectVersion", "Question", "QuestionAnswer", "Quiz", "Routine", "RoutineVersion", "SmartContract", "SmartContractVersion", "Standard", "StandardVersion") && (_jsx(VoteButton, { direction: 'row', disabled: !canReact, emoji: reaction, objectId: object?.id ?? "", voteFor: object?.__typename, score: score, onChange: (newEmoji, newScore) => { } })), bookmarkFor && _jsx(BookmarkButton, { disabled: !canBookmark, objectId: starForId, bookmarkFor: bookmarkFor, isBookmarked: isBookmarked, bookmarks: getCounts(object).bookmarks }), commentableObjects.includes(object?.__typename ?? "") && (_jsx(CommentsButton, { commentsCount: getCounts(object).comments, disabled: !canComment, object: object })), !isOfType(object, "RunRoutine", "RunProject") && reportsCount > 0 && _jsx(ReportsButton, { reportsCount: reportsCount, object: object })] }));
    }, [object, isMobile, hideUpdateButton, canUpdate, id, editUrl, handleEditClick, palette.secondary.main, canReact, reaction, score, canBookmark, isBookmarked, canComment]);
    const progressBar = useMemo(() => {
        if (!isOfType(object, "RunProject", "RunRoutine"))
            return null;
        const completedComplexity = object.completedComplexity;
        const totalComplexity = object.projectVersion?.complexity ?? object.routineVersion?.complexity ?? null;
        const percentComplete = object.status === RunStatus.Completed ? 100 :
            (completedComplexity && totalComplexity) ?
                Math.min(Math.round(completedComplexity / totalComplexity * 100), 100) :
                0;
        return (_jsx(CompletionBar, { color: "secondary", variant: loading ? "indeterminate" : "determinate", value: percentComplete, sx: { height: "15px" } }));
    }, [loading, object]);
    const actionData = useObjectActions({
        beforeNavigation,
        object,
        objectType,
        setLocation,
        setObject,
    });
    return (_jsxs(_Fragment, { children: [_jsx(ObjectActionMenu, { actionData: actionData, anchorEl: anchorEl, exclude: [ObjectAction.Comment, ObjectAction.FindInPage], object: object, onClose: closeContextMenu, zIndex: zIndex + 1 }), _jsxs(ListItem, { id: `list-item-${id}`, disablePadding: true, button: true, component: "a", href: link, ...pressEvents, onClick: (e) => { e.preventDefault(); }, sx: {
                    display: "flex",
                    background: palette.background.paper,
                    padding: "8px 16px",
                    cursor: "pointer",
                    borderBottom: `1px solid ${palette.divider}`,
                }, children: [leftColumn, _jsxs(Stack, { direction: "column", spacing: 1, pl: 2, sx: {
                            width: "-webkit-fill-available",
                            display: "grid",
                            pointerEvents: "none",
                        }, children: [loading ? _jsx(TextLoading, {}) :
                                (_jsx(Stack, { id: `list-item-title-stack-${id}`, direction: "row", spacing: 1, sx: {
                                        ...smallHorizontalScrollbar(palette),
                                    }, children: _jsx(ListItemText, { primary: title, sx: {
                                            ...multiLineEllipsis(1),
                                            lineBreak: "anywhere",
                                            pointerEvents: "none",
                                        } }) })), loading ? _jsx(TextLoading, {}) : _jsx(ListItemText, { primary: subtitle, sx: { ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: "none" } }), progressBar, _jsxs(Stack, { direction: "row", spacing: 1, sx: { pointerEvents: "none" }, children: [data && data.isComplete === false && _jsx(Tooltip, { placement: "top", title: t("MarkedIncomplete"), children: _jsx(Chip, { label: "Incomplete", size: "small", sx: {
                                                backgroundColor: palette.error.main,
                                                color: palette.error.contrastText,
                                                width: "fit-content",
                                            } }) }), data && data.isInternal === true && _jsx(Tooltip, { placement: "top", title: t("MarkedInternal"), children: _jsx(Chip, { label: "Internal", size: "small", sx: {
                                                backgroundColor: palette.warning.main,
                                                color: palette.error.contrastText,
                                                width: "fit-content",
                                            } }) }), Array.isArray(data?.tags) && data?.tags.length > 0 &&
                                        _jsx(TagList, { parentId: data?.id ?? "", tags: data.tags, sx: { ...smallHorizontalScrollbar(palette) } }), isOfType(object, "Member") && data?.roles?.length > 0 &&
                                        _jsx(RoleList, { roles: data.roles, sx: { ...smallHorizontalScrollbar(palette) } })] }), isMobile && actionButtons] }), !isMobile && actionButtons] })] }));
}
//# sourceMappingURL=ObjectListItem.js.map