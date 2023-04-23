import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Checkbox, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { multiLineEllipsis } from "../../../../styles";
import { CompletionBar } from "../../ObjectListItem/ObjectListItem";
export function ReminderListItem({ handleDelete, handleOpen, handleUpdate, reminder, zIndex, }) {
    const { palette } = useTheme();
    const { checked, checkDisabled, checkTooltip } = useMemo(() => {
        const checkDisabled = reminder.reminderItems.length > 1;
        if (reminder.isComplete) {
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? "Reminder is complete" : "Mark as incomplete" };
        }
        else if (reminder.reminderItems.length > 0 && reminder.reminderItems.every(item => item.isComplete)) {
            handleUpdate({ ...reminder, isComplete: true });
            return { checked: true, checkDisabled, checkTooltip: checkDisabled ? "Reminder is complete" : "Mark as incomplete" };
        }
        else {
            return { checked: false, checkDisabled, checkTooltip: checkDisabled ? "Reminder is incmplete" : "Mark as complete" };
        }
    }, [handleUpdate, reminder]);
    const handleCheck = useCallback((event) => {
        event.stopPropagation();
        event.preventDefault();
        if (checkDisabled)
            return;
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
    return (_jsxs(ListItem, { disablePadding: true, onClick: handleOpen, sx: {
            display: "flex",
            padding: 1,
            cursor: "pointer",
        }, children: [_jsxs(Stack, { direction: "column", spacing: 1, pl: 2, sx: { marginRight: "auto" }, children: [_jsx(ListItemText, { primary: `${reminder.name} ${stepsTotal > 0 ? `(${stepsComplete}/${stepsTotal})` : ""}`, sx: { ...multiLineEllipsis(1) } }), _jsx(ListItemText, { primary: reminder.description, sx: { ...multiLineEllipsis(1), color: palette.background.textSecondary } }), stepsTotal > 0 && _jsx(CompletionBar, { color: "secondary", variant: "determinate", value: percentComplete, sx: { height: "15px" } })] }), _jsx(Stack, { direction: "row", spacing: 1, children: _jsx(Tooltip, { placement: "top", title: checkTooltip, children: _jsx(Checkbox, { id: 'reminder-checkbox', size: "small", name: 'isComplete', color: 'secondary', checked: checked, onChange: handleCheck }) }) })] }));
}
//# sourceMappingURL=ReminderListItem.js.map