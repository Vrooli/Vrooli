import { Count, DeleteManyInput, DeleteType, endpointPostDeleteMany } from "@local/shared";
import { Box, CircularProgress, IconButton, Stack, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { SessionContext } from "contexts/SessionContext";
import { useLazyFetch } from "hooks/useLazyFetch";
import usePress from "hooks/usePress";
import { ApiIcon, DeleteIcon, HelpIcon, LinkIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { multiLineEllipsis, noSelect } from "styles";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { DirectoryListItemContextMenu } from "../DirectoryListItemContextMenu/DirectoryListItemContextMenu";
import { initializeDirectoryList } from "../common";
import { DirectoryCardProps, DirectoryItem, DirectoryListProps } from "../types";

const DirectoryBox = styled(Box)(({ theme }) => ({
    ...noSelect,
    display: "block",
    boxShadow: theme.shadows[4],
    background: theme.palette.primary.light,
    color: theme.palette.secondary.contrastText,
    borderRadius: "8px",
    padding: theme.spacing(0.5),
    cursor: "pointer",
    width: "200px",
    minWidth: "200px",
    height: `calc(${theme.typography.h3.fontSize} * 2 + ${theme.spacing(1)})`,
    position: "relative",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
})) as any;// TODO: Fix any - https://github.com/mui/material-ui/issues/38274

/**
 * Unlike ResourceCard, these aren't draggable. This is because the objects 
 * are not stored in an order - they are stored by object type
 */
export const DirectoryCard = ({
    canUpdate,
    data,
    index,
    onContextMenu,
    onDelete,
}: DirectoryCardProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [showIcons, setShowIcons] = useState(false);

    const { title, subtitle } = useMemo(() => getDisplay(data as any, getUserLanguages(session)), [data, session]);

    const Icon = useMemo(() => {
        if (!data || !data.__typename) return HelpIcon;
        if (data.__typename === "ApiVersion") return ApiIcon;
        if (data.__typename === "NoteVersion") return NoteIcon;
        if (data.__typename === "Organization") return OrganizationIcon;
        if (data.__typename === "ProjectVersion") return ProjectIcon;
        if (data.__typename === "RoutineVersion") return RoutineIcon;
        if (data.__typename === "SmartContractVersion") return SmartContractIcon;
        if (data.__typename === "StandardVersion") return StandardIcon;
        return HelpIcon;
    }, [data]);

    const href = useMemo(() => data ? getObjectUrl(data as any) : "#", [data]);
    const handleClick = useCallback((target: EventTarget) => {
        // Check if delete button was clicked
        const targetId: string | undefined = target.id;
        if (targetId && targetId.startsWith("delete-")) {
            onDelete?.(index);
        }
        else {
            // Navigate to object
            setLocation(href);
        }
    }, [href, index, onDelete, setLocation]);
    const handleContextMenu = useCallback((target: EventTarget) => {
        onContextMenu(target, index);
    }, [onContextMenu, index]);

    const handleHover = useCallback(() => {
        if (canUpdate) {
            setShowIcons(true);
        }
    }, [canUpdate]);

    const handleHoverEnd = useCallback(() => { setShowIcons(false); }, []);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onHover: handleHover,
        onHoverEnd: handleHoverEnd,
        onRightClick: handleContextMenu,
        hoverDelay: 100,
    });

    return (
        <Tooltip placement="top" title={`${subtitle ? subtitle + " - " : ""}${href}`}>
            <DirectoryBox
                {...pressEvents}
                component="a"
                href={href}
                onClick={(e) => e.preventDefault()}
            >
                {/* delete icon, only visible on hover */}
                {showIcons && (
                    <>
                        <Tooltip title={t("Delete")}>
                            <IconButton
                                id='delete-icon-button'
                                sx={{ background: palette.error.main, position: "absolute", top: 4, right: 4 }}
                            >
                                <DeleteIcon id='delete-icon' fill={palette.secondary.contrastText} />
                            </IconButton>
                        </Tooltip>
                    </>
                )}
                {/* Content */}
                <Stack
                    direction="column"
                    justifyContent="center"
                    alignItems="center"
                    sx={{
                        height: "100%",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                    }}
                >
                    <Icon fill="white" />
                    <Typography
                        gutterBottom
                        variant="body2"
                        component="h3"
                        sx={{
                            ...multiLineEllipsis(2),
                            textAlign: "center",
                            lineBreak: title ? "auto" : "anywhere", // Line break anywhere only if showing link
                        }}
                    >
                        {title}
                    </Typography>
                </Stack>
            </DirectoryBox>
        </Tooltip>
    );
};

export const DirectoryListHorizontal = ({
    canUpdate = true,
    directory,
    handleUpdate,
    loading = false,
    mutate = true,
    sortBy,
}: DirectoryListProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    const list = useMemo(() => initializeDirectoryList(directory, sortBy, session), [directory, session, sortBy]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<HTMLElement | null>(null);
    const [selectedItem, setSelectedItem] = useState<DirectoryItem | null>(null);
    const selectedIndex = useMemo(() => selectedItem ? list.findIndex(r => r.id === selectedItem.id) : -1, [list, selectedItem]);
    const contextId = useMemo(() => `directory-context-menu-${selectedItem?.id}`, [selectedItem]);
    const openContext = useCallback((target: EventTarget, index: number) => {
        setContextAnchor(target as HTMLElement);
        const item = list[index];
        setSelectedItem(item);
    }, [list]);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelectedItem(null);
    }, []);

    // Add item dialog
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); }, []);

    const onAdd = useCallback((item: DirectoryItem) => {
        console.log("directoryyy onAdd", item);
        if (!directory) return;
        // Dont add duplicates
        if (list.some(r => r.id === item.id)) {
            PubSub.get().publish("snack", { message: "Item already added", severity: "Warning" });
            return;
        }
        if (handleUpdate) {
            handleUpdate({
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? [...directory.childApiVersions, item] : directory.childApiVersions as any[],
                childNoteVersions: item.__typename === "NoteVersion" ? [...directory.childNoteVersions, item] : directory.childNoteVersions as any[],
                childOrganizations: item.__typename === "Organization" ? [...directory.childOrganizations, item] : directory.childOrganizations as any[],
                childProjectVersions: item.__typename === "ProjectVersion" ? [...directory.childProjectVersions, item] : directory.childProjectVersions as any[],
                childRoutineVersions: item.__typename === "RoutineVersion" ? [...directory.childRoutineVersions, item] : directory.childRoutineVersions as any[],
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? [...directory.childSmartContractVersions, item] : directory.childSmartContractVersions as any[],
                childStandardVersions: item.__typename === "StandardVersion" ? [...directory.childStandardVersions, item] : directory.childStandardVersions as any[],
            });
        }
        closeDialog();
    }, [closeDialog, directory, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
    const onDelete = useCallback((index: number) => {
        if (!directory) return;
        const item = list[index];
        if (mutate) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { objects: [{ id: item.id, objectType: item.__typename as DeleteType }] },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate({
                            ...directory,
                            childApiVersions: item.__typename === "ApiVersion" ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                            childNoteVersions: item.__typename === "NoteVersion" ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                            childOrganizations: item.__typename === "Organization" ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations,
                            childProjectVersions: item.__typename === "ProjectVersion" ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                            childRoutineVersions: item.__typename === "RoutineVersion" ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                            childSmartContractVersions: item.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions,
                            childStandardVersions: item.__typename === "StandardVersion" ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
                        });
                    }
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions as any[],
                childNoteVersions: item.__typename === "NoteVersion" ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions as any[],
                childOrganizations: item.__typename === "Organization" ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations as any[],
                childProjectVersions: item.__typename === "ProjectVersion" ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions as any[],
                childRoutineVersions: item.__typename === "RoutineVersion" ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions as any[],
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions as any[],
                childStandardVersions: item.__typename === "StandardVersion" ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions as any[],
            });
        }
    }, [deleteMutation, directory, handleUpdate, list, mutate]);

    return (
        <>
            {/* Add item dialog */}
            <FindObjectDialog
                find="List"
                isOpen={isDialogOpen}
                handleCancel={closeDialog}
                handleComplete={onAdd as any}
            />
            {/* Right-click context menu */}
            <DirectoryListItemContextMenu
                canUpdate={canUpdate}
                id={contextId}
                anchorEl={contextAnchor}
                index={selectedIndex ?? -1}
                onClose={closeContext}
                onDelete={onDelete}
                data={selectedItem}
            />
            {/* {title && <Typography component="h2" variant="h5" textAlign="left">{title}</Typography>} */}
            <CardGrid minWidth={120} sx={{

            }}>
                {/* Directory items */}
                {list.map((c: DirectoryItem, index) => (
                    <DirectoryCard
                        canUpdate={canUpdate}
                        key={`directory-item-card-${index}`}
                        index={index}
                        data={c}
                        onContextMenu={openContext}
                        onDelete={onDelete}
                        aria-owns={selectedIndex ? contextId : undefined}
                    />
                )) as any}
                {
                    loading && (
                        <CircularProgress sx={{
                            position: "absolute",
                            top: "50%",
                            left: "50%",
                            transform: "translate(-50%, -50%)",
                            color: palette.mode === "light" ? palette.secondary.light : "white",
                        }} />
                    )
                }
                {/* Add item button */}
                {(canUpdate ? <Tooltip placement="top" title={t("AddItem")}>
                    <DirectoryBox
                        onClick={openDialog}
                        aria-label={t("AddItem")}
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                        }}
                    >
                        <LinkIcon fill={palette.secondary.contrastText} width='56px' height='56px' />
                    </DirectoryBox>
                </Tooltip> : null) as any}
            </CardGrid>
        </>
    );
};
