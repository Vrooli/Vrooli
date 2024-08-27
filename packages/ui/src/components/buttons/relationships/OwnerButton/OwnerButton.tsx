import { OwnerShape, User, exists, getTranslation, noop } from "@local/shared";
import { Tooltip } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { ListMenu } from "components/dialogs/ListMenu/ListMenu";
import { FindObjectType, ListMenuItemData } from "components/dialogs/types";
import { userFromSession } from "components/lists/RelationshipList/RelationshipList";
import { RelationshipItemTeam, RelationshipItemUser } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { usePopover } from "hooks/usePopover";
import { TeamIcon, UserIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { RelationshipAvatar, RelationshipButton, RelationshipChip } from "../styles";
import { OwnerButtonProps } from "../types";

enum OwnerTypesEnum {
    Self = "Self",
    Team = "Team",
}

const MAX_LABEL_LENGTH = 20;
const TARGET_IMAGE_SIZE = 100;

const ownerTypes: ListMenuItemData<OwnerTypesEnum>[] = [
    { labelKey: "Self", value: OwnerTypesEnum.Self },
    { labelKey: "Team", value: OwnerTypesEnum.Team },
];

export function OwnerButton({
    isEditing,
}: OwnerButtonProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session]);
    const profileColors = useMemo(() => placeholderColor(), []);

    const [versionField, , versionHelpers] = useField("owner");
    const [rootField, , rootHelpers] = useField("root.owner");

    // Team/User owner dialogs (displayed after selecting owner dialog)
    const [isTeamDialogOpen, setTeamDialogOpen] = useState<boolean>(false);
    const openTeamDialog = useCallback(() => { setTeamDialogOpen(true); }, [setTeamDialogOpen]);
    const closeTeamDialog = useCallback(() => { setTeamDialogOpen(false); }, [setTeamDialogOpen]);
    const handleOwnerSelect = useCallback((owner: OwnerShape) => {
        const ownerId = versionField?.value?.id ?? rootField?.value?.id;
        if (owner?.id === ownerId) return;
        exists(versionHelpers) && versionHelpers.setValue(owner);
        exists(rootHelpers) && rootHelpers.setValue(owner);
        closeTeamDialog();
    }, [versionField?.value?.id, rootField?.value?.id, versionHelpers, rootHelpers, closeTeamDialog]);

    // Owner list dialog (select self, team, or another user)
    const [ownerDialogAnchor, openOwnerDialog, closeOwnerDialog] = usePopover();
    const handleOwnerClick = useCallback((ev: React.MouseEvent<HTMLElement>) => {
        ev.stopPropagation();
        const owner = versionField?.value ?? rootField?.value;
        // If not editing, navigate to owner
        if (!isEditing) {
            if (owner) openObject(owner, setLocation);
        }
        // Otherwise, open dialog
        else openOwnerDialog(ev);
    }, [versionField?.value, rootField?.value, isEditing, openOwnerDialog, setLocation]);
    const handleOwnerDialogSelect = useCallback((ownerType: OwnerTypesEnum) => {
        if (ownerType === OwnerTypesEnum.Team) {
            openTeamDialog();
        } else {
            const owner = session ? userFromSession(session) : undefined;
            console.log("self owner", owner);
            exists(versionHelpers) && versionHelpers.setValue(owner);
            exists(rootHelpers) && rootHelpers.setValue(owner);
        }
        closeOwnerDialog();
    }, [closeOwnerDialog, openTeamDialog, session, versionHelpers, rootHelpers]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[FindObjectType | null, (data: object) => unknown, () => unknown]>(() => {
        if (isTeamDialogOpen) return ["Team", handleOwnerSelect as (data: object) => unknown, closeTeamDialog];
        return [null, noop, noop];
    }, [isTeamDialogOpen, handleOwnerSelect, closeTeamDialog]);
    const limitTo = useMemo(function limitToMemo() {
        return findType ? [findType] : [];
    }, [findType]);

    const { avatarProps, label, tooltip } = useMemo(() => {
        const owner = versionField?.value ?? rootField?.value;
        // If no owner data, marked as anonymous
        if (!owner) return {
            avatarProps: null,
            label: "No owner",
            tooltip: t(`OwnerNoneTogglePress${isEditing ? "Editable" : ""}`),
        };
        // Otherwise, find owner information
        const isTeam = owner.__typename === "Team";
        const ownerName = isTeam
            ? firstString(getTranslation(owner as RelationshipItemTeam, languages, true).name, "team")
            : (owner as RelationshipItemUser).name;
        const isSelf = !isTeam && owner.id === getCurrentUser(session).id;
        const isBot = !isTeam && (owner as Partial<User>).isBot === true;
        const imageUrl = extractImageUrl(owner.profileImage, owner.updated_at, TARGET_IMAGE_SIZE);
        const label = `By: ${isSelf ? t("Self") : ownerName}`;
        const truncatedLabel = label.length > MAX_LABEL_LENGTH ? `${label.slice(0, MAX_LABEL_LENGTH)}...` : label;

        return {
            label: truncatedLabel,
            tooltip: t(`OwnerTogglePress${isEditing ? "Editable" : ""}`, { owner: isSelf ? t("Self") : ownerName }),
            avatarProps: {
                children: isTeam ? <TeamIcon /> : <UserIcon />,
                isBot,
                profileColors,
                src: imageUrl,
            },
        };
    }, [isEditing, languages, profileColors, rootField?.value, session, t, versionField?.value]);

    const Avatar = useMemo(function avatarMemo() {
        return avatarProps ? <RelationshipAvatar {...avatarProps} /> : undefined;
    }, [avatarProps]);

    // If editing, return button and popups for choosing owner type and owner
    if (isEditing) {
        return (
            <>
                {/* Popup for selecting type of owner */}
                <ListMenu
                    id={"select-owner-type-menu"}
                    anchorEl={ownerDialogAnchor}
                    title={t("Owner")}
                    data={ownerTypes}
                    onSelect={handleOwnerDialogSelect}
                    onClose={closeOwnerDialog}
                />
                {/* Popup for selecting team or user */}
                {findType && <FindObjectDialog
                    find="List"
                    isOpen={Boolean(findType)}
                    handleCancel={findHandleClose}
                    handleComplete={findHandleAdd}
                    limitTo={limitTo}
                />}
                <Tooltip title={tooltip}>
                    <RelationshipButton
                        onClick={handleOwnerClick}
                        startIcon={Avatar}
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
            icon={Avatar}
            label={label}
            onClick={handleOwnerClick}
        />
    );
}
