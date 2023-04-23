import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { OrganizationIcon, UserIcon } from "@local/icons";
import { exists } from "@local/utils";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { getCurrentUser } from "../../../../utils/authentication/session";
import { firstString } from "../../../../utils/display/stringTools";
import { getTranslation, getUserLanguages } from "../../../../utils/display/translationTools";
import { openObject } from "../../../../utils/navigation/openObject";
import { useLocation } from "../../../../utils/route";
import { SessionContext } from "../../../../utils/SessionContext";
import { FindObjectDialog } from "../../../dialogs/FindObjectDialog/FindObjectDialog";
import { ListMenu } from "../../../dialogs/ListMenu/ListMenu";
import { userFromSession } from "../../../lists/RelationshipList/RelationshipList";
import { TextShrink } from "../../../text/TextShrink/TextShrink";
import { ColorIconButton } from "../../ColorIconButton/ColorIconButton";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
var OwnerTypesEnum;
(function (OwnerTypesEnum) {
    OwnerTypesEnum["Self"] = "Self";
    OwnerTypesEnum["Organization"] = "Organization";
    OwnerTypesEnum["AnotherUser"] = "AnotherUser";
})(OwnerTypesEnum || (OwnerTypesEnum = {}));
const ownerTypes = [
    { label: "Self", value: OwnerTypesEnum.Self },
    { label: "Organization", value: OwnerTypesEnum.Organization },
];
export function OwnerButton({ isEditing, objectType, zIndex, }) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session]);
    const [versionField, , versionHelpers] = useField("owner");
    const [rootField, , rootHelpers] = useField("root.owner");
    const isAvailable = useMemo(() => ["Project", "Routine", "Standard"].includes(objectType), [objectType]);
    const [isOrganizationDialogOpen, setOrganizationDialogOpen] = useState(false);
    const openOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(true); }, [setOrganizationDialogOpen]);
    const closeOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(false); }, [setOrganizationDialogOpen]);
    const [isAnotherUserDialogOpen, setAnotherUserDialogOpen] = useState(false);
    const openAnotherUserDialog = useCallback(() => { setAnotherUserDialogOpen(true); }, [setAnotherUserDialogOpen]);
    const closeAnotherUserDialog = useCallback(() => { setAnotherUserDialogOpen(false); }, [setAnotherUserDialogOpen]);
    const handleOwnerSelect = useCallback((owner) => {
        const ownerId = versionField?.value?.id ?? rootField?.value?.id;
        if (owner?.id === ownerId)
            return;
        exists(versionHelpers) && versionHelpers.setValue(owner);
        exists(rootHelpers) && rootHelpers.setValue(owner);
        closeOrganizationDialog();
        closeAnotherUserDialog();
    }, [versionField?.value?.id, rootField?.value?.id, versionHelpers, rootHelpers, closeOrganizationDialog, closeAnotherUserDialog]);
    const [ownerDialogAnchor, setOwnerDialogAnchor] = useState(null);
    const handleOwnerClick = useCallback((ev) => {
        if (!isAvailable)
            return;
        ev.stopPropagation();
        const owner = versionField?.value ?? rootField?.value;
        if (!isEditing) {
            if (owner)
                openObject(owner, setLocation);
        }
        else
            setOwnerDialogAnchor(ev.currentTarget);
    }, [isEditing, isAvailable, versionField?.value, rootField?.value, setLocation]);
    const closeOwnerDialog = useCallback(() => setOwnerDialogAnchor(null), []);
    const handleOwnerDialogSelect = useCallback((ownerType) => {
        if (ownerType === OwnerTypesEnum.Organization) {
            openOrganizationDialog();
        }
        else if (ownerType === OwnerTypesEnum.AnotherUser) {
            openAnotherUserDialog();
        }
        else {
            const owner = session ? userFromSession(session) : undefined;
            exists(versionHelpers) && versionHelpers.setValue(owner);
            exists(rootHelpers) && rootHelpers.setValue(owner);
        }
        closeOwnerDialog();
    }, [closeOwnerDialog, openOrganizationDialog, openAnotherUserDialog, session, versionHelpers, rootHelpers]);
    const [findType, findHandleAdd, findHandleClose] = useMemo(() => {
        if (isOrganizationDialogOpen)
            return ["Organization", handleOwnerSelect, closeOrganizationDialog];
        else if (isAnotherUserDialogOpen)
            return ["User", handleOwnerSelect, closeAnotherUserDialog];
        return [null, () => { }, () => { }];
    }, [isOrganizationDialogOpen, handleOwnerSelect, closeOrganizationDialog, isAnotherUserDialogOpen, closeAnotherUserDialog]);
    const { Icon, tooltip } = useMemo(() => {
        const owner = versionField?.value ?? rootField?.value;
        if (!owner)
            return {
                Icon: null,
                tooltip: `Marked as anonymous${isEditing ? "" : ". Press to set owner"}`,
            };
        if (owner.__typename === "Organization") {
            const Icon = OrganizationIcon;
            const ownerName = firstString(getTranslation(owner, languages, true).name, "organization");
            return {
                Icon,
                tooltip: `Owner: ${ownerName}`,
            };
        }
        const Icon = UserIcon;
        const isSelf = owner.id === getCurrentUser(session).id;
        const ownerName = owner.name;
        return {
            Icon,
            tooltip: `Owner: ${isSelf ? "Self" : ownerName}`,
        };
    }, [isEditing, languages, rootField?.value, session, versionField?.value]);
    if (!isAvailable || (!isEditing && !Icon))
        return null;
    return (_jsxs(_Fragment, { children: [_jsx(ListMenu, { id: "select-owner-type-menu", anchorEl: ownerDialogAnchor, title: 'Owner Type', data: ownerTypes, onSelect: handleOwnerDialogSelect, onClose: closeOwnerDialog, zIndex: zIndex + 1 }), findType && _jsx(FindObjectDialog, { find: "List", isOpen: Boolean(findType), handleCancel: findHandleClose, handleComplete: findHandleAdd, limitTo: [findType], zIndex: zIndex + 1 }), _jsxs(Stack, { direction: "column", alignItems: "center", justifyContent: "center", children: [_jsx(TextShrink, { id: "owner", sx: { ...commonLabelProps() }, children: "Owner" }), _jsx(Tooltip, { title: tooltip, children: _jsx(ColorIconButton, { background: palette.primary.light, sx: { ...commonButtonProps(isEditing, true) }, onClick: handleOwnerClick, children: Icon && _jsx(Icon, { ...commonIconProps() }) }) })] })] }));
}
//# sourceMappingURL=OwnerButton.js.map