// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { CommonKey, Count, DeleteManyInput, endpointPostDeleteMany, Resource, ResourceUsedFor } from "@local/shared";
import { Box, Stack, styled, Tooltip, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { ResourceDialog } from "components/dialogs/ResourceDialog/ResourceDialog";
import { cardRoot } from "components/lists/styles";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { DeleteIcon, EditIcon, LinkIcon } from "icons";
import { forwardRef, useCallback, useContext, useMemo, useState } from "react";
import { DragDropContext, Draggable, Droppable, DropResult } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { multiLineEllipsis, noSelect } from "styles";
import { getResourceIcon } from "utils/display/getResourceIcon";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import usePress from "utils/hooks/usePress";
import { getResourceType, getResourceUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { updateArray } from "utils/shape/general";
import { ResourceListItemContextMenu } from "../ResourceListItemContextMenu/ResourceListItemContextMenu";
import { ResourceCardProps, ResourceListHorizontalProps } from "../types";

const ResourceBox = styled(Box)(({ theme }) => ({
    ...noSelect,
    boxShadow: theme.spacing(4),
    background: theme.palette.primary.light,
    color: theme.palette.secondary.contrastText,
    borderRadius: "16px",
    margin: theme.spacing(0),
    padding: theme.spacing(1),
    cursor: "pointer",
    width: "120px",
    minWidth: "120px",
    minHeight: "120px",
    height: "120px",
    position: "relative",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
}));

const ResourceCard = forwardRef<any, ResourceCardProps>(({
    canUpdate,
    data,
    dragProps,
    dragHandleProps,
    index,
    onContextMenu,
    onEdit,
    onDelete,
}, ref) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [showIcons, setShowIcons] = useState(false);

    const { title, subtitle } = useMemo(() => {
        const { title, subtitle } = getDisplay(data, getUserLanguages(session));
        return {
            title: title ? title : t((data.usedFor ?? "Context") as CommonKey),
            subtitle,
        };
    }, [data, session, t]);

    const Icon = useMemo(() => {
        return getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link);
    }, [data]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback((target: EventTarget) => {
        // Check if edit or delete button was clicked
        const targetId: string | undefined = target.id;
        if (targetId && targetId.startsWith("edit-")) {
            onEdit?.(index);
        }
        else if (targetId && targetId.startsWith("delete-")) {
            onDelete?.(index);
        }
        else {
            // If no resource type or link, show error
            const resourceType = getResourceType(data.link);
            if (!resourceType || !href) {
                PubSub.get().publishSnack({ messageKey: "CannotOpenLink", severity: "Error" });
                return;
            }
            // Open link
            else openLink(setLocation, href);
        }
    }, [data.link, href, index, onDelete, onEdit, setLocation]);
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
        <Tooltip placement="top" title={`${subtitle ? subtitle + " - " : ""}${data.link}`}>
            <ResourceBox
                ref={ref}
                {...dragProps}
                {...dragHandleProps}
                {...pressEvents}
                component="a"
                href={href}
                onClick={(e) => e.preventDefault()}
            >
                {/* Edit and delete icons, only visible on hover */}
                {showIcons && (
                    <>
                        <Tooltip title={t("Edit")}>
                            <ColorIconButton
                                id='edit-icon-button'
                                background='#c5ab17'
                                sx={{ position: "absolute", top: 4, left: 4 }}
                            >
                                <EditIcon id='edit-icon' fill={palette.secondary.contrastText} />
                            </ColorIconButton>
                        </Tooltip>
                        <Tooltip title={t("Delete")}>
                            <ColorIconButton
                                id='delete-icon-button'
                                background={palette.error.main}
                                sx={{ position: "absolute", top: 4, right: 4 }}
                            >
                                <DeleteIcon id='delete-icon' fill={palette.secondary.contrastText} />
                            </ColorIconButton>
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
                    <Icon sx={{ fill: "white" }} />
                    <Typography
                        gutterBottom
                        variant="body2"
                        component="h3"
                        sx={{
                            ...multiLineEllipsis(3),
                            textAlign: "center",
                            lineBreak: title ? "auto" : "anywhere", // Line break anywhere only if showing link
                        }}
                    >
                        {firstString(title, data.link)}
                    </Typography>
                </Stack>
            </ResourceBox>
        </Tooltip>
    );
});

const LoadingCard = () => {
    return (
        <ResourceBox>
            <Stack
                direction="column"
                justifyContent="center"
                alignItems="center"
                sx={{
                    height: "100%",
                    overflow: "hidden",
                }}
            >
                <TextLoading size="subheader" lines={2} sx={{ width: "70%", opacity: "0.5" }} />
            </Stack>
        </ResourceBox>
    );
};

export const ResourceListHorizontal = ({
    canUpdate = true,
    handleUpdate,
    id,
    list,
    loading = false,
    mutate = true,
    title,
    zIndex,
}: ResourceListHorizontalProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const onAdd = useCallback((newResource: Resource) => {
        if (!list) return;
        if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: [...(list?.resources as any) ?? [], newResource],
            });
        }
    }, [handleUpdate, list]);

    const onUpdate = useCallback((index: number, updatedResource: Resource) => {
        if (!list) return;
        if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, index, updatedResource) as any[],
            });
        }
    }, [handleUpdate, list]);

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!destination) return;
        if (source.index === destination.index) return;
        // Handle the reordering of the resources in the list
        if (handleUpdate && list) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, source.index, list.resources[destination.index]) as any[],
            });
        }
    }, [handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
    const onDelete = useCallback((index: number) => {
        if (!list) return;
        const resource = list.resources[index];
        if (mutate && resource.id) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { ids: [resource.id], objectType: "Resource" as any },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate({
                            ...list,
                            resources: list.resources.filter(r => r.id !== resource.id) as any,
                        });
                    }
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: list.resources.filter(r => r.id !== resource.id) as any,
            });
        }
    }, [deleteMutation, handleUpdate, list, mutate]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selectedResource, setSelectedResource] = useState<Resource | null>(null);
    const selectedIndex = useMemo(() => selectedResource ? list?.resources?.findIndex(r => r.id === selectedResource.id) : -1, [list, selectedResource]);
    const contextId = useMemo(() => `resource-context-menu-${selectedResource?.id}`, [selectedResource]);
    const openContext = useCallback((target: EventTarget, index: number) => {
        setContextAnchor(target);
        const resource = list?.resources[index];
        setSelectedResource(resource as any);
    }, [list?.resources]);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelectedResource(null);
    }, []);

    // Add/update resource dialog
    const [editingIndex, setEditingIndex] = useState<number>(-1);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((index: number) => {
        setEditingIndex(index);
        setIsDialogOpen(true);
    }, []);

    const dialog = useMemo(() => (
        list ? <ResourceDialog
            partialData={editingIndex >= 0 ? list.resources[editingIndex as number] : undefined}
            index={editingIndex}
            isOpen={isDialogOpen}
            listId={list.id}
            onClose={closeDialog}
            onCreated={onAdd}
            onUpdated={onUpdate}
            mutate={mutate}
            zIndex={zIndex + 1}
        /> : null
    ), [list, editingIndex, isDialogOpen, closeDialog, onAdd, onUpdate, mutate, zIndex]);

    return (
        <>
            {/* Add resource dialog */}
            {dialog}
            {/* Right-click context menu */}
            <ResourceListItemContextMenu
                canUpdate={canUpdate}
                id={contextId}
                anchorEl={contextAnchor}
                index={selectedIndex ?? -1}
                onClose={closeContext}
                onAddBefore={() => {
                    setEditingIndex(selectedIndex ?? 0);
                    openDialog();
                }}
                onAddAfter={() => {
                    setEditingIndex(selectedIndex ? selectedIndex + 1 : 0);
                    openDialog();
                }}
                onDelete={onDelete}
                onEdit={() => openUpdateDialog(selectedIndex ?? 0)}
                onMove={(index: number) => {
                    if (handleUpdate && list) {
                        handleUpdate({
                            ...list,
                            resources: updateArray(list.resources, selectedIndex ?? 0, list.resources[index]) as any[],
                        });
                    }
                }}
                resource={selectedResource}
                zIndex={zIndex + 1}
            />
            {title && <Typography component="h2" variant="h5" textAlign="left">{title}</Typography>}
            <DragDropContext onDragEnd={onDragEnd}>
                <Droppable droppableId="resource-list" direction="horizontal">
                    {(provided) => (
                        <Stack
                            ref={provided.innerRef}
                            id={id}
                            {...provided.droppableProps}
                            direction="row"
                            justifyContent="center"
                            alignItems="center"
                            spacing={2}
                            p={1}
                            sx={{
                                width: "100%",
                                maxWidth: "700px",
                                marginLeft: "auto",
                                marginRight: "auto",
                                // Custom scrollbar styling
                                overflowX: "auto",
                                "&::-webkit-scrollbar": {
                                    width: 5,
                                },
                                "&::-webkit-scrollbar-track": {
                                    backgroundColor: "transparent",
                                },
                                "&::-webkit-scrollbar-thumb": {
                                    borderRadius: "100px",
                                    backgroundColor: "#409590",
                                },
                            }}>
                            {/* Resources */}
                            {list?.resources?.map((c: Resource, index) => (
                                <Draggable key={`resource-card-${index}`} draggableId={`resource-card-${index}`} index={index}>
                                    {(provided) => (
                                        <ResourceCard
                                            ref={provided.innerRef}
                                            dragProps={provided.draggableProps}
                                            dragHandleProps={provided.dragHandleProps}
                                            canUpdate={canUpdate}
                                            key={`resource-card-${index}`}
                                            index={index}
                                            data={c}
                                            onContextMenu={openContext}
                                            onEdit={openUpdateDialog}
                                            onDelete={onDelete}
                                            aria-owns={selectedIndex ? contextId : undefined}
                                        />
                                    )}
                                </Draggable>
                            ))}
                            {/* Dummy resources when loading */}
                            {
                                loading && !Array.isArray(list?.resources) && Array.from(Array(3).keys()).map((i) => (
                                    <LoadingCard key={`resource-card-${i}`} />
                                ))
                            }
                            {/* Add resource button */}
                            {canUpdate ? <Tooltip placement="top" title={t("CreateResource")}>
                                <Box
                                    onClick={openDialog}
                                    aria-label={t("CreateResource")}
                                    sx={{
                                        ...cardRoot,
                                        background: palette.primary.light,
                                        width: "120px",
                                        minWidth: "120px",
                                        height: "120px",
                                        minHeight: "120px",
                                        display: "flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                    }}
                                >
                                    <LinkIcon fill={palette.secondary.contrastText} width='56px' height='56px' />
                                </Box>
                            </Tooltip> : null}
                            {provided.placeholder}
                        </Stack>
                    )}
                </Droppable>
            </DragDropContext>
        </>
    );
};
