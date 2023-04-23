import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ResourceUsedFor } from "@local/consts";
import { DeleteIcon, EditIcon, OpenInNewIcon } from "@local/icons";
import { adaHandleRegex, urlRegex, walletAddressRegex } from "@local/validation";
import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { multiLineEllipsis } from "../../../../styles";
import { ResourceType } from "../../../../utils/consts";
import { getResourceIcon } from "../../../../utils/display/getResourceIcon";
import { getDisplay } from "../../../../utils/display/listTools";
import { firstString } from "../../../../utils/display/stringTools";
import { getUserLanguages } from "../../../../utils/display/translationTools";
import usePress from "../../../../utils/hooks/usePress";
import { getResourceUrl } from "../../../../utils/navigation/openObject";
import { PubSub } from "../../../../utils/pubsub";
import { openLink, useLocation } from "../../../../utils/route";
import { SessionContext } from "../../../../utils/SessionContext";
import { TextLoading } from "../../TextLoading/TextLoading";
const getResourceType = (link) => {
    if (urlRegex.test(link))
        return ResourceType.Url;
    if (walletAddressRegex.test(link))
        return ResourceType.Wallet;
    if (adaHandleRegex.test(link))
        return ResourceType.Handle;
    return null;
};
export function ResourceListItem({ canUpdate, data, handleContextMenu, handleDelete, handleEdit, index, loading, }) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { title, subtitle } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);
    const Icon = useMemo(() => getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link), [data]);
    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback((target) => {
        if (target.id && ["delete-icon-button", "edit-icon-button"].includes(target.id))
            return;
        const resourceType = getResourceType(data.link);
        if (!resourceType || !href) {
            PubSub.get().publishSnack({ messageKey: "CannotOpenLink", severity: "Error" });
            return;
        }
        else
            openLink(setLocation, href);
    }, [data.link, href, setLocation]);
    const onEdit = useCallback((e) => {
        handleEdit(index);
    }, [handleEdit, index]);
    const onDelete = useCallback(() => {
        handleDelete(index);
    }, [handleDelete, index]);
    const pressEvents = usePress({
        onLongPress: (target) => { handleContextMenu(target, index); },
        onClick: handleClick,
        onRightClick: (target) => { handleContextMenu(target, index); },
    });
    return (_jsx(Tooltip, { placement: "top", title: "Open in new tab", children: _jsxs(ListItem, { disablePadding: true, ...pressEvents, onClick: (e) => { e.preventDefault(); }, component: "a", href: href, sx: {
                display: "flex",
                background: palette.background.paper,
                color: palette.background.textPrimary,
                borderBottom: `1px solid ${palette.divider}`,
                padding: 1,
                cursor: "pointer",
            }, children: [_jsx(IconButton, { sx: {
                        width: "48px",
                        height: "48px",
                    }, children: _jsx(Icon, { fill: palette.background.textPrimary, width: "80%", height: "80%" }) }), _jsxs(Stack, { direction: "column", spacing: 1, pl: 2, sx: { width: "-webkit-fill-available" }, children: [loading ? _jsx(TextLoading, {}) : _jsx(ListItemText, { primary: firstString(title, data.link), sx: { ...multiLineEllipsis(1) } }), loading ? _jsx(TextLoading, {}) : _jsx(ListItemText, { primary: subtitle, sx: { ...multiLineEllipsis(2), color: palette.text.secondary } })] }), canUpdate && _jsx(IconButton, { id: 'delete-icon-button', onClick: onDelete, children: _jsx(DeleteIcon, { fill: palette.background.textPrimary }) }), canUpdate && _jsx(IconButton, { id: 'edit-icon-button', onClick: onEdit, children: _jsx(EditIcon, { fill: palette.background.textPrimary }) }), _jsx(IconButton, { children: _jsx(OpenInNewIcon, { fill: palette.background.textPrimary }) })] }) }));
}
//# sourceMappingURL=ResourceListItem.js.map