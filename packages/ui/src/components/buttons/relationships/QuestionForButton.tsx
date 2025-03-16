import { exists, isOfType, QuestionForType, User } from "@local/shared";
import { Tooltip } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AddIcon, ApiIcon, ArticleIcon, NoteIcon, ProjectIcon, RoutineIcon, StandardIcon, TeamIcon, TerminalIcon } from "../../../icons/common.js";
import { useLocation } from "../../../route/router.js";
import { SvgComponent } from "../../../types.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { getDisplay, placeholderColor } from "../../../utils/display/listTools.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { FindObjectDialog } from "../../dialogs/FindObjectDialog/FindObjectDialog.js";
import { RelationshipItemQuestionForObject } from "../../lists/types.js";
import { RelationshipAvatar, RelationshipButton, RelationshipChip } from "./styles.js";
import { QuestionForButtonProps } from "./types.js";

const MAX_LABEL_LENGTH = 20;
const TARGET_IMAGE_SIZE = 100;

/**
 * Maps QuestionForTypes to their icons
 */
const questionForTypeIcons: Record<QuestionForType, SvgComponent> = {
    Api: ApiIcon,
    Code: TerminalIcon,
    Note: NoteIcon,
    Project: ProjectIcon,
    Routine: RoutineIcon,
    Standard: StandardIcon,
    Team: TeamIcon,
};

const limitTo = [
    "Api",
    "DataConverter",
    "Note",
    "Project",
    "Prompt",
    "RoutineSingleStep",
    "RoutineMultiStep",
    "SmartContract",
    "Team",
] as const;

export function QuestionForButton({
    isEditing,
}: QuestionForButtonProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const [field, , helpers] = useField("forObject");

    // Select object dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
        ev.stopPropagation();
        const forObject = field?.value;
        // If not editing, navigate to object's page
        if (!isEditing) {
            if (forObject) openObject(forObject, setLocation);
        }
        else {
            setDialogOpen(true);
        }
    }, [field?.value, isEditing, setLocation]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((forObject: RelationshipItemQuestionForObject | null | undefined) => {
        if (!forObject) return;
        const forObjectId = field?.value?.id;
        if (forObject?.id === forObjectId) return;
        exists(helpers) && helpers.setValue(forObject);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    const { Icon, label, tooltip, avatarProps } = useMemo(() => {
        const forObject = field?.value;
        // If no data
        if (!forObject) {
            // If not editing, don't show anything
            if (!isEditing) return {
                Icon: undefined,
                label: null,
                tooltip: null,
                avatarProps: null,
            };
            // Otherwise, mark as unset
            return {
                Icon: AddIcon,
                label: "Connect to object",
                tooltip: t(`QuestionForNoneTogglePress${isEditing ? "Editable" : ""}`),
                avatarProps: null,
            };
        }
        const questionForName = getDisplay(forObject).title ?? null;
        const imageUrl = extractImageUrl(forObject.profileImage, forObject.updated_at, TARGET_IMAGE_SIZE);
        const isBot = isOfType(forObject, "User") && (forObject as Partial<User>).isBot === true;
        const label = questionForName ? `For: ${questionForName}` : "Question";
        const truncatedLabel = label.length > MAX_LABEL_LENGTH ? `${label.slice(0, MAX_LABEL_LENGTH)}...` : label;
        const Icon = questionForTypeIcons[forObject.__typename.replace("Version", "")];

        return {
            Icon: null,
            label: truncatedLabel,
            tooltip: t(`QuestionForTogglePress${isEditing ? "Editable" : ""}`, { questionFor: questionForName || "" }),
            avatarProps: {
                children: Icon ? <Icon /> : <ArticleIcon />,
                isBot,
                profileColors,
                src: imageUrl,
            },
        };
    }, [field?.value, t, isEditing, profileColors]);

    const Avatar = useMemo(() => {
        return avatarProps ? <RelationshipAvatar {...avatarProps} /> : undefined;
    }, [avatarProps]);

    // If not editing and no object, return null
    if (!isEditing && !Avatar && !Icon) return null;
    // If editing, return button and popups for changing data
    if (isEditing) {
        return (
            <>
                <FindObjectDialog
                    find="List"
                    isOpen={isDialogOpen}
                    handleCancel={closeDialog}
                    handleComplete={handleSelect as (data: object) => unknown}
                    limitTo={limitTo}
                />
                <Tooltip title={tooltip}>
                    <RelationshipButton
                        onClick={handleClick}
                        startIcon={Avatar || (Icon && <Icon />)}
                        variant="outlined"
                    >
                        {label}
                    </RelationshipButton>
                </Tooltip>
            </>
        );
    }
    // Otherwise, return chip
    return (
        <RelationshipChip
            icon={Avatar || (Icon && <Icon />) || undefined}
            label={label}
            onClick={handleClick}
        />
    );
}
