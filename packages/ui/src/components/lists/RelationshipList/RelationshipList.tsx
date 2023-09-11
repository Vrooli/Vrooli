import { Session } from "@local/shared";
import { Stack, useTheme } from "@mui/material";
import { FocusModeButton, IsCompleteButton, IsPrivateButton, MeetingButton, MembersButton, OwnerButton, ParentButton, ProjectButton, QuestionForButton, RunProjectButton, RunRoutineButton } from "components/buttons/relationships";
import { OrganizationButton } from "components/buttons/relationships/OrganizationButton/OrganizationButton";
import { ParticipantsButton } from "components/buttons/relationships/ParticipantsButton/ParticipantsButton";
import { UserButton } from "components/buttons/relationships/UserButton/UserButton";
import { noSelect } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { OwnerShape } from "utils/shape/models/types";
import { RelationshipListProps } from "../types";

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
export function RelationshipList(props: RelationshipListProps) {
    const { palette } = useTheme();

    return (
        <Stack
            spacing={{ xs: 1, sm: 1.5, md: 2 }}
            direction="row"
            alignItems="center"
            justifyContent="center"
            p={1}
            sx={{
                borderRadius: 2,
                background: palette.mode === "dark" ? palette.background.paper : palette.background.default,
                overflowX: "auto",
                ...noSelect,
                ...(props.sx ?? {}),
                "@media print": {
                    border: `1px solid ${palette.divider}`,
                },
            }}
        >
            {/* Buttons applicable to main objects (e.g. projects, notes, routines, organizations) */}
            <OwnerButton {...props} />
            <ProjectButton {...props} />
            <ParentButton {...props} />
            <IsPrivateButton {...props} />
            <IsCompleteButton {...props} />
            {/* Buttons for special cases (e.g. schedules) */}
            <FocusModeButton {...props} />
            <MeetingButton {...props} />
            <RunProjectButton {...props} />
            <RunRoutineButton {...props} />
            <QuestionForButton {...props} />
            <MembersButton {...props} />
            <ParticipantsButton {...props} />
            <OrganizationButton {...props} />
            <UserButton {...props} />
        </Stack>
    );
}
