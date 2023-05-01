import { EditIcon, isOfType, OrganizationIcon, ReactionFor, RunProject, RunRoutine, RunStatus, SvgComponent, useLocation, UserIcon, uuid } from "@local/shared";
import { Box, Chip, LinearProgress, ListItem, ListItemText, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { CommentsButton } from "components/buttons/CommentsButton/CommentsButton";
import { ReportsButton } from "components/buttons/ReportsButton/ReportsButton";
import { VoteButton } from "components/buttons/VoteButton/VoteButton";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
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
import { RoleList } from "../RoleList/RoleList";
import { smallHorizontalScrollbar } from "../styles";
import { TagList } from "../TagList/TagList";
import { TextLoading } from "../TextLoading/TextLoading";
import { ObjectListItemProps } from "../types";

export function CompletionBar(props) {
    return (
        <Box sx={{ display: "flex", alignItems: "center", pointerEvents: "none" }}>
            <Box sx={{ width: "100%", mr: 1 }}>
                <LinearProgress variant={props.variant} {...props} sx={{ borderRadius: 1, height: 8 }} />
            </Box>
            <Box sx={{ minWidth: 35 }}>
                <Typography variant="body2" color="text.secondary">{`${Math.round(
                    props.value,
                )}%`}</Typography>
            </Box>
        </Box>
    );
}

export function ObjectListItem<T extends ListObjectType>({
    beforeNavigation,
    hideUpdateButton,
    index,
    loading,
    data,
    objectType,
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
    const { canComment, canUpdate, canReact, canBookmark, isBookmarked, reaction } = useMemo(() => getYou(data), [data]);
    const { subtitle, title } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);
    const { score } = useMemo(() => getCounts(data), [data]);

    // Context menu
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const handleContextMenu = useCallback((target: EventTarget) => {
        setAnchorEl(target as HTMLElement);
    }, []);
    const closeContextMenu = useCallback(() => setAnchorEl(null), []);

    const link = useMemo(() => data ? getObjectUrl(data) : "", [data]);
    const handleClick = useCallback((target: EventTarget) => {
        if (!target.id || !target.id.startsWith("list-item-")) return;
        // If data not supplied, don't open
        if (data === null || link.length === 0) return;
        // If beforeNavigation is supplied, call it
        if (beforeNavigation) {
            const shouldContinue = beforeNavigation(data);
            if (shouldContinue === false) return;
        }
        // Navigate to the object's page
        setLocation(link);
    }, [link, beforeNavigation, setLocation, data]);

    const editUrl = useMemo(() => data ? getObjectEditUrl(data) : "", [data]);
    const handleEditClick = useCallback((event: any) => {
        event.preventDefault();
        const target = event.target;
        if (!target.id || !target.id.startsWith("edit-list-item-")) return;
        // If data not supplied, don't open
        if (!data) return;
        // If beforeNavigation is supplied, call it
        if (beforeNavigation) {
            const shouldContinue = beforeNavigation(data);
            if (shouldContinue === false) return;
        }
        // Navigate to the object's edit page
        setLocation(editUrl);
    }, [beforeNavigation, data, editUrl, setLocation]);

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
        if (isMobile && !isOfType(object, "Organization", "User")) return null;
        // Show icons for organizations and users
        switch (object?.__typename) {
            case "Organization":
            case "User": {
                const Icon: SvgComponent = object?.__typename === "Organization" ? OrganizationIcon : UserIcon;
                return (
                    <Box
                        width={isMobile ? "40px" : "50px"}
                        minWidth={isMobile ? "40px" : "50px"}
                        height={isMobile ? "40px" : "50px"}
                        marginBottom={isMobile ? "auto" : "unset"}
                        borderRadius='100%'
                        bgcolor={profileColors[0]}
                        justifyContent='center'
                        alignItems='center'
                        sx={{
                            display: "flex",
                            pointerEvents: "none",
                        }}
                    >
                        <Icon
                            fill={profileColors[1]}
                            width={isMobile ? "25px" : "35px"}
                            height={isMobile ? "25px" : "35px"}
                        />
                    </Box>
                );
            }
            case "Project":
            case "Routine":
            case "Standard":
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
            default:
                return null;
        }
    }, [isMobile, object, profileColors, canReact, reaction, score]);

    /**
     * Action buttons are shown as a column on wide screens, and 
     * a row on mobile. It displays 
     * the star, comments, and reports buttons.
     */
    const actionButtons = useMemo(() => {
        const commentableObjects: string[] = ["Project", "Routine", "Standard"];
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
                {isMobile && isOfType(object,
                    "Api",
                    "ApiVersion",
                    "Comment",
                    "Issue",
                    "Note",
                    "NoteVersion",
                    "Post",
                    "Project",
                    "ProjectVersion",
                    "Question",
                    "QuestionAnswer",
                    "Quiz",
                    "Routine",
                    "RoutineVersion",
                    "SmartContract",
                    "SmartContractVersion",
                    "Standard",
                    "StandardVersion") && (
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
                {bookmarkFor && <BookmarkButton
                    disabled={!canBookmark}
                    objectId={starForId}
                    bookmarkFor={bookmarkFor}
                    isBookmarked={isBookmarked}
                    bookmarks={getCounts(object).bookmarks}
                    zIndex={zIndex}
                />}
                {commentableObjects.includes(object?.__typename ?? "") && (<CommentsButton
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

    /**
     * Run list items may get a progress bar
     */
    const progressBar = useMemo(() => {
        if (!isOfType(object, "RunProject", "RunRoutine")) return null;
        const completedComplexity = (object as any as RunProject | RunRoutine).completedComplexity;
        const totalComplexity = (object as any as RunProject).projectVersion?.complexity ?? (object as any as RunRoutine).routineVersion?.complexity ?? null;
        const percentComplete = (object as any as RunProject | RunRoutine).status === RunStatus.Completed ? 100 :
            (completedComplexity && totalComplexity) ?
                Math.min(Math.round(completedComplexity / totalComplexity * 100), 100) :
                0;
        return (<CompletionBar
            color="secondary"
            variant={loading ? "indeterminate" : "determinate"}
            value={percentComplete}
            sx={{ height: "15px" }}
        />);
    }, [loading, object]);

    const actionData = useObjectActions({
        beforeNavigation,
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
                component="a"
                href={link}
                {...pressEvents}
                onClick={(e) => { e.preventDefault(); }}
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
                    {loading ? <TextLoading /> : <ListItemText
                        primary={subtitle}
                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: "none" }}
                    />}
                    {/* Progress bar */}
                    {progressBar}
                    <Stack direction="row" spacing={1} sx={{ pointerEvents: "none" }}>
                        {/* Incomplete chip */}
                        {
                            data && (data as any).isComplete === false && <Tooltip placement="top" title={t("MarkedIncomplete")}>
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
                        {/* Roles (Member objects only) */}
                        {isOfType(object, "Member") && (data as any)?.roles?.length > 0 &&
                            <RoleList
                                roles={(data as any).roles}
                                sx={{ ...smallHorizontalScrollbar(palette) }}
                            />}
                    </Stack>
                    {/* Action buttons if mobile */}
                    {isMobile && actionButtons}
                </Stack>
                {!isMobile && actionButtons}
            </ListItem>
        </>
    );
}
