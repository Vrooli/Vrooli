// Used to display popular/search results of a particular object type
import { DeleteIcon, ScheduleIcon } from "@local/shared";
import { Checkbox, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { CompletionBar } from "components/CompletionBar/CompletionBar";
import { ObjectListItemBase } from "components/lists/ObjectListItemBase/ObjectListItemBase";
import { ReminderListItemProps } from "components/lists/types";
import { useCallback, useMemo } from "react";

/**
 * A list item for a reminder, which can contain sub-items 
 * if the reminder has steps. 
 * 
 * All mutations are handled by the reminder list
 */
export function ReminderListItem({
    data,
    onAction,
    ...props
}: ReminderListItemProps) {
    const { palette } = useTheme();
    console.log("in reminder list item", data);

    // State of the checkbox
    const { checked, checkDisabled, checkTooltip } = useMemo(() => {
        if (!data) return { checked: false, checkDisabled: true, checkTooltip: "" };
        // Can't press checkbox if there is more than one step. 
        // Instead, must check steps individually
        const checkDisabled = data.reminderItems.length > 1;
        if (data.isComplete) {
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? "Reminder is complete" : "Mark as incomplete" };
        } else if (data.reminderItems.length > 0 && data.reminderItems.every(item => item.isComplete)) {
            onAction("Update", { ...data, isComplete: true });
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? "Reminder is complete" : "Mark as incomplete" };
        } else {
            return { checked: false, checkDisabled, checkTooltip: checkDisabled ? "Reminder is incmplete" : "Mark as complete" };
        }
    }, [onAction, data]);
    const handleCheck = useCallback(() => {
        if (checkDisabled || !data) return;
        const updatedItems = data.reminderItems.length > 0 ?
            { ...(data.reminderItems.map(item => ({ ...item, isComplete: !checked }))) } :
            [];
        onAction("Update", { ...data, isComplete: !checked, reminderItems: updatedItems });
    }, [checked, checkDisabled, onAction, data]);

    const { stepsComplete, stepsTotal, percentComplete } = useMemo(() => {
        if (!data) return { stepsComplete: 0, stepsTotal: 0, percentComplete: 0 };
        const stepsTotal = data.reminderItems.length;
        const stepsComplete = data.reminderItems.filter(item => item.isComplete).length;
        const percentComplete = stepsTotal > 0 ? Math.round(stepsComplete / stepsTotal * 100) : 0;
        return { stepsComplete, stepsTotal, percentComplete };
    }, [data]);

    const handleDeleteClick = useCallback(() => {
        if (!data?.id) return;
        onAction("Delete", data.id);
    }, [onAction, data?.id]);

    const dueDateIcon = useMemo(() => {
        if (!data?.dueDate) return null;
        const dueDate = new Date(data.dueDate);
        // Check if due Date is today, in the past, or in the future to determine color
        const today = new Date();
        const isToday = dueDate.getDate() === today.getDate() &&
            dueDate.getMonth() === today.getMonth() &&
            dueDate.getFullYear() === today.getFullYear();
        const isPast = dueDate < today;
        const color = isToday ? palette.warning.main : isPast ? palette.error.main : palette.background.textPrimary;
        return <ScheduleIcon fill={color} style={{ margin: "auto", pointerEvents: "none" }} />;
    }, [palette.background.textPrimary, palette.error.main, palette.warning.main, data?.dueDate]);

    return (
        <ObjectListItemBase
            {...props}
            belowSubtitle={
                stepsTotal > 0 ? <CompletionBar
                    color="secondary"
                    value={percentComplete}
                    sx={{ height: "15px" }}
                /> : null
            }
            canNavigate={() => !props.onClick}
            data={data}
            loading={false}
            objectType="Reminder"
            toTheRight={
                <Stack id="list-item-right-stack" direction="row" spacing={1}>
                    {dueDateIcon}
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
