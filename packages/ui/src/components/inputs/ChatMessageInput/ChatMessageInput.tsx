import { AITaskInfo, ChatMessageShape, ChatParticipantShape, ListObject, LlmTask, getTranslation } from "@local/shared";
import { Box, Chip, ChipProps, CircularProgress, Divider, IconButton, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../../contexts/session.js";
import useDraggableScroll from "../../../hooks/gestures.js";
import { UseChatTaskReturn } from "../../../hooks/tasks.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { IconCommon } from "../../../icons/Icons.js";
import { ViewDisplayType } from "../../../types.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { ELEMENT_IDS, MAX_CHAT_INPUT_WIDTH } from "../../../utils/consts.js";
import { isTaskStale } from "../../../utils/display/chatTools.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { displayDate } from "../../../utils/display/stringTools.js";
import { MarkdownDisplay } from "../../text/MarkdownDisplay.js";
import { AdvancedInputBase } from "../AdvancedInput/AdvancedInput.js";

const DEFAULT_TYPING_INDICATOR_MAX_CHARS = 40;
const NUM_TYPING_INDICATOR_ELLIPSIS_DOTS = 3;
const TYPING_INDICATOR_DOTS_INTERVAL_MS = 500;

type ChatIndicatorProps = {
    participantsTyping: Omit<ChatParticipantShape, "chat">[],
}

/**
 * Determines the text displayed in the typing indicator
 * @param participants The participants who are typing
 * @param maxChars The maximum number of characters to display
 * @returns The text to display in the typing indicator
 */
export function getTypingIndicatorText(participants: ListObject[], maxChars: number) {
    if (!Array.isArray(participants) || participants.length === 0) return "";

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

const ChatIndicatorBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    justifyContent: "flex-start",
    alignItems: "center",
    gap: 0,
    width: `min(100%, ${MAX_CHAT_INPUT_WIDTH}px)`,
    margin: "auto",
    color: theme.palette.background.textSecondary,
}));

/**
 * Displays text information about the chat. Can be a warning that you 
 * are chatting with a bot, a typing indicator, or blank.
 */
export function ChatIndicator({
    participantsTyping,
}: ChatIndicatorProps) {
    const [dots, setDots] = useState("");

    const typingIndicatorText = useMemo(function typingIndicatorTextMemo() {
        return getTypingIndicatorText(participantsTyping, DEFAULT_TYPING_INDICATOR_MAX_CHARS);
    }, [participantsTyping]);

    useEffect(function typingIndicatorDotsAnimationEffect() {
        let interval: NodeJS.Timeout;
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

    if (typingIndicatorText === "") return null;
    return (
        <ChatIndicatorBox>
            <Typography variant="body2" p={1}>{typingIndicatorText}{dots}</Typography>
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
        if (isStale || !task) return <IconCommon
            decorative
            name="Error"
        />;
        switch (status) {
            case "Running":
            case "Canceling":
                return <CircularProgress size={20} color="inherit" />;
            case "Completed":
                return <IconCommon
                    decorative
                    name="Success"
                />;
            case "Failed":
                return <IconCommon
                    decorative
                    name="Error"
                />;
            default:
                // Base Icon style on task type
                if (task.endsWith("Add"))
                    return <IconCommon
                        decorative
                        name="Add"
                    />;
                if (task.endsWith("Delete"))
                    return <IconCommon
                        decorative
                        name="Delete"
                    />;
                if (task.endsWith("Find"))
                    return <IconCommon
                        decorative
                        name="Search"
                    />;
                if (task.endsWith("Update"))
                    return <IconCommon
                        decorative
                        name="Edit"
                    />;
                return <IconCommon decorative name="Play" />;
        }
    }

    function getStatusIcon() {
        if (isStale || !task) return <IconCommon
            decorative
            name="Error"
        />;
        switch (status) {
            case "Running":
            case "Canceling":
                return <CircularProgress size={20} color="inherit" />;
            case "Completed":
                return <IconCommon
                    decorative
                    name="Success"
                />;
            case "Failed":
                return <IconCommon
                    decorative
                    name="Error"
                />;
            default:
                // Base Icon style on task type
                if (task.endsWith("Add"))
                    return <IconCommon
                        decorative
                        name="Add"
                    />;
                if (task.endsWith("Delete"))
                    return <IconCommon
                        decorative
                        name="Delete"
                    />;
                if (task.endsWith("Find"))
                    return <IconCommon
                        decorative
                        name="Search"
                    />;
                if (task.endsWith("Update"))
                    return <IconCommon
                        decorative
                        name="Edit"
                    />;
                return <IconCommon decorative name="Play" />;
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
    width: `min(100%, ${MAX_CHAT_INPUT_WIDTH}px)`,
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
            id={ELEMENT_IDS.TasksRow}
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
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const { languages } = useMemo(() => getCurrentUser(session), [session]);

    return (
        <Box p={1}>
            <Box display="flex" flexDirection="row" justifyContent="flex-start" alignItems="center">
                <IconButton
                    aria-label={t("Cancel")}
                    onClick={onCancelReply}
                    sx={replyToCloseIconStyle}
                >
                    <IconCommon
                        decorative
                        name="Cancel"
                    />
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
    const { t } = useTranslation();

    return (
        <Box p={1}>
            <Box display="flex" flexDirection="row" justifyContent="flex-start" alignItems="center">
                <IconButton
                    aria-label={t("Cancel")}
                    onClick={onCancelEdit}
                    sx={replyToCloseIconStyle}
                >
                    <IconCommon
                        decorative
                        name="Cancel"
                    />
                </IconButton>
                <Typography variant="body2" color="textSecondary">
                    Editing message
                </Typography>
            </Box>
        </Box>
    );
}

type OuterProps = {
    isMobile: boolean;
}
const Outer = styled(Box)<OuterProps>(({ isMobile }) => ({
    width: "100%",
    maxWidth: `min(100%, ${MAX_CHAT_INPUT_WIDTH}px)`,
    margin: "auto",
    "& .advanced-input": {
        // Remove rounded bottom corners on mobile
        borderBottomLeftRadius: isMobile ? 0 : undefined,
        borderBottomRightRadius: isMobile ? 0 : undefined,
    },
}));

type ChatMessageInputProps = Pick<ChatIndicatorProps, "participantsTyping"> & {
    disabled: boolean;
    display: ViewDisplayType;
    isLoading: boolean;
    message: string;
    messageBeingEdited: ChatMessageShape | null;
    messageBeingRepliedTo: ChatMessageShape | null;
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
    participantsTyping,
    placeholder,
    setMessage,
    stopEditingMessage,
    stopReplyingToMessage,
    submitMessage,
    taskInfo,
}: ChatMessageInputProps) {
    const isMobile = useWindowSize(({ width }) => width <= MAX_CHAT_INPUT_WIDTH);

    const handleSubmit = useCallback(function handleAddMessageCallback(message: string) {
        if (disabled || isLoading) return;
        submitMessage(message);
    }, [disabled, isLoading, submitMessage]);

    return (
        <Outer isMobile={isMobile}>
            <ChatIndicator
                participantsTyping={participantsTyping}
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
            <AdvancedInputBase
                contextData={[]}
                disabled={disabled}
                maxChars={1500}
                onChange={setMessage}
                onSubmit={handleSubmit}
                name="newMessage"
                placeholder={placeholder}
                // taskInfo={taskInfo}
                tools={[]}
                value={message}
            />
        </Outer>
    );
}
