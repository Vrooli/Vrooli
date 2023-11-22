import { Session } from "@local/shared";
import { Stack, useTheme } from "@mui/material";
import { FocusModeButton, IsCompleteButton, IsPrivateButton, MeetingButton, MembersButton, OwnerButton, ParentButton, ProjectButton, QuestionForButton, RunProjectButton, RunRoutineButton } from "components/buttons/relationships";
import { OrganizationButton } from "components/buttons/relationships/OrganizationButton/OrganizationButton";
import { ParticipantsButton } from "components/buttons/relationships/ParticipantsButton/ParticipantsButton";
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
            {shouldShowButton("Project") && <ProjectButton {...props} />}
            {shouldShowButton("Parent") && <ParentButton {...props} />}
            {shouldShowButton("IsPrivate") && <IsPrivateButton {...props} />}
            {shouldShowButton("IsComplete") && <IsCompleteButton {...props} />}
            {shouldShowButton("FocusMode") && <FocusModeButton {...props} />}
            {shouldShowButton("Meeting") && <MeetingButton {...props} />}
            {shouldShowButton("RunProject") && <RunProjectButton {...props} />}
            {shouldShowButton("RunRoutine") && <RunRoutineButton {...props} />}
            {shouldShowButton("QuestionFor") && <QuestionForButton {...props} />}
            {shouldShowButton("Members") && <MembersButton {...props} />}
            {shouldShowButton("Participants") && <ParticipantsButton {...props} />}
            {shouldShowButton("Organization") && <OrganizationButton {...props} />}
            {shouldShowButton("User") && <UserButton {...props} />}
        </Stack>
    );
}
