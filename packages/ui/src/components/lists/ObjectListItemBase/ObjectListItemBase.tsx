import { Chat, ChatInvite, ChatParticipant, ListObject, Meeting, Member, MemberInvite, ReactionFor, getObjectUrl, isOfType, uuid } from "@local/shared";
import { AvatarGroup, Box, BoxProps, Chip, ChipProps, ListItemProps, ListItemText, Palette, Stack, Tooltip, styled, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../../contexts/session.js";
import { UsePressEvent, usePress } from "../../../hooks/gestures.js";
import { Icon, IconCommon, IconInfo } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { ObjectListProfileAvatar, multiLineEllipsis, noSelect } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { getBookmarkFor, getCounts, getDisplay, getYou, placeholderColor } from "../../../utils/display/listTools.js";
import { fontSizeToPixels } from "../../../utils/display/stringTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { setCookiePartialData } from "../../../utils/localStorage.js";
import { getObjectEditUrl } from "../../../utils/navigation/openObject.js";
import { CompletionBar } from "../../CompletionBar/CompletionBar.js";
import { BookmarkButton } from "../../buttons/BookmarkButton.js";
import { CommentsButton } from "../../buttons/CommentsButton.js";
import { ReportsButton } from "../../buttons/ReportsButton.js";
import { VoteButton } from "../../buttons/VoteButton.js";
import { MarkdownDisplay } from "../../text/MarkdownDisplay.js";
import { CompletionBarProps } from "../../types.js";
import { TagList } from "../TagList/TagList.js";
import { TextLoading } from "../TextLoading/TextLoading.js";
import { ObjectListItemProps } from "../types.js";

const LIST_PREFIX = "list-item-";
const EDIT_PREFIX = "edit-list-item-";
const TARGET_IMAGE_SIZE = 100;
const MAX_AVATARS_IN_GROUP = 4;

export type ListItemStyleColor = "Default" | "Red" | "Yellow" | "Green" | "Blue" | "Purple";
type ListItemStyleColorPair = { background: string, color: string };
type ListItemStyleColorThemed = { dark: ListItemStyleColorPair, light: ListItemStyleColorPair };

const LIST_ITEM_STYLE_COLORS: Record<ListItemStyleColor, (palette: Palette) => (ListItemStyleColorPair | ListItemStyleColorThemed)> = {
    Default: (palette) => ({ background: palette.secondary.main, color: palette.secondary.contrastText }),
    Red: (palette) => ({ background: palette.error.main, color: palette.error.contrastText }),
    Yellow: (palette) => ({ background: palette.warning.main, color: palette.warning.contrastText }),
    Green: (palette) => ({ background: palette.success.main, color: palette.success.contrastText }),
    Blue: () => ({
        dark: { background: "#016d97", color: "#ffffff" },
        light: { background: "#1d7691", color: "#ffffff" },
    }),
    Purple: () => ({ background: "#8148b0", color: "#ffffff" }),
};

function getStyleColor(color: ListItemStyleColor, palette: Palette): { background: string | undefined, color: string | undefined } {
    const colorFunc = LIST_ITEM_STYLE_COLORS[color];
    if (!colorFunc) {
        console.error(`Invalid color "${color}" for component`);
        return { background: undefined, color: undefined };
    }
    const colorData = colorFunc(palette);
    if (Object.prototype.hasOwnProperty.call(colorData, "dark")) {
        const themed = colorData as ListItemStyleColorThemed;
        return {
            background: themed.dark.background,
            color: themed.dark.color,
        };
    }
    const unthemed = colorData as ListItemStyleColorPair;
    return {
        background: unthemed.background,
        color: unthemed.color,
    };
}

type ListItemChipProps = Omit<ChipProps, "color"> & {
    color: ListItemStyleColor;
}

/**
 * Chip component used in several ObjectListItem components.
 */
export function ListItemChip({
    color,
    size,
    variant,
    ...props
}: ListItemChipProps) {
    const { palette } = useTheme();

    const style = useMemo(function styleMemo() {
        const baseStyle = { width: "fit-content" } as const;
        const { background, color: textColor } = getStyleColor(color, palette);
        return {
            ...baseStyle,
            backgroundColor: background,
            color: textColor,
        };
    }, [color, palette]);

    return (
        <Chip
            {...props}
            size={size || "small"}
            sx={style}
            variant={variant || "outlined"}
        />
    );
}

type ListItemCompletionBarProps = Omit<CompletionBarProps, "color"> & {
    color: ListItemStyleColor;
}

/**
 * Bar component used in several ObjectListItem components.
 */
export function ListItemCompletionBar({
    color,
    ...props
}: ListItemCompletionBarProps) {
    const { palette } = useTheme();

    const style = useMemo(function styleMemo() {
        const { background } = getStyleColor(color, palette);
        return {
            bar: {
                backgroundColor: background,
            },
        } as const;
    }, [color, palette]);

    return (
        <CompletionBar
            {...props}
            sxs={style}
        />
    );
}

interface StyledListItemProps extends ListItemProps {
    isMobile: boolean;
    isSelected: boolean;
}

const StyledListItem = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isMobile" && prop !== "isSelected",
})<StyledListItemProps>(({ isMobile, isSelected, theme }) => ({
    display: "flex",
    padding: isMobile ? "4px 8px" : "8px 16px",
    cursor: "pointer",
    borderBottom: `1px solid ${theme.palette.divider}`,
    background: isSelected ? theme.palette.secondary.light : theme.palette.background.paper,
    "&:hover": {
        background: isSelected ? theme.palette.secondary.light : theme.palette.action.hover,
    },
    textDecoration: "none",
    "& *": {
        textDecoration: "none",
    },
    ...noSelect,
}));

const TitleBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(0.5),
    lineBreak: "auto",
    wordBreak: "break-word",
    pointerEvents: "none",
}));

interface ActionButtonsRowProps extends BoxProps {
    isMobile: boolean;
}
const ActionButtonsRow = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<ActionButtonsRowProps>(({ isMobile, theme }) => ({
    display: "flex",
    flexDirection: isMobile ? "row" : "column",
    gap: theme.spacing(1),
    justifyContent: isMobile ? "right" : "center",
    pointerEvents: "none",
    alignItems: "center",
}));

interface EditIconBoxProps extends BoxProps {
    isMobile: boolean;
}
const EditIconBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<EditIconBoxProps>(({ isMobile, theme }) => ({
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    cursor: "pointer",
    pointerEvents: "all",
    paddingBottom: isMobile ? "0px" : "4px",
}));

interface GiantSelectorProps extends BoxProps {
    isSelected: boolean;
}
const GiantSelector = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isSelected",
})<GiantSelectorProps>(({ isSelected, theme }) => ({
    // eslint-disable-next-line no-magic-numbers
    width: theme.spacing(3),
    // eslint-disable-next-line no-magic-numbers
    height: theme.spacing(3),
    borderRadius: "50%",
    backgroundColor: isSelected ? theme.palette.secondary.dark : theme.palette.background.paper,
    border: `1px solid ${theme.palette.divider}`,
    pointerEvents: "none",
    marginRight: theme.spacing(1),
    marginTop: "auto",
    marginBottom: "auto",
}));

const TitleText = styled(ListItemText)(({ theme }) => ({
    color: theme.palette.background.textPrimary,
    ...noSelect,
}));

const SubtitleText = styled(MarkdownDisplay)(({ theme }) => ({
    color: theme.palette.text.secondary,
    pointerEvents: "none",
    ...multiLineEllipsis(2),
    ...noSelect,
}));

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
    const handleClick = useCallback((event: UsePressEvent) => {
        if (!event.target.id || !event.target.id.startsWith(LIST_PREFIX)) return;
        // If data not supplied, don't open
        if (data === null) return;
        // If in selection mode, toggle selection
        if (isSelecting && typeof handleToggleSelect === "function") {
            handleToggleSelect(data, event);
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
        onLongPress: ({ target }) => { handleContextMenu(target, data); },
        onClick: handleClick,
        onRightClick: ({ target }) => { handleContextMenu(target, data); },
    });

    /**
     * Left column is only shown on wide screens (if not a profile picture). It's either 
     * a vote button, an object icon, or nothing.
     */
    const leftColumn = useMemo(() => {
        // Show icons for teams, users, and objects with display teams/users
        if (isOfType(object, "Team", "User", "Member", "MemberInvite", "ChatParticipant", "ChatInvite")) {
            type OrgOrUser = { __typename: "Team" | "User", profileImage: string, updatedAt: string, isBot?: boolean };
            const orgOrUser: OrgOrUser = (isOfType(object, "Member", "MemberInvite", "ChatParticipant", "ChatInvite") ? (object as unknown as (Member | MemberInvite | ChatParticipant | ChatInvite)).user : object) as unknown as OrgOrUser;
            const isBot = orgOrUser.isBot;
            let iconInfo: IconInfo;
            if (object.__typename === "Team") {
                iconInfo = { name: "Team", type: "Common" };
            } else if (isBot) {
                iconInfo = { name: "Bot", type: "Common" };
            } else {
                iconInfo = { name: "User", type: "Common" };
            }
            return (
                <ObjectListProfileAvatar
                    alt={`${getDisplay(object).title}'s profile picture`}
                    isBot={isBot ?? false}
                    isMobile={isMobile}
                    profileColors={profileColors}
                    src={extractImageUrl(orgOrUser.profileImage, orgOrUser.updatedAt, TARGET_IMAGE_SIZE)}
                >
                    <Icon
                        fill={profileColors[1]}
                        info={iconInfo}
                    />
                </ObjectListProfileAvatar>
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
                    <ObjectListProfileAvatar
                        alt={`${getDisplay(firstUser).title}'s profile picture`}
                        isBot={firstUser?.isBot ?? false}
                        isMobile={isMobile}
                        profileColors={profileColors}
                        src={extractImageUrl(firstUser?.profileImage, firstUser?.updatedAt, TARGET_IMAGE_SIZE)}
                    >
                        <IconCommon
                            decorative
                            name={firstUser?.isBot ? "Bot" : "User"}
                        />
                    </ObjectListProfileAvatar>
                );
            }
            // Otherwise, show a group
            return (
                <AvatarGroup max={4} total={attendeesOrParticipants.length}>
                    {attendeesOrParticipants.slice(0, MAX_AVATARS_IN_GROUP).map((p: Meeting["attendees"][0] | Chat["participants"][0], index: number) => {
                        const user = (p as Chat["participants"][0])?.user ?? p as Meeting["attendees"][0];
                        return (
                            <ObjectListProfileAvatar
                                key={user.id || index}
                                alt={`${getDisplay(user).title}'s profile picture`}
                                isBot={user?.isBot ?? false}
                                isMobile={isMobile}
                                profileColors={placeholderColor(user.id)}
                                src={extractImageUrl(user?.profileImage, user?.updatedAt, TARGET_IMAGE_SIZE)}
                            >
                                <IconCommon
                                    decorative
                                    name={user?.isBot ? "Bot" : "User"}
                                />
                            </ObjectListProfileAvatar>
                        );
                    })}
                </AvatarGroup>
            );
        }
        // Other custom object icons
        if (isOfType(object, "BookmarkList")) {
            return <IconCommon
                decorative
                fill="#cbae30"
                name="BookmarkFilled"
                size={isMobile ? 40 : 50}
            />;
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

        const willShowEditButton = !hideUpdateButton && canUpdate;
        const willShowVoteButton = isMobile && canReact && object; // Displayed elsewhere on wide screens
        const willShowBookmarkButton = canBookmark && bookmarkFor && starForId;
        const willShowCommentButton = canComment;
        const willShowReportsButton = !isOfType(object, "RunRoutine", "RunProject") && reportsCount > 0;

        if (!willShowEditButton && !willShowVoteButton && !willShowBookmarkButton && !willShowCommentButton && !willShowReportsButton) return null;

        return (
            <ActionButtonsRow isMobile={isMobile}>
                {willShowEditButton &&
                    <EditIconBox
                        aria-label={t("Edit")}
                        component="a"
                        id={`${EDIT_PREFIX}button-${id}`}
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        href={editUrl}
                        isMobile={isMobile}
                        onClick={handleEditClick}
                    >
                        <IconCommon
                            decorative
                            fill={palette.secondary.main}
                            id={`${EDIT_PREFIX}icon-${id}`}
                            name="Edit"
                        />
                    </EditIconBox>}
                {willShowVoteButton && (
                    <VoteButton
                        disabled={!canReact}
                        emoji={reaction}
                        objectId={object?.id ?? ""}
                        voteFor={object.__typename as ReactionFor}
                        score={score}
                        onChange={(newEmoji: string | null, newScore: number) => { }}
                    />
                )}
                {willShowBookmarkButton && <BookmarkButton
                    disabled={!canBookmark}
                    objectId={starForId}
                    bookmarkFor={bookmarkFor}
                    isBookmarked={isBookmarked}
                    bookmarks={getCounts(object).bookmarks}
                />}
                {willShowCommentButton && <CommentsButton
                    commentsCount={getCounts(object).comments}
                    disabled={!canComment}
                    object={object}
                />}
                {willShowReportsButton && <ReportsButton
                    reportsCount={reportsCount}
                    object={object}
                />}
            </ActionButtonsRow>
        );
    }, [object, isMobile, hideUpdateButton, canUpdate, id, t, editUrl, handleEditClick, palette.secondary.main, canReact, reaction, score, canBookmark, isBookmarked, canComment]);

    const titleId = `${LIST_PREFIX}title-stack-${id}`;

    const showIncompleteChip = useMemo(() => data && data.__typename !== "Reminder" && (data as any).isComplete === false, [data]);
    const showInternalChip = useMemo(() => data && (data as any).isInternal === true, [data]);
    const showTags = useMemo(() => Array.isArray((data as any)?.tags) && (data as any)?.tags.length > 0, [data]);

    return (
        <>
            {/* List item */}
            <StyledListItem
                id={`${LIST_PREFIX}${id}`}
                disablePadding
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                button
                component={link.length > 0 ? "a" : "div"}
                href={link.length > 0 ? link : undefined}
                isMobile={isMobile}
                isSelected={isSelected}
                {...pressEvents}
            >
                {isSelecting && <GiantSelector
                    isSelected={isSelected}
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
                    {loading
                        ? <TextLoading />
                        : (titleOverride ?? title)
                            ? (
                                <TitleBox id={titleId}>
                                    <TitleText primary={titleOverride ?? title} />
                                    {adornments.map(({ Adornment, key }) => (
                                        <Box key={key} sx={{
                                            height: fontSizeToPixels(typography.body1.fontSize ?? "1rem", titleId) * Number(typography.body1.lineHeight ?? "1.5"),
                                        }}>
                                            {Adornment}
                                        </Box>
                                    ))}
                                </TitleBox>
                            )
                            : null
                    }
                    {/* Subtitle */}
                    {
                        loading
                            ? <TextLoading />
                            : (subtitleOverride ?? subtitle)
                                ? <SubtitleText
                                    content={subtitleOverride ?? subtitle}
                                />
                                : null
                    }
                    {/* Any custom components to display below the subtitle */}
                    {belowSubtitle}
                    {(showIncompleteChip || showInternalChip || showTags || belowTags) && <Stack direction="row" spacing={1} sx={{ pointerEvents: "none" }}>
                        {showIncompleteChip && <Tooltip placement="top" title={t("MarkedIncomplete")}>
                            <ListItemChip
                                color="Red"
                                label="Incomplete"
                            />
                        </Tooltip>}
                        {showInternalChip && <Tooltip placement="top" title={t("MarkedInternal")}>
                            <ListItemChip
                                color="Yellow"
                                label="Internal"
                            />
                        </Tooltip>}
                        {showTags &&
                            <TagList
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
            </StyledListItem>
        </>
    );
}
