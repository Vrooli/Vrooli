import { BotIcon, EditIcon, isOfType, Member, OrganizationIcon, ReactionFor, SvgComponent, useLocation, User, UserIcon, uuid } from "@local/shared";
import { Avatar, Box, Chip, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { CommentsButton } from "components/buttons/CommentsButton/CommentsButton";
import { ReportsButton } from "components/buttons/ReportsButton/ReportsButton";
import { VoteButton } from "components/buttons/VoteButton/VoteButton";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import Markdown from "markdown-to-jsx";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis } from "styles";
import { ObjectAction } from "utils/actions/objectActions";
import { getBookmarkFor, getCounts, getDisplay, getYou, ListObjectType, placeholderColor } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import usePress from "utils/hooks/usePress";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { getObjectEditUrl, getObjectUrl } from "utils/navigation/openObject";
import { SessionContext } from "utils/SessionContext";
import { smallHorizontalScrollbar } from "../styles";
import { TagList } from "../TagList/TagList";
import { TextLoading } from "../TextLoading/TextLoading";
import { ObjectListItemProps } from "../types";

/**
 * A list item that automatically supports most object types, with props 
 * for adding additional, type-specific content. General layout is:
 * - To the left: object icon, vote buttons, or nothing
 * - To the left, but right of icon/vote: stack of title, subtitle, any custom 
 * component(s) below subtitle, chips of various types (including for tags), and 
 * custom component(s) below chips
 * - To the right: various action buttons, including bookmark, comment, report. 
 * (On large screens, these are displayed at the bottom instead of the right.)
 * - To the right, but left of action buttons: custom component(s)
 */
export function ObjectListItemBase<T extends ListObjectType>({
    canNavigate,
    belowSubtitle,
    belowTags,
    hideUpdateButton,
    loading,
    data,
    objectType,
    onClick,
    toTheRight,
    zIndex,
}: ObjectListItemProps<T>) {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const id = useMemo(() => data?.id ?? uuid(), [data]);

    const [object, setObject] = useState<T | null | undefined>(data);
    useEffect(() => { setObject(data); }, [data]);

    const profileColors = useMemo(() => placeholderColor(), []);
    const { canBookmark, canComment, canUpdate, canReact, isBookmarked, reaction } = useMemo(() => getYou(data), [data]);
    const { subtitle, title } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);
    const { score } = useMemo(() => getCounts(data), [data]);

    // Context menu
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const handleContextMenu = useCallback((target: EventTarget) => {
        setAnchorEl(target as HTMLElement);
    }, []);
    const closeContextMenu = useCallback(() => setAnchorEl(null), []);

    const link = useMemo(() => (data && (typeof canNavigate !== "function" || canNavigate(data))) ? getObjectUrl(data) : "", [data, canNavigate]);
    const handleClick = useCallback((target: EventTarget) => {
        if (!target.id || !target.id.startsWith("list-item-")) return;
        // If data not supplied, don't open
        if (data === null) return;
        // If onClick is supplied, call it instead of navigating
        if (onClick) {
            onClick(data);
            return;
        }
        // If canNavigate is supplied, call it
        if (canNavigate) {
            const shouldContinue = canNavigate(data);
            if (shouldContinue === false) return;
        }
        // Navigate to the object's page
        setLocation(link);
    }, [data, link, onClick, canNavigate, setLocation]);

    const editUrl = useMemo(() => data ? getObjectEditUrl(data) : "", [data]);
    const handleEditClick = useCallback((event: any) => {
        event.preventDefault();
        const target = event.target;
        if (!target.id || !target.id.startsWith("edit-list-item-")) return;
        // If data not supplied, don't open
        if (!data) return;
        // If canNavigate is supplied, call it
        if (canNavigate) {
            const shouldContinue = canNavigate(data);
            if (shouldContinue === false) return;
        }
        // Navigate to the object's edit page
        setLocation(editUrl);
    }, [canNavigate, data, editUrl, setLocation]);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onRightClick: handleContextMenu,
    });

    /**
     * Left column is only shown on wide screens (if not a profile picture). It's either 
     * a vote button, an object icon, or nothing.
     */
    const leftColumn = useMemo(() => {
        // Show icons for organizations, users, and members
        if (isOfType(object, "Organization", "User", "Member")) {
            console.log("calculating left column", object);
            let Icon: SvgComponent;
            if (object.__typename === "Organization") {
                Icon = OrganizationIcon;
            } else if ((object as unknown as User).isBot || (object as unknown as Member).user?.isBot) {
                Icon = BotIcon;
            } else {
                Icon = UserIcon;
            }
            return (
                <Avatar
                    src="/broken-image.jpg" //TODO
                    sx={{
                        backgroundColor: profileColors[0],
                        width: isMobile ? "40px" : "50px",
                        height: isMobile ? "40px" : "50px",
                        pointerEvents: "none",
                    }}
                >
                    <Icon fill={profileColors[1]} width="75%" height="75%" />
                </Avatar>
            );
        }
        // Otherwise, only show on wide screens
        if (isMobile) return null;
        // Show vote buttons if supported
        if (canReact) {
            return (
                <VoteButton
                    disabled={!canReact}
                    emoji={reaction}
                    objectId={object?.id ?? ""}
                    voteFor={object?.__typename as ReactionFor}
                    score={score}
                    onChange={(newEmoji: string | null, newScore: number) => { }}
                />
            );
        }
        return null;
    }, [isMobile, object, profileColors, canReact, reaction, score]);

    /**
     * Action buttons are shown as a column on wide screens, and 
     * a row on mobile. It displays 
     * the star, comments, and reports buttons.
     */
    const actionButtons = useMemo(() => {
        const reportsCount: number = getCounts(object).reports;
        const { bookmarkFor, starForId } = getBookmarkFor(object);
        return (
            <Stack
                direction={isMobile ? "row" : "column"}
                spacing={1}
                sx={{
                    pointerEvents: "none",
                    justifyContent: isMobile ? "right" : "center",
                    alignItems: isMobile ? "center" : "start",
                }}
            >
                {!hideUpdateButton && canUpdate &&
                    <Box
                        id={`edit-list-item-button-${id}`}
                        component="a"
                        href={editUrl}
                        onClick={handleEditClick}
                        sx={{
                            display: "flex",
                            justifyContent: "center",
                            alignItems: "center",
                            cursor: "pointer",
                            pointerEvents: "all",
                            paddingBottom: isMobile ? "0px" : "4px",
                        }}>
                        <EditIcon id={`edit-list-item-icon${id}`} fill={palette.secondary.main} />
                    </Box>}
                {/* Add upvote/downvote if mobile */}
                {isMobile && canReact && (
                    <VoteButton
                        direction='row'
                        disabled={!canReact}
                        emoji={reaction}
                        objectId={object?.id ?? ""}
                        voteFor={object?.__typename as ReactionFor}
                        score={score}
                        onChange={(newEmoji: string | null, newScore: number) => { }}
                    />
                )}
                {canBookmark && bookmarkFor && <BookmarkButton
                    disabled={!canBookmark}
                    objectId={starForId}
                    bookmarkFor={bookmarkFor}
                    isBookmarked={isBookmarked}
                    bookmarks={getCounts(object).bookmarks}
                    zIndex={zIndex}
                />}
                {canComment && (<CommentsButton
                    commentsCount={getCounts(object).comments}
                    disabled={!canComment}
                    object={object}
                />)}
                {!isOfType(object, "RunRoutine", "RunProject") && reportsCount > 0 && <ReportsButton
                    reportsCount={reportsCount}
                    object={object}
                />}
            </Stack>
        );
    }, [object, isMobile, hideUpdateButton, canUpdate, id, editUrl, handleEditClick, palette.secondary.main, canReact, reaction, score, canBookmark, isBookmarked, zIndex, canComment]);

    const actionData = useObjectActions({
        canNavigate,
        object,
        objectType,
        setLocation,
        setObject,
    });

    return (
        <>
            {/* Context menu */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={anchorEl}
                exclude={[ObjectAction.Comment, ObjectAction.FindInPage]} // Find in page only relevant when viewing object - not in list. And shouldn't really comment without viewing full page
                object={object}
                onClose={closeContextMenu}
                zIndex={zIndex + 1}
            />
            {/* List item */}
            <ListItem
                id={`list-item-${id}`}
                disablePadding
                button
                component={link ? "a" : "div"}
                href={link}
                {...pressEvents}
                sx={{
                    display: "flex",
                    background: palette.background.paper,
                    padding: "8px 16px",
                    cursor: "pointer",
                    borderBottom: `1px solid ${palette.divider}`,
                }}
            >
                {leftColumn}
                <Stack
                    direction="column"
                    spacing={1}
                    pl={2}
                    sx={{
                        width: "-webkit-fill-available",
                        display: "grid",
                        pointerEvents: "none",
                    }}
                >
                    {/* Title */}
                    {loading ? <TextLoading /> :
                        (
                            <Stack id={`list-item-title-stack-${id}`} direction="row" spacing={1} sx={{
                                ...smallHorizontalScrollbar(palette),
                            }}>
                                <ListItemText
                                    primary={title}
                                    sx={{
                                        ...multiLineEllipsis(1),
                                        lineBreak: "anywhere",
                                        pointerEvents: "none",
                                    }}
                                />
                            </Stack>
                        )
                    }
                    {/* Subtitle */}
                    {loading ? <TextLoading /> : <Markdown
                        style={{ ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: "none" }}
                    >{subtitle}</Markdown>}
                    {/* Any custom components to display below the subtitle */}
                    {belowSubtitle}
                    <Stack direction="row" spacing={1} sx={{ pointerEvents: "none" }}>
                        {/* Incomplete chip */}
                        {
                            data && data.__typename !== "Reminder" && (data as any).isComplete === false && <Tooltip placement="top" title={t("MarkedIncomplete")}>
                                <Chip
                                    label="Incomplete"
                                    size="small"
                                    sx={{
                                        backgroundColor: palette.error.main,
                                        color: palette.error.contrastText,
                                        width: "fit-content",
                                    }} />
                            </Tooltip>
                        }
                        {/* Internal chip */}
                        {
                            data && (data as any).isInternal === true && <Tooltip placement="top" title={t("MarkedInternal")}>
                                <Chip
                                    label="Internal"
                                    size="small"
                                    sx={{
                                        backgroundColor: palette.warning.main,
                                        color: palette.error.contrastText,
                                        width: "fit-content",
                                    }} />
                            </Tooltip>
                        }
                        {/* Tags */}
                        {Array.isArray((data as any)?.tags) && (data as any)?.tags.length > 0 &&
                            <TagList
                                parentId={data?.id ?? ""}
                                tags={(data as any).tags}
                                sx={{ ...smallHorizontalScrollbar(palette) }}
                            />}
                        {/* Any custom components to display below tags */}
                        {belowTags}
                    </Stack>
                    {/* Action buttons if mobile */}
                    {isMobile && actionButtons}
                </Stack>
                {!isMobile && actionButtons}
                {/* Custom components displayed on the right */}
                {toTheRight}
            </ListItem>
        </>
    );
}
