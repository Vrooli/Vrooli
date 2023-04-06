// Used to display popular/search results of a particular object type
import { Checkbox, ListItem, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { useCallback, useMemo } from 'react';
import { multiLineEllipsis } from 'styles';
import { ReminderListItemProps } from '../types';

/**
 * A list item for a reminder, which can contain sub-items 
 * if the reminder has steps. 
 * 
 * All mutations are handled by the reminder list
 */
export function ReminderListItem({
    handleDelete,
    handleUpdate,
    reminder,
    zIndex,
}: ReminderListItemProps) {
    const { palette } = useTheme();

    // State of the checkbox
    const { checked, checkDisabled, checkTooltip } = useMemo(() => {
        // Can't press checkbox if there is more than one step. 
        // Instead, must check steps individually
        const checkDisabled = reminder.reminderItems.length > 1;
        if (reminder.isComplete) {
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? 'Reminder is complete' : 'Mark as incomplete' };
        } else if (reminder.reminderItems.length > 0 && reminder.reminderItems.every(item => item.isComplete)) {
            handleUpdate({ ...reminder, isComplete: true });
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? 'Reminder is complete' : 'Mark as incomplete' };
        } else {
            return { checked: false, checkDisabled, checkTooltip: checkDisabled ? 'Reminder is incmplete' : 'Mark as complete' };
        }
    }, [handleUpdate, reminder]);
    const handleCheck = useCallback(() => {
        if (checkDisabled) return;
        const updatedItems = reminder.reminderItems.length > 0 ?
            { ...(reminder.reminderItems.map(item => ({ ...item, isComplete: !checked }))) } :
            [];
        handleUpdate({ ...reminder, isComplete: !checked, reminderItems: updatedItems });
    }, [checked, checkDisabled, handleUpdate, reminder]);

    return (
        <ListItem
            disablePadding
            sx={{
                display: 'flex',
                padding: 1,
            }}
        >
            {/* Left informational column */}
            <Stack direction="column" spacing={1} pl={2} sx={{ marginRight: 'auto' }}>
                <ListItemText
                    primary={reminder.name}
                    sx={{ ...multiLineEllipsis(1) }}
                />
            </Stack>
            {/* Right-aligned checkbox */}
            <Stack direction="row" spacing={1}>
                <Tooltip placement={'top'} title={checkTooltip}>
                    <Checkbox
                        id='reminder-checkbox'
                        size="small"
                        name='isComplete'
                        color='secondary'
                        checked={checked}
                        onChange={handleCheck}
                    />
                </Tooltip>
            </Stack>
        </ListItem>
    )
}