import { ModelType, OwnerShape, Session } from "@local/shared";
import { Box, styled } from "@mui/material";
import { useMemo } from "react";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { ELEMENT_IDS, RelationshipButtonType } from "../../../utils/consts.js";
import { FocusModeButton } from "../../buttons/relationships/FocusModeButton.js";
import { IsCompleteButton } from "../../buttons/relationships/IsCompleteButton.js";
import { IsPrivateButton } from "../../buttons/relationships/IsPrivateButton.js";
import { MembersButton } from "../../buttons/relationships/MembersButton.js";
import { OwnerButton } from "../../buttons/relationships/OwnerButton.js";
import { ParticipantsButton } from "../../buttons/relationships/ParticipantsButton.js";
import { QuestionForButton } from "../../buttons/relationships/QuestionForButton.js";
import { RelationshipListProps } from "../types.js";

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

/** Map of button types to objects they're shown on */
const buttonTypeMap: Record<RelationshipButtonType, (ModelType | `${ModelType}`)[]> = {
    IsPrivate: ["Api", "Code", "Note", "Project", "Routine", "RunProject", "RunRoutine", "Standard", "Team", "User"],
    IsComplete: ["Project", "Routine"],
    Owner: ["Api", "Code", "Comment", "Label", "Note", "Project", "Routine", "Standard"],
    FocusMode: ["Reminder"],
    QuestionFor: ["Question"],
    Members: ["Team"],
    Participants: ["Chat"],
};

const OuterBox = styled(Box)(({ theme }) => ({
    alignItems: "center",
    background: theme.palette.background.paper,
    borderRadius: theme.spacing(3),
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    justifyContent: "flex-start",
    overflowX: "auto",
    padding: theme.spacing(2),
}));

/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
export function RelationshipList({
    limitTo,
    ...props
}: RelationshipListProps) {
    const visibleButtons = useMemo(() => {
        return Object.values(RelationshipButtonType).filter(buttonType => {
            if (limitTo && !limitTo.includes(buttonType)) {
                return false;
            }

            const allowedTypes = buttonTypeMap[buttonType];
            if (allowedTypes.includes(props.objectType)) {
                return true;
            }

            return false;
        });
    }, [limitTo, props.objectType]);

    if (!visibleButtons.length) {
        return null;
    }
    return (
        <OuterBox id={ELEMENT_IDS.RelationshipList}>
            {visibleButtons.includes(RelationshipButtonType.IsPrivate) && <IsPrivateButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.IsComplete) && <IsCompleteButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.Owner) && <OwnerButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.FocusMode) && <FocusModeButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.QuestionFor) && <QuestionForButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.Members) && <MembersButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.Participants) && <ParticipantsButton {...props} />}
        </OuterBox>
    );
}
