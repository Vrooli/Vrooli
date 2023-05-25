// Used to display popular/search results of a particular object type
import { DeleteIcon } from "@local/shared";
import { Checkbox, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { CompletionBar } from "components/CompletionBar/CompletionBar";
import { ObjectListItemBase } from "components/lists/ObjectListItemBase/ObjectListItemBase";
import { useCallback, useMemo } from "react";
import { ReminderListItemProps } from "../types";

/**
 * A list item for a reminder, which can contain sub-items 
 * if the reminder has steps. 
 * 
 * All mutations are handled by the reminder list
 */
export function ReminderListItem({
    handleDelete,
    handleOpen,
    handleUpdate,
    reminder,
}: ReminderListItemProps) {
    const { palette } = useTheme();

    // State of the checkbox
    const { checked, checkDisabled, checkTooltip } = useMemo(() => {
        // Can't press checkbox if there is more than one step. 
        // Instead, must check steps individually
        const checkDisabled = reminder.reminderItems.length > 1;
        if (reminder.isComplete) {
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? "Reminder is complete" : "Mark as incomplete" };
        } else if (reminder.reminderItems.length > 0 && reminder.reminderItems.every(item => item.isComplete)) {
            handleUpdate({ ...reminder, isComplete: true });
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? "Reminder is complete" : "Mark as incomplete" };
        } else {
            return { checked: false, checkDisabled, checkTooltip: checkDisabled ? "Reminder is incmplete" : "Mark as complete" };
        }
    }, [handleUpdate, reminder]);
    const handleCheck = useCallback(() => {
        console.log("in handle check");
        if (checkDisabled) return;
        const updatedItems = reminder.reminderItems.length > 0 ?
            { ...(reminder.reminderItems.map(item => ({ ...item, isComplete: !checked }))) } :
            [];
        handleUpdate({ ...reminder, isComplete: !checked, reminderItems: updatedItems });
    }, [checked, checkDisabled, handleUpdate, reminder]);

    const { stepsComplete, stepsTotal, percentComplete } = useMemo(() => {
        const stepsTotal = reminder.reminderItems.length;
        const stepsComplete = reminder.reminderItems.filter(item => item.isComplete).length;
        const percentComplete = stepsTotal > 0 ? Math.round(stepsComplete / stepsTotal * 100) : 0;
        return { stepsComplete, stepsTotal, percentComplete };
    }, [reminder]);

    const handleDeleteClick = useCallback(() => {
        console.log("in handle delete click");
        handleDelete(reminder);
    }, [handleDelete, reminder]);

    return (
        <ObjectListItemBase
            belowSubtitle={
                stepsTotal > 0 ? <CompletionBar
                    color="secondary"
                    value={percentComplete}
                    sx={{ height: "15px" }}
                /> : null
            }
            data={reminder}
            loading={false}
            objectType="Reminder"
            onClick={() => { handleOpen(); }}
            toTheRight={
                <Stack direction="row" spacing={1}>
                    <Tooltip placement={"top"} title={checkTooltip}>
                        <Checkbox
                            id='reminder-checkbox'
                            size="small"
                            name='isComplete'
                            color='secondary'
                            checked={checked}
                            onChange={handleCheck}
                        />
                    </Tooltip>
                    {checked && (
                        <Tooltip title="Delete">
                            <IconButton edge="end" size="small" onClick={handleDeleteClick}>
                                <DeleteIcon fill={palette.error.main} />
                            </IconButton>
                        </Tooltip>
                    )}
                </Stack>
            }
            zIndex={201}
        />
    );
}
