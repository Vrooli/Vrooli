import { Chat, isOfType, Member, ReactionFor, User, uuid } from "@local/shared";
import { Avatar, Box, Chip, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { CommentsButton } from "components/buttons/CommentsButton/CommentsButton";
import { ReportsButton } from "components/buttons/ReportsButton/ReportsButton";
import { VoteButton } from "components/buttons/VoteButton/VoteButton";
import { ProfileGroup } from "components/ProfileGroup/ProfileGroup";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { SessionContext } from "contexts/SessionContext";
import usePress from "hooks/usePress";
import { useWindowSize } from "hooks/useWindowSize";
import { BotIcon, EditIcon, OrganizationIcon, UserIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { multiLineEllipsis } from "styles";
import { SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { setCookiePartialData } from "utils/cookies";
import { extractImageUrl } from "utils/display/imageTools";
import { getBookmarkFor, getCounts, getDisplay, getYou, ListObject, placeholderColor } from "utils/display/listTools";
import { fontSizeToPixels } from "utils/display/textTools";
import { getUserLanguages } from "utils/display/translationTools";
import { getObjectEditUrl, getObjectUrl } from "utils/navigation/openObject";
import { TagList } from "../TagList/TagList";
import { TextLoading } from "../TextLoading/TextLoading";
import { ObjectListItemProps } from "../types";

const LIST_PREFIX = "list-item-";
const EDIT_PREFIX = "edit-list-item-";

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
export function ObjectListItemBase<T extends ListObject>({
    canNavigate,
    belowSubtitle,
    belowTags,
    handleContextMenu,
    hideUpdateButton,
    loading,
    data,
    onClick,
    subtitleOverride,
    titleOverride,
    toTheRight,
}: ObjectListItemProps<T>) {
    const session = useContext(SessionContext);
    const { breakpoints, palette, typography } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const id = useMemo(() => data?.id ?? uuid(), [data]);

    const [object, setObject] = useState<T | null | undefined>(data);
    useEffect(() => { setObject(data); }, [data]);

    const profileColors = useMemo(() => placeholderColor(), []);
    const { canBookmark, canComment, canUpdate, canReact, isBookmarked, reaction } = useMemo(() => getYou(data), [data]);
    const { subtitle, title, adornments } = useMemo(() => getDisplay(data, getUserLanguages(session), palette), [data, session]);
    const { score } = useMemo(() => getCounts(data), [data]);

    const link = useMemo(() => (
        data &&
        (typeof canNavigate !== "function" || canNavigate(data))) &&
        typeof onClick !== "function" ?
        getObjectUrl(data) :
        "", [data, canNavigate, onClick]);
    const handleClick = useCallback((target: EventTarget) => {
        if (!target.id || !target.id.startsWith(LIST_PREFIX)) return;
        // If data not supplied, don't open
        if (data === null) return;
        // If onClick is supplied, call it instead of navigating
        if (typeof onClick === "function") {
            onClick(data);
            return;
        }
        // If canNavigate is supplied, call it
        if (canNavigate) {
            const shouldContinue = canNavigate(data);
            if (shouldContinue === false) return;
        }
        // Store object in local storage, so we can display it while the full data loads
        setCookiePartialData(data, "list");
        // Navigate to the object's page
        setLocation(link);
    }, [data, link, onClick, canNavigate, setLocation]);

    const editUrl = useMemo(() => data ? getObjectEditUrl(data) : "", [data]);
    const handleEditClick = useCallback((event: any) => {
        event.preventDefault();
        const target = event.target;
        if (!target.id || !target.id.startsWith(EDIT_PREFIX)) return;
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
        onLongPress: (target) => { handleContextMenu(target, data); },
        onClick: handleClick,
        onRightClick: (target) => { handleContextMenu(target, data); },
    });

    /**
     * Left column is only shown on wide screens (if not a profile picture). It's either 
     * a vote button, an object icon, or nothing.
     */
    const leftColumn = useMemo(() => {
        // Show icons for organizations, users, and members
        if (isOfType(object, "Organization", "User", "Member")) {
            const isBot = (object as unknown as User).isBot || (object as unknown as Member).user?.isBot || (object as unknown as Chat).participants?.[0]?.user?.isBot;
            let Icon: SvgComponent;
            if (object.__typename === "Organization") {
                Icon = OrganizationIcon;
            } else if (isBot) {
                Icon = BotIcon;
            } else {
                Icon = UserIcon;
            }
            return (
                <Avatar
                    src={extractImageUrl((object as unknown as { profileImage: string }).profileImage, (object as unknown as { updated_at: string }).updated_at, 50)}
                    alt={`${object.name}'s profile picture`}
                    sx={{
                        backgroundColor: profileColors[0],
                        width: isMobile ? "40px" : "50px",
                        height: isMobile ? "40px" : "50px",
                        pointerEvents: "none",
                        // Bots show up as squares, to distinguish them from users
                        ...(isBot ? { borderRadius: "8px" } : {}),
                    }}
                >
                    <Icon fill={profileColors[1]} width="75%" height="75%" />
                </Avatar>
            );
        }
        // Show multiple icons for chats
        if (isOfType(object, "Chat")) {
            // Filter yourself out of participants
            const participants = (object as unknown as Chat).participants?.filter(p => p.user?.id !== getCurrentUser(session).id) ?? [];
            // If no participants, show nothing
            if (participants.length === 0) return null;
            // If only one participant, show their profile picture instead of a group
            if (participants.length === 1) {
                return (
                    <Avatar
                        src={extractImageUrl((participants[0]?.user as unknown as { profileImage: string }).profileImage, (participants[0]?.user as unknown as { updated_at: string }).updated_at, 50)}
                        alt={`${(participants[0]?.user as unknown as { name: string }).name}'s profile picture`}
                        sx={{
                            backgroundColor: profileColors[0],
                            width: isMobile ? "40px" : "50px",
                            height: isMobile ? "40px" : "50px",
                            pointerEvents: "none",
                            // Bots show up as squares, to distinguish them from users
                            ...(participants[0]?.user?.isBot ? { borderRadius: "8px" } : {}),
                        }}
                    />
                );
            }
            // Otherwise, show a group
            return <ProfileGroup users={participants.map(p => p.user)} />;
        }
        // Otherwise, only show on wide screens
        if (isMobile) return null;
        // Show vote buttons if supported
        if (canReact && object) {
            return (
                <VoteButton
                    disabled={!canReact}
                    emoji={reaction}
                    objectId={object?.id ?? ""}
                    voteFor={object.__typename as ReactionFor}
                    score={score}
                    onChange={(newEmoji: string | null, newScore: number) => { }}
                />
            );
        }
        return null;
    }, [isMobile, object, profileColors, canReact, reaction, score, session]);

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
                    alignItems: "center",
                }}
            >
                {!hideUpdateButton && canUpdate &&
                    <Box
                        id={`${EDIT_PREFIX}button-${id}`}
                        component="a"
                        aria-label={t("Edit")}
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
                        <EditIcon id={`${EDIT_PREFIX}icon-${id}`} fill={palette.secondary.main} />
                    </Box>}
                {/* Add upvote/downvote if mobile */}
                {isMobile && canReact && object && (
                    <VoteButton
                        direction='row'
                        disabled={!canReact}
                        emoji={reaction}
                        objectId={object?.id ?? ""}
                        voteFor={object.__typename as ReactionFor}
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
    }, [object, isMobile, hideUpdateButton, canUpdate, id, t, editUrl, handleEditClick, palette.secondary.main, canReact, reaction, score, canBookmark, isBookmarked, canComment]);

    const titleId = `${LIST_PREFIX}title-stack-${id}`
    return (
        <>
            {/* List item */}
            <ListItem
                id={`${LIST_PREFIX}${id}`}
                disablePadding
                button
                component={link.length > 0 ? "a" : "div"}
                href={link.length > 0 ? link : undefined}
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
                            <Stack id={titleId} direction="row" spacing={0.5} sx={{
                                lineBreak: "auto",
                                wordBreak: "break-word",
                                pointerEvents: "none",
                            }}>
                                <ListItemText primary={titleOverride ?? title} sx={{ display: "contents" }} />
                                {adornments.map((Adornment) => (
                                    <Box sx={{
                                        width: fontSizeToPixels(typography.body1.fontSize ?? "1rem", titleId) * Number(typography.body1.lineHeight ?? "1.5"),
                                        height: fontSizeToPixels(typography.body1.fontSize ?? "1rem", titleId) * Number(typography.body1.lineHeight ?? "1.5"),
                                    }}>
                                        {Adornment}
                                    </Box>
                                ))}
                            </Stack>
                        )
                    }
                    {/* Subtitle */}
                    {loading ? <TextLoading /> : <MarkdownDisplay
                        content={subtitleOverride ?? subtitle}
                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: "none" }}
                    />}
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
