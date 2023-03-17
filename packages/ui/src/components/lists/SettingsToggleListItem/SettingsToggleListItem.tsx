import { ListItem, Stack, Switch, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { SettingsToggleListItemProps } from "../types";

export const SettingsToggleListItem = ({
    description,
    disabled,
    name,
    title,
}: SettingsToggleListItemProps) => {
    const { palette } = useTheme();
    const [field, , helpers] = useField(name);

    return (
        <ListItem
            disablePadding
            component="a"
            sx={{
                display: 'flex',
                background: palette.background.paper,
                padding: '8px 16px',
                cursor: 'pointer',
                borderBottom: `1px solid ${palette.divider}`,
            }}
        >
            <Stack direction="column" spacing={0} sx={{ width: '100%' }}>
                <Typography variant="h6" component="div">
                    {title}
                </Typography>
                <Typography variant="body2" component="div">
                    {description}
                </Typography>
            </Stack>
            <Switch
                checked={field.value}
                color="secondary"
                disabled={disabled}
                onChange={(e) => helpers.setValue(e.target.checked)}
            />
        </ListItem>
    )
}