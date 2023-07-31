// Used to display popular/search results of a particular object type
import { adaHandleRegex, ResourceUsedFor, urlRegex, walletAddressRegex } from "@local/shared";
import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { DeleteIcon, EditIcon, OpenInNewIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { openLink, useLocation } from "route";
import { multiLineEllipsis } from "styles";
import { ResourceType } from "utils/consts";
import { getResourceIcon } from "utils/display/getResourceIcon";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import usePress from "utils/hooks/usePress";
import { getResourceUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ResourceListItemProps } from "../types";

/**
 * Determines if a resource is a URL, wallet payment address, or an ADA handle
 * @param link String to check
 * @returns ResourceType if type found, or null if not
 */
const getResourceType = (link: string): ResourceType | null => {
    if (urlRegex.test(link)) return ResourceType.Url;
    if (walletAddressRegex.test(link)) return ResourceType.Wallet;
    if (adaHandleRegex.test(link)) return ResourceType.Handle;
    return null;
};

export function ResourceListItem({
    canUpdate,
    data,
    handleContextMenu,
    handleDelete,
    handleEdit,
    index,
    loading,
}: ResourceListItemProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { title, subtitle } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);

    const Icon = useMemo(() => getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link), [data]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback((target: EventTarget) => {
        // Ignore if clicked edit or delete button
        if (target.id && ["delete-icon-button", "edit-icon-button"].includes(target.id)) return;
        // If no resource type or link, show error
        const resourceType = getResourceType(data.link);
        if (!resourceType || !href) {
            PubSub.get().publishSnack({ messageKey: "CannotOpenLink", severity: "Error" });
            return;
        }
        // Open link
        else openLink(setLocation, href);
    }, [data.link, href, setLocation]);

    const onEdit = useCallback((e: any) => {
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

    return (
        <Tooltip placement="top" title="Open in new tab">
            <ListItem
                disablePadding
                {...pressEvents}
                onClick={(e) => { e.preventDefault(); }}
                component="a"
                href={href}
                sx={{
                    display: "flex",
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    borderBottom: `1px solid ${palette.divider}`,
                    padding: 1,
                    cursor: "pointer",
                }}
            >
                <IconButton sx={{
                    width: "48px",
                    height: "48px",
                }}>
                    <Icon fill={palette.background.textPrimary} width="80%" height="80%" />
                </IconButton>
                <Stack direction="column" spacing={1} pl={2} sx={{ width: "-webkit-fill-available" }}>
                    {/* Name/Title */}
                    {loading ? <TextLoading /> : <ListItemText
                        primary={firstString(title, data.link)}
                        sx={{ ...multiLineEllipsis(1) }}
                    />}
                    {/* Bio/Description */}
                    {loading ? <TextLoading /> : <ListItemText
                        primary={subtitle}
                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                    />}
                </Stack>
                {
                    canUpdate && <IconButton id='delete-icon-button' onClick={onDelete}>
                        <DeleteIcon fill={palette.background.textPrimary} />
                    </IconButton>
                }
                {
                    canUpdate && <IconButton id='edit-icon-button' onClick={onEdit}>
                        <EditIcon fill={palette.background.textPrimary} />
                    </IconButton>
                }
                <IconButton>
                    <OpenInNewIcon fill={palette.background.textPrimary} />
                </IconButton>
            </ListItem>
        </Tooltip>
    );
}
