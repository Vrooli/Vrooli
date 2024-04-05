import { Chat, ChatInvite, ChatParticipant, isOfType, Meeting, Member, MemberInvite, ReactionFor, uuid } from "@local/shared";
import { Avatar, Box, Chip, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { CommentsButton } from "components/buttons/CommentsButton/CommentsButton";
import { ReportsButton } from "components/buttons/ReportsButton/ReportsButton";
import { VoteButton } from "components/buttons/VoteButton/VoteButton";
import { ProfileGroup } from "components/ProfileGroup/ProfileGroup";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { SessionContext } from "contexts/SessionContext";
import usePress from "hooks/usePress";
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
import { fontSizeToPixels } from "utils/display/stringTools";
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
    handleToggleSelect,
    hideUpdateButton,
    isMobile,
    isSelecting,
    isSelected,
    loading,
    data,
    onClick,
    subtitleOverride,
    titleOverride,
    toTheRight,
}: ObjectListItemProps<T>) {
    const session = useContext(SessionContext);
    const { palette, typography } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const id = useMemo(() => data?.id ?? uuid(), [data]);

    const [object, setObject] = useState<T | null | undefined>(data);
    useEffect(() => { setObject(data); }, [data]);

    const profileColors = useMemo(() => placeholderColor(), []);
    const { canBookmark, canComment, canUpdate, canReact, isBookmarked, reaction } = useMemo(() => getYou(data), [data]);
    const { subtitle, title, adornments } = useMemo(() => getDisplay(data, getUserLanguages(session), palette), [data, palette, session]);
    const { score } = useMemo(() => getCounts(data), [data]);

    const link = useMemo(() => (
        data &&
        (typeof canNavigate !== "function" || canNavigate(data))) &&
        typeof onClick !== "function" &&
        !isSelecting ?
        getObjectUrl(data) :
        "", [data, canNavigate, isSelecting, onClick]);
    const handleClick = useCallback((target: EventTarget) => {
        if (!target.id || !target.id.startsWith(LIST_PREFIX)) return;
        // If data not supplied, don't open
        if (data === null) return;
        // If in selection mode, toggle selection
        if (isSelecting && typeof handleToggleSelect === "function") {
            handleToggleSelect(data);
            return;
        }
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
    }, [data, isSelecting, handleToggleSelect, onClick, canNavigate, setLocation, link]);

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
        // Show icons for organizations, users, and objects with display organizations/users
        if (isOfType(object, "Organization", "User", "Member", "MemberInvite", "ChatParticipant", "ChatInvite")) {
            type OrgOrUser = { __typename: "Organization" | "User", profileImage: string, updated_at: string, isBot?: boolean };
            const orgOrUser: OrgOrUser = (isOfType(object, "Member", "MemberInvite", "ChatParticipant", "ChatInvite") ? (object as unknown as (Member | MemberInvite | ChatParticipant | ChatInvite)).user : object) as unknown as OrgOrUser;
            const isBot = orgOrUser.isBot;
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
                    src={extractImageUrl(orgOrUser.profileImage, orgOrUser.updated_at, 50)}
                    alt={`${getDisplay(object).title}'s profile picture`}
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
        // Show multiple icons for chats and meetings
        if (isOfType(object, "Chat", "Meeting")) {
            // Filter yourself out of participants
            const attendeesOrParticipants = ((object as unknown as Meeting).attendees ?? (object as unknown as Chat).participants)?.filter((p: Meeting["attendees"][0] | Chat["participants"][0]) => (p as Meeting["attendees"][0])?.id !== getCurrentUser(session)?.id && (p as Chat["participants"][0])?.user?.id !== getCurrentUser(session)?.id) ?? [];
            // If no participants, show nothing
            if (attendeesOrParticipants.length === 0) return null;
            // If only one participant, show their profile picture instead of a group
            if (attendeesOrParticipants.length === 1) {
                const firstUser = (attendeesOrParticipants as unknown as Chat["participants"])[0]?.user ?? (attendeesOrParticipants as unknown as Meeting["attendees"])[0];
                return (
                    <Avatar
                        src={extractImageUrl(firstUser?.profileImage, firstUser?.updated_at, 50)}
                        alt={`${getDisplay(firstUser).title}'s profile picture`}
                        sx={{
                            backgroundColor: profileColors[0],
                            width: isMobile ? "40px" : "50px",
                            height: isMobile ? "40px" : "50px",
                            pointerEvents: "none",
                            display: "flex",
                            // Bots show up as squares, to distinguish them from users
                            ...(firstUser?.isBot ? { borderRadius: "8px" } : {}),
                        }}
                    >
                        {firstUser?.isBot ? <BotIcon width="75%" height="75%" /> : <UserIcon width="75%" height="75%" />}
                    </Avatar>
                );
            }
            // Otherwise, show a group
            return <ProfileGroup users={attendeesOrParticipants.map((p: Meeting["attendees"][0] | Chat["participants"][0]) => (p as Chat["participants"][0])?.user ?? p as Meeting["attendees"][0])} />;
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

    const titleId = `${LIST_PREFIX}title-stack-${id}`;

    const showIncompleteChip = useMemo(() => data && data.__typename !== "Reminder" && (data as any).isComplete === false, [data]);
    const showInternalChip = useMemo(() => data && (data as any).isInternal === true, [data]);
    const showTags = useMemo(() => Array.isArray((data as any)?.tags) && (data as any)?.tags.length > 0, [data]);

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
                    padding: isMobile ? "8px" : "8px 16px",
                    cursor: "pointer",
                    borderBottom: `1px solid ${palette.divider}`,
                    background: isSelected ? palette.secondary.light : palette.background.paper,
                    "&:hover": {
                        background: isSelected ? palette.secondary.light : palette.action.hover,
                    },
                }}
            >
                {/* Giant radio button if isSelecting */}
                {isSelecting && <Box
                    sx={{
                        width: "24px",
                        height: "24px",
                        borderRadius: "50%",
                        backgroundColor: isSelected ? palette.secondary.main : palette.background.paper,
                        border: `1px solid ${palette.divider}`,
                        pointerEvents: "none",
                        marginRight: "8px",
                    }}
                />}
                {leftColumn}
                <Stack
                    direction="column"
                    spacing={1}
                    pl={(isSelecting || leftColumn) ? 2 : 0}
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
                                {adornments.map(({ Adornment, key }) => (
                                    <Box key={key} sx={{
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
                    {(showIncompleteChip || showInternalChip || showTags || belowTags) && <Stack direction="row" spacing={1} sx={{ pointerEvents: "none" }}>
                        {showIncompleteChip && <Tooltip placement="top" title={t("MarkedIncomplete")}>
                            <Chip
                                label="Incomplete"
                                size="small"
                                sx={{
                                    backgroundColor: palette.error.main,
                                    color: palette.error.contrastText,
                                    width: "fit-content",
                                }} />
                        </Tooltip>}
                        {showInternalChip && <Tooltip placement="top" title={t("MarkedInternal")}>
                            <Chip
                                label="Internal"
                                size="small"
                                sx={{
                                    backgroundColor: palette.warning.main,
                                    color: palette.error.contrastText,
                                    width: "fit-content",
                                }} />
                        </Tooltip>}
                        {showTags &&
                            <TagList
                                parentId={data?.id ?? ""}
                                tags={(data as any).tags}
                            />}
                        {belowTags}
                    </Stack>}
                    {/* Action buttons if mobile */}
                    {isMobile && !isSelecting && actionButtons}
                </Stack>
                {!isMobile && !isSelecting && actionButtons}
                {/* Custom components displayed on the right */}
                {toTheRight}
            </ListItem>
        </>
    );
}
