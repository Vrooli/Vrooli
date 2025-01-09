// Used to display popular/search results of a particular object type
import { endpointsReminder, Reminder, ReminderUpdateInput, shapeReminder } from "@local/shared";
import { Checkbox, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { CompletionBar } from "components/CompletionBar/CompletionBar";
import { ObjectListItemBase } from "components/lists/ObjectListItemBase/ObjectListItemBase";
import { ReminderListItemProps } from "components/lists/types";
import { useObjectActions } from "hooks/objectActions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { DeleteIcon, ScheduleIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useLocation } from "route";

//  // Internal state
//  const [allReminders, setAllReminders] = useState<Reminder[]>(reminders);
//  useEffect(() => {
//      setAllReminders(reminders);
//  }, [reminders]);

//  const handleUpdated = useCallback((index: number, reminder: Reminder) => {
//      const newList = [...allReminders];
//      newList[index] = reminder;
//      setAllReminders(newList);
//      handleUpdate && handleUpdate(newList);
//  }, [allReminders, handleUpdate]);

//  // Handle update mutation
//  const [updateMutation, { errors: updateErrors }] = useLazyFetch<ReminderUpdateInput, Reminder>(endpointPutReminder);
//  const saveUpdate = useCallback((updated: Reminder) => {
//      const index = allReminders.findIndex((reminder) => reminder.id === updated.id);
//      if (index < 0) return;
//      const original = allReminders[index];
//      // Don't wait for the mutation to call handleUpdated
//      handleUpdated(index, updated);
//      // Call the mutation
//      fetchLazyWrapper<ReminderUpdateInput, Reminder>({
//          fetch: updateMutation,
//          inputs: shapeReminder.update(original, updated),
//          successCondition: (data) => !!data.id,
//          successMessage: () => ({ messageKey: "ObjectUpdated", messageVariables: { objectName: updated.name } }),
//      });
//  }, [allReminders, handleUpdated, updateMutation]);

//  const handleDeleted = useCallback((id: string) => {
//      const index = allReminders.findIndex((reminder) => reminder.id === id);
//      if (index < 0) return;
//      const newList = [...allReminders];
//      newList.splice(index, 1);
//      setAllReminders(newList);
//      handleUpdate && handleUpdate(newList);
//  }, [allReminders, handleUpdate]);

const scheduleIconStyle = { margin: "auto", pointerEvents: "none" } as const;

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
    const [, setLocation] = useLocation();

    const [updateMutation, { errors: updateErrors }] = useLazyFetch<ReminderUpdateInput, Reminder>(endpointsReminder.updateOne);

    // State of the checkbox
    const { checked, checkDisabled, checkTooltip } = useMemo(() => {
        if (!data) return { checked: false, checkDisabled: true, checkTooltip: "" };
        // Can't press checkbox if there is more than one step. 
        // Instead, must check steps individually
        const checkDisabled = data.reminderItems?.length > 1;
        if (data.isComplete) {
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? "Reminder is complete" : "Mark as incomplete" };
        } else if (data.reminderItems?.length > 0 && data.reminderItems.every(item => item.isComplete)) {
            onAction("Updated", { ...data, isComplete: true });
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? "Reminder is complete" : "Mark as incomplete" };
        } else {
            return { checked: false, checkDisabled, checkTooltip: checkDisabled ? "Reminder is incmplete" : "Mark as complete" };
        }
    }, [onAction, data]);
    const handleCheck = useCallback(() => {
        console.log("in handle check", onAction, data);
        if (checkDisabled || !data) return;
        const original = data;
        const updatedItems = data.reminderItems.length > 0 ?
            [...(data.reminderItems.map(item => ({ ...item, isComplete: !checked })))] :
            [];
        const updated = { ...data, isComplete: !checked, reminderItems: updatedItems };
        onAction("Updated", updated);
        fetchLazyWrapper<ReminderUpdateInput, Reminder>({
            fetch: updateMutation,
            inputs: shapeReminder.update(original, updated),
            successCondition: (data) => !!data.id,
            onError: () => { onAction("Updated", original); },
        });
    }, [onAction, data, checkDisabled, checked, updateMutation]);

    const { stepsComplete, stepsTotal, percentComplete } = useMemo(() => {
        if (!data) return { stepsComplete: 0, stepsTotal: 0, percentComplete: 0 };
        const stepsTotal = data.reminderItems.length;
        const stepsComplete = data.reminderItems?.filter(item => item.isComplete).length ?? 0;
        const percentComplete = stepsTotal > 0 ? Math.round(stepsComplete / stepsTotal * 100) : 0;
        return { stepsComplete, stepsTotal, percentComplete };
    }, [data]);

    const { onActionStart } = useObjectActions({
        object: data,
        objectType: "Reminder",
        onAction,
        setLocation,
        setObject: (reminder) => onAction("Updated", reminder),
    });

    const handleDeleteClick = useCallback((event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
        if (!data?.id) return;
        event.stopPropagation();
        event.preventDefault();
        onActionStart("Delete");
    }, [data?.id, onActionStart]);

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
        return <ScheduleIcon fill={color} style={scheduleIconStyle} />;
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
            onAction={onAction}
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
        />
    );
}
