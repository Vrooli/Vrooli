import ListItem from "@mui/material/ListItem";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { useField } from "formik";
import { Switch } from "../../inputs/Switch/Switch.js";
import { type SettingsToggleListItemProps } from "../types.js";

export function SettingsToggleListItem({
    description,
    disabled,
    name,
    title,
}: SettingsToggleListItemProps) {
    const { palette } = useTheme();
    const [field] = useField(name);

    return (
        <ListItem
            disablePadding
            component="a"
            sx={{
                display: "flex",
                background: palette.background.paper,
                padding: "8px 16px",
                cursor: "pointer",
                borderBottom: `1px solid ${palette.divider}`,
            }}
        >
            <Stack direction="column" spacing={0} sx={{
                width: "100%",
                color: disabled ? palette.text.disabled : palette.text.primary,
            }}>
                <Typography variant="h6" component="div">
                    {title}
                </Typography>
                <Typography variant="body2" component="div">
                    {description}
                </Typography>
            </Stack>
            <Switch
                checked={field.value}
                disabled={disabled}
                name={name}
                onChange={(checked, event) => field.onChange(event)}
                size="md"
                labelPosition="none"
            />
        </ListItem>
    );
}
