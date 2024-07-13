import { GqlModelType, Session } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { FocusModeButton, IsCompleteButton, IsPrivateButton, MeetingButton, MembersButton, OwnerButton, QuestionForButton, RunProjectButton, RunRoutineButton } from "components/buttons/relationships";
import { ParticipantsButton } from "components/buttons/relationships/ParticipantsButton/ParticipantsButton";
import { TeamButton } from "components/buttons/relationships/TeamButton/TeamButton";
import { UserButton } from "components/buttons/relationships/UserButton/UserButton";
import { useCallback, useMemo } from "react";
import { formSection, noSelect } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { OwnerShape } from "utils/shape/models/types";
import { RelationshipButtonType, RelationshipListProps } from "../types";

/**
 * Converts session to user object
 * @param session Current user session
 * @returns User object
 */
export function userFromSession(session: Session): Exclude<OwnerShape, null> {
    return {
        __typename: "User",
        id: getCurrentUser(session).id as string,
        handle: null,
        name: "Self",
        profileImage: getCurrentUser(session).profileImage,
    };
}

//TODO: working on a way to be able to know how many buttons will be shown, to display it differently in some cases.
//If it's public, not your own, and only the public button is shown, we can hide it altogether

/** Available relationship buttions, in display order */
const buttonTypes: RelationshipButtonType[] = [
    "IsPrivate", // Whether the object is private
    "IsComplete", // Whether the object is complete, if versioned
    "Owner", // Who owns the object (when it can be you or a team)
    "Parent", // Parent object (when forked)
    "FocusMode", // Associated focus mode
    "Meeting", // Associated meeting
    "RunProject", // Associated RunProject
    "RunRoutine", // Associated RunRoutine
    "QuestionFor", // Associated question
    "Members", // Members of a team object
    "Participants", // Participants of a chat object
    "Team", // Associated team object (NOT for owner)
    "User", // Associated user object (NOT for owner)
];

/** Map of button types to objects they're shown on */
const buttonTypeMap: Record<RelationshipButtonType, (GqlModelType | `${GqlModelType}`)[]> = {
    IsPrivate: ["Api", "Code", "Note", "Project", "Routine", "RunProject", "RunRoutine", "Standard", "Team", "User"],
    IsComplete: ["Project", "Routine"],
    Owner: ["Api", "Code", "Comment", "Label", "Note", "Project", "Routine", "Standard"],
    FocusMode: ["Reminder", "Schedule"],
    Meeting: ["Schedule"],
    RunProject: ["all"],
    RunRoutine: ["all"],
    QuestionFor: ["all"],
    Members: ["all"],
    Participants: ["all"],
    Team: ["all"],
    User: ["all"],
} as any;// TODO complete and implement

//TODO chain: update bot selector to change configCallData instead of being its own prop state, finish updating RelationshipButtons to new style, update RelationshipList to address TODOs, fix schedule findMany error, persistent snack when downloading new site update, remove relationship buttons only used by schedules

/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
export function RelationshipList({
    limitTo,
    ...props
}: RelationshipListProps) {
    const theme = useTheme();

    const shouldShowButton = useCallback(function shouldShowButtonCallback(type: RelationshipButtonType): boolean {
        // If no limit is specified, show all buttons
        if (!limitTo) return true;
        // Otherwise, show only the buttons specified in the limitTo array
        return limitTo.includes(type);
    }, [limitTo]);

    const outerStyle = useMemo(function outerStyleMemo() {
        return {
            ...noSelect,
            ...formSection(theme),
            display: "flex",
            flexDirection: "row",
            alignItems: "center",
            justifyContent: "flex-start",
            gap: 1,
            overflowX: "auto",
            ...props.sx,
        } as const;
    }, [theme, props.sx]);

    return (
        <Box sx={outerStyle}>
            {shouldShowButton("IsPrivate") && <IsPrivateButton {...props} />}
            {shouldShowButton("IsComplete") && <IsCompleteButton {...props} />}
            {shouldShowButton("Owner") && <OwnerButton {...props} />}
            {shouldShowButton("FocusMode") && <FocusModeButton {...props} />}
            {shouldShowButton("Meeting") && <MeetingButton {...props} />}
            {shouldShowButton("RunProject") && <RunProjectButton {...props} />}
            {shouldShowButton("RunRoutine") && <RunRoutineButton {...props} />}
            {shouldShowButton("QuestionFor") && <QuestionForButton {...props} />}
            {shouldShowButton("Members") && <MembersButton {...props} />}
            {shouldShowButton("Participants") && <ParticipantsButton {...props} />}
            {shouldShowButton("Team") && <TeamButton {...props} />}
            {shouldShowButton("User") && <UserButton {...props} />}
        </Box>
    );
}
