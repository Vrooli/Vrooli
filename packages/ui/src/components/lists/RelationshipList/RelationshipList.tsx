import { GqlModelType, Session } from "@local/shared";
import { Stack, useTheme } from "@mui/material";
import { FocusModeButton, IsCompleteButton, IsPrivateButton, MeetingButton, MembersButton, OwnerButton, QuestionForButton, RunProjectButton, RunRoutineButton } from "components/buttons/relationships";
import { ParticipantsButton } from "components/buttons/relationships/ParticipantsButton/ParticipantsButton";
import { TeamButton } from "components/buttons/relationships/TeamButton/TeamButton";
import { UserButton } from "components/buttons/relationships/UserButton/UserButton";
import { noSelect } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { OwnerShape } from "utils/shape/models/types";
import { RelationshipButtonType, RelationshipListProps } from "../types";

/**
 * Converts session to user object
 * @param session Current user session
 * @returns User object
 */
export const userFromSession = (session: Session): Exclude<OwnerShape, null> => ({
    __typename: "User",
    id: getCurrentUser(session).id as string,
    handle: null,
    name: "Self",
});

//TODO: working on a way to be able to know how many buttons will be shown, to display it differently in some cases.
//If it's public, not your own, and only the public button is shown, we can hide it altogether

/** Available relationship buttions, in display order */
const buttonTypes: RelationshipButtonType[] = [
    "Owner", // Who owns the object (when it can be you or a team)
    "Parent", // Parent object (when forked)
    "IsPrivate", // Whether the object is private
    "IsComplete", // Whether the object is complete, if versioned
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
    Owner: ["Api", "Code", "Comment", "Label", "Note", "Project", "Routine", "Standard"],
    IsPrivate: ["all"],
    IsComplete: ["all"],
    FocusMode: ["all"],
    Meeting: ["all"],
    RunProject: ["all"],
    RunRoutine: ["all"],
    QuestionFor: ["all"],
    Members: ["all"],
    Participants: ["all"],
    Team: ["all"],
    User: ["all"],
} as any;// TODO complete and implement

/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
export function RelationshipList({
    limitTo,
    ...props
}: RelationshipListProps) {
    const { palette } = useTheme();

    const shouldShowButton = (type: RelationshipButtonType): boolean => {
        // If no limit is specified, show all buttons
        if (!limitTo) return true;
        // Otherwise, show only the buttons specified in the limitTo array
        return limitTo.includes(type);
    };

    return (
        <Stack
            spacing={{ xs: 1, sm: 1.5, md: 2 }}
            direction="row"
            alignItems="center"
            justifyContent="center"
            p={1}
            sx={{
                borderRadius: 1,
                background: palette.mode === "dark" ? palette.background.paper : palette.background.default,
                overflowX: "auto",
                ...noSelect,
                ...(props.sx ?? {}),
                "@media print": {
                    border: `1px solid ${palette.divider}`,
                },
            }}
        >
            {shouldShowButton("Owner") && <OwnerButton {...props} />}
            {shouldShowButton("IsPrivate") && <IsPrivateButton {...props} />}
            {shouldShowButton("IsComplete") && <IsCompleteButton {...props} />}
            {shouldShowButton("FocusMode") && <FocusModeButton {...props} />}
            {shouldShowButton("Meeting") && <MeetingButton {...props} />}
            {shouldShowButton("RunProject") && <RunProjectButton {...props} />}
            {shouldShowButton("RunRoutine") && <RunRoutineButton {...props} />}
            {shouldShowButton("QuestionFor") && <QuestionForButton {...props} />}
            {shouldShowButton("Members") && <MembersButton {...props} />}
            {shouldShowButton("Participants") && <ParticipantsButton {...props} />}
            {shouldShowButton("Team") && <TeamButton {...props} />}
            {shouldShowButton("User") && <UserButton {...props} />}
        </Stack>
    );
}
