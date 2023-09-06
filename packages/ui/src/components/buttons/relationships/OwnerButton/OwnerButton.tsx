import { exists } from "@local/shared";
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { ListMenu } from "components/dialogs/ListMenu/ListMenu";
import { ListMenuItemData, SelectOrCreateObjectType } from "components/dialogs/types";
import { userFromSession } from "components/lists/RelationshipList/RelationshipList";
import { RelationshipItemOrganization, RelationshipItemUser } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { OrganizationIcon, UserIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { firstString } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { OwnerShape } from "utils/shape/models/types";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { OwnerButtonProps } from "../types";

enum OwnerTypesEnum {
    Self = "Self",
    Organization = "Organization",
}

const ownerTypes: ListMenuItemData<OwnerTypesEnum>[] = [
    { labelKey: "Self", value: OwnerTypesEnum.Self },
    { labelKey: "Organization", value: OwnerTypesEnum.Organization },
];

export function OwnerButton({
    isEditing,
    objectType,
}: OwnerButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [versionField, , versionHelpers] = useField("owner");
    const [rootField, , rootHelpers] = useField("root.owner");

    const isAvailable = useMemo(() => ["Project", "Routine", "Standard"].includes(objectType), [objectType]);

    // Organization/User owner dialogs (displayed after selecting owner dialog)
    const [isOrganizationDialogOpen, setOrganizationDialogOpen] = useState<boolean>(false);
    const openOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(true); }, [setOrganizationDialogOpen]);
    const closeOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(false); }, [setOrganizationDialogOpen]);
    const handleOwnerSelect = useCallback((owner: OwnerShape) => {
        const ownerId = versionField?.value?.id ?? rootField?.value?.id;
        if (owner?.id === ownerId) return;
        exists(versionHelpers) && versionHelpers.setValue(owner);
        exists(rootHelpers) && rootHelpers.setValue(owner);
        closeOrganizationDialog();
    }, [versionField?.value?.id, rootField?.value?.id, versionHelpers, rootHelpers, closeOrganizationDialog]);

    // Owner list dialog (select self, organization, or another user)
    const [ownerDialogAnchor, setOwnerDialogAnchor] = useState<any>(null);
    const handleOwnerClick = useCallback((ev: React.MouseEvent<Element>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const owner = versionField?.value ?? rootField?.value;
        // If not editing, navigate to owner
        if (!isEditing) {
            if (owner) openObject(owner, setLocation);
        }
        // Otherwise, open dialog
        else setOwnerDialogAnchor(ev.currentTarget);
    }, [isEditing, isAvailable, versionField?.value, rootField?.value, setLocation]);
    const closeOwnerDialog = useCallback(() => setOwnerDialogAnchor(null), []);
    const handleOwnerDialogSelect = useCallback((ownerType: OwnerTypesEnum) => {
        if (ownerType === OwnerTypesEnum.Organization) {
            openOrganizationDialog();
        } else {
            const owner = session ? userFromSession(session) : undefined;
            exists(versionHelpers) && versionHelpers.setValue(owner);
            exists(rootHelpers) && rootHelpers.setValue(owner);
        }
        closeOwnerDialog();
    }, [closeOwnerDialog, openOrganizationDialog, session, versionHelpers, rootHelpers]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => unknown, () => unknown]>(() => {
        if (isOrganizationDialogOpen) return ["Organization", handleOwnerSelect, closeOrganizationDialog];
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        return [null, () => { }, () => { }];
    }, [isOrganizationDialogOpen, handleOwnerSelect, closeOrganizationDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const owner = versionField?.value ?? rootField?.value;
        // If no owner data, marked as anonymous
        if (!owner) return {
            Icon: null,
            tooltip: t(`OwnerNoneTogglePress${isEditing ? "Editable" : ""}`),
        };
        // If owner is organization, use organization icon
        if (owner.__typename === "Organization") {
            const Icon = OrganizationIcon;
            const ownerName = firstString(getTranslation(owner as RelationshipItemOrganization, languages, true).name, "organization");
            return {
                Icon,
                tooltip: t(`OwnerTogglePress${isEditing ? "Editable" : ""}`, { owner: ownerName }),
            };
        }
        // If owner is user, use self icon
        const Icon = UserIcon;
        const isSelf = owner.id === getCurrentUser(session).id;
        const ownerName = (owner as RelationshipItemUser).name;
        return {
            Icon,
            tooltip: t(`OwnerTogglePress${isEditing ? "Editable" : ""}`, { owner: isSelf ? t("Self") : ownerName }),
        };
    }, [isEditing, languages, rootField?.value, session, t, versionField?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    // Return button with label on top
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
            {/* Popup for selecting organization or user */}
            {findType && <FindObjectDialog
                find="List"
                isOpen={Boolean(findType)}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={[findType]}
            />}
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
            >
                <TextShrink id="owner" sx={{ ...commonLabelProps() }}>{t("Owner")}</TextShrink>
                <Tooltip title={tooltip}>
                    <IconButton
                        onClick={handleOwnerClick}
                        sx={{ ...smallButtonProps(isEditing, true), background: palette.primary.light }}
                    >
                        {Icon && <Icon {...commonIconProps()} />}
                    </IconButton>
                </Tooltip>
            </Stack>
        </>
    );
}
