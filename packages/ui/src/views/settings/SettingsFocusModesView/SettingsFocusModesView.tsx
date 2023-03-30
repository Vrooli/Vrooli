import { Box, IconButton, ListItem, ListItemText, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { DeleteOneInput, DeleteType, FocusMode, FocusModeStopCondition, Success } from '@shared/consts';
import { AddIcon, DeleteIcon, EditIcon } from "@shared/icons";
import { mutationWrapper } from "api";
import { deleteOneOrManyDeleteOne } from "api/generated/endpoints/deleteOneOrMany_deleteOne";
import { useCustomMutation } from "api/hooks";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { FocusModeDialog } from "components/dialogs/FocusModeDialog/FocusModeDialog";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { t } from "i18next";
import { useCallback, useContext, useEffect, useState } from "react";
import { multiLineEllipsis } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { SettingsFocusModesViewProps } from "../types";

export const SettingsFocusModesView = ({
    display = 'page',
}: SettingsFocusModesViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();

    const [focusModes, setFocusModes] = useState<FocusMode[]>([]);
    useEffect(() => {
        if (session) {
            setFocusModes(getCurrentUser(session).focusModes ?? []);
        }
    }, [session]);

    // Handle delete
    const [deleteMutation] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
    const handleDelete = useCallback((focusMode: FocusMode) => {
        // Don't delete if there is only one focus mode left
        if (focusModes.length === 1) {
            PubSub.get().publishSnack({ messageKey: 'MustHaveFocusMode', severity: 'Error' });
            return;
        }
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            messageKey: 'DeleteFocusModeConfirm',
            buttons: [
                {
                    labelKey: 'Yes', onClick: () => {
                        mutationWrapper<Success, DeleteOneInput>({
                            mutation: deleteMutation,
                            input: { id: focusMode.id, objectType: DeleteType.FocusMode },
                            successCondition: (data) => data.success,
                            successMessage: () => ({ key: 'FocusModeDeleted' }),
                            onSuccess: () => {
                                setFocusModes((prevFocusModes) => prevFocusModes.filter((prevFocusMode) => prevFocusMode.id !== focusMode.id));
                            },
                        })
                    }
                },
                { labelKey: 'Cancel' },
            ]
        });
    }, [deleteMutation, focusModes.length]);

    // // Handle filters
    // const [filters, setFilters] = useState<FocusModeFilterShape[]>([]);
    // const handleFiltersUpdate = useCallback((updatedList: FocusModeFilterShape[], filterType: FocusModeFilterType) => {
    //     // Hidden tags are wrapped in a shape that includes an isBlur flag. 
    //     // Because of this, we must loop through the updatedList to see which tags have been added or removed.
    //     // const updatedFilters = updatedList.map((tag) => {
    //     //     const existingTag = filters.find((filter) => filter.tag.id === tag.id);
    //     //     return existingTag ?? {
    //     //         id: uuid(),
    //     //         filterType,
    //     //         tag,
    //     //     }
    //     // });
    //     // setFilters(updatedFilters);
    // }, [filters]);
    // const { blurs, hides, showMores } = useMemo(() => {
    //     const blurs: TagShape[] = [];
    //     const hides: TagShape[] = [];
    //     const showMores: TagShape[] = [];
    //     filters.forEach((filter) => {
    //         if (filter.filterType === FocusModeFilterType.Blur) {
    //             blurs.push(filter.tag);
    //         } else if (filter.filterType === FocusModeFilterType.Hide) {
    //             hides.push(filter.tag);
    //         } else if (filter.filterType === FocusModeFilterType.ShowMore) {
    //             showMores.push(filter.tag);
    //         }
    //     });
    //     return { blurs, hides, showMores };
    // }, [filters]);

    // Handle dialog for adding new focus mode
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const [editingFocusMode, setEditingFocusMode] = useState<FocusMode | null>(null);
    const handleAdd = () => {
        setIsDialogOpen(true);
    };
    const handleUpdate = (focusMode: FocusMode) => {
        setEditingFocusMode(focusMode);
        setIsDialogOpen(true);
    };
    const handleCloseDialog = () => {
        setIsDialogOpen(false);
    };
    const handleCreated = (newFocusMode: FocusMode) => {
        setIsDialogOpen(false);
        PubSub.get().publishFocusMode({
            __typename: 'ActiveFocusMode' as const,
            mode: newFocusMode,
            stopCondition: FocusModeStopCondition.Automatic,
        })
    }

    return (
        <>
            {/* Dialog to create a new focus mode */}
            <FocusModeDialog
                isCreate={editingFocusMode === null}
                isOpen={isDialogOpen}
                onClose={handleCloseDialog}
                onCreated={handleCreated}
                onUpdated={() => { }}
                zIndex={1000}
            />
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'FocusMode',
                    titleVariables: { count: 2 },
                }}
            />
            <Stack direction="row">
                <SettingsList />
                <Stack direction="column" sx={{
                    margin: 'auto',
                    display: 'block',
                }}>
                    <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                        <Typography component="h2" variant="h4">{t('FocusMode', { count: 2 })}</Typography>
                        <Tooltip title="Add new" placement="top">
                            <IconButton
                                size="medium"
                                onClick={handleAdd}
                                sx={{
                                    padding: 1,
                                }}
                            >
                                <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    <ListContainer
                        emptyText={t(`NoFocusModes`, { ns: 'error' })}
                        isEmpty={focusModes.length === 0}
                        sx={{ maxWidth: '500px' }}
                    >
                        {focusModes.map((focusMode) => (
                            <ListItem key={focusMode.id}>
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    pl={2}
                                    sx={{
                                        width: '-webkit-fill-available',
                                        display: 'grid',
                                        pointerEvents: 'none',
                                    }}
                                >
                                    {/* Name */}
                                    <ListItemText
                                        primary={focusMode.name}
                                        sx={{
                                            ...multiLineEllipsis(1),
                                            lineBreak: 'anywhere',
                                            pointerEvents: 'none',
                                        }}
                                    />
                                    {/* Description */}
                                    <ListItemText
                                        primary={focusMode.description}
                                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: 'none' }}
                                    />
                                </Stack>
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    sx={{
                                        pointerEvents: 'none',
                                        justifyContent: 'center',
                                        alignItems: 'start',
                                    }}
                                >
                                    {/* Edit */}
                                    <Box
                                        component="a"
                                        onClick={() => handleUpdate(focusMode)}
                                        sx={{
                                            display: 'flex',
                                            justifyContent: 'center',
                                            alignItems: 'center',
                                            cursor: 'pointer',
                                            pointerEvents: 'all',
                                            paddingBottom: '4px',
                                        }}>
                                        <EditIcon fill={palette.secondary.main} />
                                    </Box>
                                    {/* Delete */}
                                    <Box
                                        component="a"
                                        onClick={() => handleDelete(focusMode)}
                                        sx={{
                                            display: 'flex',
                                            justifyContent: 'center',
                                            alignItems: 'center',
                                            cursor: 'pointer',
                                            pointerEvents: 'all',
                                            paddingBottom: '4px',
                                        }}>
                                        <DeleteIcon fill={palette.secondary.main} />
                                    </Box>
                                </Stack>
                            </ListItem>
                        ))}
                    </ListContainer>
                </Stack>
            </Stack>
        </>
    )
}