import { AITaskInfo, ChatMessageShape, ChatParticipantShape, ListObject, LlmTask, getTranslation } from "@local/shared";
import { Box, Button, Chip, ChipProps, CircularProgress, Divider, IconButton, ListItemText, Menu, MenuItem, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { RichInputBase } from "components/inputs/RichInput/RichInput";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { SessionContext } from "contexts";
import useDraggableScroll from "hooks/gestures";
import { useShowBotWarning } from "hooks/subscriptions";
import { UseChatTaskReturn } from "hooks/tasks";
import { useKeyboardOpen } from "hooks/useKeyboardOpen";
import { useWindowSize } from "hooks/useWindowSize";
import { AddIcon, CancelIcon, DeleteIcon, EditIcon, ErrorIcon, PlayIcon, SearchIcon, SendIcon, SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "styles";
import { ViewDisplayType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { isTaskStale } from "utils/display/chatTools";
import { getDisplay } from "utils/display/listTools";
import { displayDate } from "utils/display/stringTools";
import { PubSub } from "utils/pubsub";

const DEFAULT_TYPING_INDICATOR_MAX_CHARS = 40;
const NUM_TYPING_INDICATOR_ELLIPSIS_DOTS = 3;
const TYPING_INDICATOR_DOTS_INTERVAL_MS = 500;

type ChatIndicatorProps = {
    participantsAll: Omit<ChatParticipantShape, "chat">[] | undefined,
    participantsTyping: Omit<ChatParticipantShape, "chat">[],
    messagesCount: number,
}

type MenuPosition = {
    mouseX: number;
    mouseY: number
}

/**
 * Determines the text displayed in the typing indicator
 * @param participants The participants who are typing
 * @param maxChars The maximum number of characters to display
 * @returns The text to display in the typing indicator
 */
export function getTypingIndicatorText(participants: ListObject[], maxChars: number) {
    if (participants.length === 0) return "";

    const ellipsis = "…";
    const remainingCountPrefix = ", +";
    const append = participants.length === 1 ? " is typing" : " are typing";
    let text = "";
    let remainingCount = participants.length;

    for (let i = 0; i < participants.length; i++) {
        const name = getDisplay(participants[i]).title;
        const separator = i === 0 ? "" : ", ";
        remainingCount--; // Decrease remaining count as we're adding this participant
        // Total potential length of the text if we add this name, the separator, the remaining count (if applicable), and the append
        const additionalLength = text.length + name.length + separator.length + append.length + (remainingCount > 0 ? (`${remainingCountPrefix}${remainingCount}`.length) : 0);

        // Check if adding this name would exceed maxChars
        if (additionalLength <= maxChars) {
            text += `${separator}${name}`;
        } else {
            // What the test would look like with only one character of this participant
            // const minimumName = `${text}${separator}Z${ellipsis}${remainingCountPrefix}${remainingCount}${append}`;
            const availableLength = maxChars - additionalLength + name.length - 1;
            if (availableLength <= 0) {
                remainingCount++; // Increase remaining count as we can't fit a single character of this participant
                break;
            }
            text += `${separator}${name.substring(0, availableLength)}${ellipsis}`;
            break;
        }
    }

    if (remainingCount > 0) {
        text += remainingCount === 1
            ? "" :
            remainingCount === participants.length
                ? `${remainingCountPrefix.replace(", ", "")}${remainingCount}`
                : `${remainingCountPrefix}${remainingCount}`;
    }

    return `${text}${append}`;
}

function showBotWarningDetails() {
    PubSub.get().publish("alertDialog", {
        messageKey: "BotChatWarningDetails",
        buttons: [
            { labelKey: "Ok" },
        ],
    });
}

const ChatIndicatorBox = styled(Box)(() => ({
    display: "flex",
    flexDirection: "row",
    justifyContent: "flex-start",
    alignItems: "center",
    gap: 0,
    width: "min(100%, 700px)",
    margin: "auto",
}));

// interface HideBotWarningMenuProps extends MenuProps {
//     menuPosition: MenuPosition | null;
// }
// const HideBotWarningMenu = styled(Box, {
//     shouldForwardProp: (prop) => prop !== "menuPosition",
// })<HideBotWarningMenuProps>(({ menuPosition, theme }) => ({
//    anch
// }));

const showBotWarningButtonStyle = { textTransform: "none" } as const;
const contextItemStyle = { display: "flex", alignItems: "center" } as const;

/**
 * Displays text information about the chat. Can be a warning that you 
 * are chatting with a bot, a typing indicator, or blank.
 */
export function ChatIndicator({
    participantsAll,
    participantsTyping,
    messagesCount,
}: ChatIndicatorProps) {
    const { t } = useTranslation();

    const botWarningPreferences = useShowBotWarning();
    const [contextMenu, setContextMenu] = useState<MenuPosition | null>(null);
    const onHideBotWarning = useCallback(function onHideBotWarningCallback() {
        botWarningPreferences.handleUpdateShowBotWarning(false);
        setContextMenu(null);
    }, [botWarningPreferences]);
    const handleContextMenu = useCallback(function handleContextMenuCallback(event: React.MouseEvent) {
        if (botWarningPreferences.showBotWarning === false) return;
        event.preventDefault();
        setContextMenu(
            contextMenu === null
                ? { mouseX: event.clientX - 2, mouseY: event.clientY - 4 }
                : null,
        );
    }, [botWarningPreferences.showBotWarning, contextMenu]);
    const handleContextMenuClose = useCallback(function handleContextMenuClose() {
        setContextMenu(null);
    }, []);

    // Bot warning should only be displayed at the beginning of a chat, 
    // when there are no other participants typing, and there is a bot participant
    const showBotWarning = useMemo(function showBotWarningMemo() {
        if (botWarningPreferences.showBotWarning === false) return false;
        const hasBotParticipant = participantsAll?.some(p => p.user?.isBot) ?? false;
        return participantsTyping.length === 0
            && messagesCount <= 2
            && hasBotParticipant;
    }, [botWarningPreferences.showBotWarning, messagesCount, participantsAll, participantsTyping.length]);

    const typingIndicatorText = useMemo(function typingIndicatorTextMemo() {
        return getTypingIndicatorText(participantsTyping, DEFAULT_TYPING_INDICATOR_MAX_CHARS);
    }, [participantsTyping]);

    const [dots, setDots] = useState("");
    useEffect(function typingIndicatorDotsAnimationEffect() {
        let interval;
        if (typingIndicatorText !== "") {
            interval = setInterval(() => {
                setDots(prevDots => {
                    if (prevDots.length < NUM_TYPING_INDICATOR_ELLIPSIS_DOTS) return prevDots + ".";
                    return "";
                });
            }, TYPING_INDICATOR_DOTS_INTERVAL_MS);
        } else {
            setDots("");
        }

        return () => clearInterval(interval);
    }, [typingIndicatorText]);

    const contextAnchorPosition = useMemo(function contextAnchorPositionMemo() {
        if (contextMenu === null) return undefined;
        return { top: contextMenu.mouseY, left: contextMenu.mouseX };
    }, [contextMenu]);

    if (!showBotWarning && typingIndicatorText === "") return null;
    return (
        <ChatIndicatorBox onContextMenu={handleContextMenu}>
            {/* Right-click context menu */}
            <Menu
                open={contextMenu !== null}
                onClose={handleContextMenuClose}
                anchorReference="anchorPosition"
                anchorPosition={contextAnchorPosition}
            >
                <MenuItem onClick={onHideBotWarning}
                    sx={contextItemStyle}>
                    <ListItemText primary="Hide Warning" secondary="Hide bot warning. You can reenable this in the settings by clearing display cache." />
                </MenuItem>
            </Menu>
            {showBotWarning && <>
                <Typography variant="body2" p={1}>{t("BotChatWarning")}</Typography>
                <Button variant="text" sx={showBotWarningButtonStyle} onClick={showBotWarningDetails}>{t("LearnMore")}</Button>
            </>}
            {!showBotWarning && <Typography variant="body2" p={1}>{typingIndicatorText}{dots}</Typography>}
        </ChatIndicatorBox>
    );
}

const INPUT_ROWS_FOCUSED = 10;
const INPUT_ROWS_UNFOCUSED = 2;

interface StyledChipProps extends ChipProps {
    canPress: boolean;
    isActive: boolean;
}

const StyledChip = styled(Chip, {
    shouldForwardProp: (prop) => prop !== "canPress" && prop !== "isActive",
})<StyledChipProps>(({ canPress, isActive }) => ({
    transition: "all 0.3s ease",
    cursor: canPress ? "pointer" : "default",
    borderRadius: "4px",
    outline: isActive ? "2px solid" : "none",
    paddingLeft: "4px",
    height: "32px",
}));

type TaskChipProps = {
    isActive: boolean,
    onTaskClick: (task: AITaskInfo) => unknown,
    taskInfo: AITaskInfo,
}

/** Displays a suggested, active, or finished task */
function TaskChip({
    isActive,
    onTaskClick,
    taskInfo,
}: TaskChipProps) {
    const { label, status, resultLabel, resultLink, task } = taskInfo;
    const { palette } = useTheme();

    const isStale = isTaskStale(taskInfo);
    const canPress =
        isStale // Has been running or canceling for too long
        || !["Completed", "Canceling"].includes(status)
        || Boolean(status === "Completed" && resultLink); // Result can be opened

    function getStatusColor() {
        if (isStale) return "warning";
        switch (status) {
            case "Running":
            case "Canceling":
                return "primary";
            case "Completed":
                return "success";
            case "Failed":
                return "error";
            default:
                return "default";
        }
    }

    function getActionIcon() {
        //TODO should be different from status icon, and should trigger respondToTask. Should also support multiple actions
        if (isStale || !task) return <ErrorIcon />;
        switch (status) {
            case "Running":
            case "Canceling":
                return <CircularProgress size={20} color="inherit" />;
            case "Completed":
                return <SuccessIcon />;
            case "Failed":
                return <ErrorIcon />;
            default:
                // Base Icon style on task type
                if (task.endsWith("Add"))
                    return <AddIcon />;
                if (task.endsWith("Delete"))
                    return <DeleteIcon />;
                if (task.endsWith("Find"))
                    return <SearchIcon />;
                if (task.endsWith("Update"))
                    return <EditIcon />;
                return <PlayIcon />;
        }
    }

    function getStatusIcon() {
        if (isStale || !task) return <ErrorIcon />;
        switch (status) {
            case "Running":
            case "Canceling":
                return <CircularProgress size={20} color="inherit" />;
            case "Completed":
                return <SuccessIcon />;
            case "Failed":
                return <ErrorIcon />;
            default:
                // Base Icon style on task type
                if (task.endsWith("Add"))
                    return <AddIcon />;
                if (task.endsWith("Delete"))
                    return <DeleteIcon />;
                if (task.endsWith("Find"))
                    return <SearchIcon />;
                if (task.endsWith("Update"))
                    return <EditIcon />;
                return <PlayIcon />;
        }
    }

    function getStatusTooltip() {
        if (isStale) return `Task is stale: ${label}`;
        switch (status) {
            case "Suggested":
                return `Press to start task: ${label}`;
            case "Running":
                return `Task is running: ${label} (Started: ${displayDate(taskInfo.lastUpdated)})`;
            case "Canceling":
                return `Task is canceling: ${label}`;
            case "Completed":
                return `Task completed: ${resultLabel || label}`;
            case "Failed":
                return `Task failed: ${label}`;
            default:
                return `Task: ${label}`;
        }
    }

    function handleTaskClick() {
        console.log("in handleTaskClick", canPress, taskInfo);
        if (!canPress) return;
        onTaskClick(taskInfo);
    }

    return (
        <Tooltip title={getStatusTooltip()}>
            <Box display="flex" alignItems="center">
                {isActive && <Box display="flex" pr={0.5} color={palette.secondary.main}>
                    {getActionIcon()}
                </Box>}
                <StyledChip
                    canPress={canPress}
                    disabled={!canPress}
                    isActive={isActive}
                    icon={!isActive ? getStatusIcon() : undefined}
                    label={resultLabel || label}
                    onClick={handleTaskClick}
                    color={getStatusColor()}
                />
            </Box>
        </Tooltip>
    );
}

const TasksRowOuter = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    padding: theme.spacing(0.5),
    gap: theme.spacing(1),
    height: "50px",
    width: "min(700px, 100%)",
    marginLeft: "auto",
    marginRight: "auto",
    overflowX: "auto",
    "&::-webkit-scrollbar": {
        display: "none",
    },
}));

type TasksRowProps = Pick<UseChatTaskReturn, "activeTask" | "inactiveTasks" | "unselectTask" | "selectTask">;

function TasksRow({
    activeTask,
    inactiveTasks,
    unselectTask,
    selectTask,
}: TasksRowProps) {
    console.log("in TasksRow", activeTask, inactiveTasks);
    const ref = useRef<HTMLDivElement>(null);
    const { onMouseDown } = useDraggableScroll({ ref, options: { direction: "horizontal" } });

    const onTaskClick = useCallback(function onTaskClickCallback(task: AITaskInfo) {
        if (activeTask?.taskId === task.taskId) {
            unselectTask();
        } else {
            selectTask(task);
        }
    }, [activeTask?.taskId, selectTask, unselectTask]);

    const hasNonStartTasks = (activeTask && activeTask.task !== LlmTask.Start) || inactiveTasks.some(task => task.task !== LlmTask.Start);
    if (!hasNonStartTasks) return null;
    return (
        <TasksRowOuter
            onMouseDown={onMouseDown}
            ref={ref}
        >
            {activeTask.task !== LlmTask.Start && (
                <>
                    <TaskChip
                        key={activeTask.taskId}
                        isActive={true}
                        taskInfo={activeTask}
                        onTaskClick={onTaskClick}
                    />
                    {inactiveTasks.length > 0 && <Divider orientation="vertical" flexItem />}
                </>
            )}
            {inactiveTasks.map((taskInfo) => {
                if (taskInfo.task === LlmTask.Start) return null;
                return (
                    <TaskChip
                        key={taskInfo.taskId}
                        isActive={false}
                        taskInfo={taskInfo}
                        onTaskClick={onTaskClick}
                    />
                );
            })}
        </TasksRowOuter>
    );
}

const markdownDisplayStyle = {
    whiteSpace: "pre-wrap",
    wordWrap: "break-word",
    overflowWrap: "anywhere",
    maxHeight: "50vh",
    overflowY: "auto",
} as const;
const replyToCloseIconStyle = { paddingLeft: 0 } as const;

type ReplyingToMessageDisplayProps = {
    messageBeingRepliedTo: ChatMessageShape;
    onCancelReply: () => unknown;
}

export function ReplyingToMessageDisplay({
    messageBeingRepliedTo,
    onCancelReply,
}: ReplyingToMessageDisplayProps) {
    const session = useContext(SessionContext);
    const { languages } = useMemo(() => getCurrentUser(session), [session]);

    return (
        <Box p={1}>
            <Box display="flex" flexDirection="row" justifyContent="flex-start" alignItems="center">
                <IconButton onClick={onCancelReply} sx={replyToCloseIconStyle}>
                    <CancelIcon />
                </IconButton>
                <Typography variant="body2" color="textSecondary">
                    Replying to: {getDisplay(messageBeingRepliedTo.user).title || "User"}
                </Typography>
            </Box>
            <MarkdownDisplay
                content={getTranslation(messageBeingRepliedTo, languages, true)?.text}
                sx={markdownDisplayStyle}
            />
        </Box>
    );
}

type EditingMessageDisplayProps = {
    onCancelEdit: () => unknown;
}

function EditingMessageDisplay({
    onCancelEdit,
}: EditingMessageDisplayProps) {
    return (
        <Box p={1}>
            <Box display="flex" flexDirection="row" justifyContent="flex-start" alignItems="center">
                <IconButton onClick={onCancelEdit} sx={replyToCloseIconStyle}>
                    <CancelIcon />
                </IconButton>
                <Typography variant="body2" color="textSecondary">
                    Editing message
                </Typography>
            </Box>
        </Box>
    );
}

type ChatMessageInputProps = Pick<ChatIndicatorProps, "participantsAll" | "participantsTyping" | "messagesCount"> & {
    disabled: boolean;
    display: ViewDisplayType;
    isLoading: boolean;
    message: string;
    messageBeingEdited: ChatMessageShape | null;
    messageBeingRepliedTo: ChatMessageShape | null;
    onFocused?: () => unknown;
    placeholder: string;
    stopEditingMessage: () => unknown;
    stopReplyingToMessage: () => unknown;
    setMessage: (message: string) => unknown;
    /**
     * Creates, edits, or replies to a message
     */
    submitMessage: (message: string) => unknown;
    taskInfo: UseChatTaskReturn;
}

export function ChatMessageInput({
    disabled,
    display,
    isLoading,
    message,
    messageBeingEdited,
    messageBeingRepliedTo,
    messagesCount,
    onFocused,
    participantsAll,
    participantsTyping,
    placeholder,
    setMessage,
    stopEditingMessage,
    stopReplyingToMessage,
    submitMessage,
    taskInfo,
}: ChatMessageInputProps) {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isKeyboardOpen = useKeyboardOpen();

    const [inputFocused, setInputFocused] = useState(false);
    const onFocus = useCallback(() => {
        setInputFocused(true);
        onFocused?.();
    }, [onFocused]);
    const onBlur = useCallback(() => { setInputFocused(false); }, []);

    // TODO update to connect more than just participants (though these results should be at the top).
    // Should also be able to tag any public or owned object (e.g. "Create routine like @some_existing_routine, but change a to b")
    const getTaggableItems = useCallback(async function getTaggableItemsCallback(searchString: string) {
        // Find all users in the chat, plus @Everyone
        let users = [
            //TODO handle @Everyone
            ...participantsAll?.map(p => p.user) ?? [],
        ];
        // Filter out current user
        users = users.filter(p => p.id !== getCurrentUser(session).id);
        // Filter out users that don't match the search string
        users = users.filter(p => p.name.toLowerCase().includes(searchString.toLowerCase()));
        console.log("got taggable items", users, searchString);
        return users;
    }, [participantsAll, session]);

    const inputActionButtons = useMemo(function inputActionButtonsMemo() {
        return [{
            Icon: SendIcon,
            disabled: disabled || isLoading,
            onClick: () => {
                submitMessage(message);
            },
        }];
    }, [disabled, isLoading, message, submitMessage]);

    const inputStyle = useMemo(function inputStyleMemo() {
        // If this is the full width of the page, we have to add 
        // additional padding to account for the side menu touch areas
        const usesFullViewportWidth = display === "page" && isMobile;
        // If this is place on the bottom of the page and the virtual keyboard 
        // is open, we need to remove the bottom margin that keeps this above the 
        // BottomNav, since the nav is hiddenwhen the keyboard is open
        const shouldRemoveBottomPadding = display !== "page" || (display === "page" && isKeyboardOpen);

        return {
            root: {
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                maxHeight: "min(75vh, 500px)",
                width: "min(700px, 100%)",
                marginTop: "auto",
                marginLeft: "auto",
                marginRight: "auto",
                paddingBottom: shouldRemoveBottomPadding ? "0" : pagePaddingBottom,
            },
            topBar: {
                borderRadius: 0,
                paddingLeft: usesFullViewportWidth ? "20px" : 0,
                paddingRight: usesFullViewportWidth ? "20px" : 0,
            },
            bottomBar: {
                paddingLeft: usesFullViewportWidth ? "20px" : 0,
                paddingRight: usesFullViewportWidth ? "20px" : 0,
            },
            inputRoot: {
                border: "none",
                background: palette.background.paper,
            },
        };
    }, [display, isKeyboardOpen, isMobile, palette.background.paper, palette.primary.contrastText, palette.primary.dark]);

    const handleSubmit = useCallback(function handleAddMessageCallback(message: string) {
        if (disabled || isLoading) return;
        submitMessage(message);
    }, [disabled, isLoading, submitMessage]);

    return (
        <>
            <ChatIndicator
                participantsAll={participantsAll}
                participantsTyping={participantsTyping}
                messagesCount={messagesCount}
            />
            {!messageBeingRepliedTo && <TasksRow
                activeTask={taskInfo.activeTask}
                inactiveTasks={taskInfo.inactiveTasks}
                unselectTask={taskInfo.unselectTask}
                selectTask={taskInfo.selectTask}
            />}
            {messageBeingRepliedTo && <ReplyingToMessageDisplay
                messageBeingRepliedTo={messageBeingRepliedTo}
                onCancelReply={stopReplyingToMessage}
            />}
            {messageBeingEdited && <EditingMessageDisplay onCancelEdit={stopEditingMessage} />}
            <RichInputBase
                actionButtons={inputActionButtons}
                disabled={disabled}
                disableAssistant={true}
                fullWidth
                getTaggableItems={getTaggableItems}
                maxChars={1500}
                maxRows={inputFocused ? INPUT_ROWS_FOCUSED : INPUT_ROWS_UNFOCUSED}
                minRows={1}
                onBlur={onBlur}
                onChange={setMessage}
                onFocus={onFocus}
                onSubmit={handleSubmit}
                name="newMessage"
                placeholder={placeholder}
                sxs={inputStyle}
                taskInfo={taskInfo}
                value={message}
            />
        </>
    );
}
