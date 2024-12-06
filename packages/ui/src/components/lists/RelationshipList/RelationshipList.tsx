import { GqlModelType, OwnerShape, Session } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { FocusModeButton } from "components/buttons/relationships/FocusModeButton/FocusModeButton";
import { IsCompleteButton } from "components/buttons/relationships/IsCompleteButton/IsCompleteButton";
import { IsPrivateButton } from "components/buttons/relationships/IsPrivateButton/IsPrivateButton";
import { MembersButton } from "components/buttons/relationships/MembersButton/MembersButton";
import { OwnerButton } from "components/buttons/relationships/OwnerButton/OwnerButton";
import { ParticipantsButton } from "components/buttons/relationships/ParticipantsButton/ParticipantsButton";
import { QuestionForButton } from "components/buttons/relationships/QuestionForButton/QuestionForButton";
import { useMemo } from "react";
import { formSection, noSelect } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { ELEMENT_IDS, RelationshipButtonType } from "utils/consts";
import { RelationshipListProps } from "../types";

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
const buttonTypeMap: Record<RelationshipButtonType, (GqlModelType | `${GqlModelType}`)[]> = {
    IsPrivate: ["Api", "Code", "Note", "Project", "Routine", "RunProject", "RunRoutine", "Standard", "Team", "User"],
    IsComplete: ["Project", "Routine"],
    Owner: ["Api", "Code", "Comment", "Label", "Note", "Project", "Routine", "Standard"],
    FocusMode: ["Reminder"],
    QuestionFor: ["Question"],
    Members: ["Team"],
    Participants: ["Chat"],
};

/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
export function RelationshipList({
    limitTo,
    ...props
}: RelationshipListProps) {
    const theme = useTheme();

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
            background: theme.palette.background.paper,
            ...props.sx,
        } as const;
    }, [theme, props.sx]);

    if (!visibleButtons.length) {
        return null;
    }
    return (
        <Box id={ELEMENT_IDS.RelationshipList} sx={outerStyle}>
            {visibleButtons.includes(RelationshipButtonType.IsPrivate) && <IsPrivateButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.IsComplete) && <IsCompleteButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.Owner) && <OwnerButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.FocusMode) && <FocusModeButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.QuestionFor) && <QuestionForButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.Members) && <MembersButton {...props} />}
            {visibleButtons.includes(RelationshipButtonType.Participants) && <ParticipantsButton {...props} />}
        </Box>
    );
}
