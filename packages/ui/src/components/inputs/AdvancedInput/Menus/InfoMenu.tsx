import Box from "@mui/material/Box";
import Divider from "@mui/material/Divider";
import ListItemText from "@mui/material/ListItemText";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { FormStructureType, noop } from "@vrooli/shared";
import React, { useMemo } from "react";
import { randomString } from "../../../../utils/codes.js";
import { keyComboToString } from "../../../../utils/display/device.js";
import { Switch } from "../../Switch/index.js";
import { FormTip } from "../../form/FormTip.js";
import type { AdvancedInputFeatures } from "../utils.js";
import { dividerStyle } from "../utils/constants.js";

/**
 * InfoMenu Component - renders a menu showing information about the input and customization settings.
 */
interface InfoMenuProps {
    anchorEl: HTMLElement | null;
    enterWillSubmit: boolean;
    mergedFeatures: AdvancedInputFeatures;
    onClose: () => void;
    onToggleEnterWillSubmit: () => void;
    onToggleToolbar: () => void;
    onToggleSpellcheck: () => void;
    showToolbar: boolean;
    spellcheck: boolean;
}

const infoMenuAnchorOrigin = { vertical: "bottom", horizontal: "left" } as const;
const secondaryTypographyProps = {
    style: { whiteSpace: "pre-wrap", fontSize: "0.875rem", color: "#666" },
} as const;
const infoMenuTipData = {
    id: randomString(),
    icon: "Warning",
    isMarkdown: false,
    label: "Some features may be unavailable depending on the AI models you're chatting with and the device this is running on.",
    type: FormStructureType.Tip,
} as const;

export const InfoMenu: React.FC<InfoMenuProps> = React.memo(({
    anchorEl,
    enterWillSubmit,
    mergedFeatures,
    onClose,
    onToggleEnterWillSubmit,
    onToggleToolbar,
    onToggleSpellcheck,
    showToolbar,
    spellcheck,
}: InfoMenuProps) => {
    const theme = useTheme();

    const infoMenuSlotProps = useMemo(() => ({
        paper: {
            style: {
                backgroundColor: theme.palette.background.default,
                borderRadius: "12px",
                maxWidth: "500px",
                maxHeight: "500px",
                overflow: "auto",
                padding: "8px 0",
            },
        },
    }), [theme]);

    return (
        <Menu
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={onClose}
            anchorOrigin={infoMenuAnchorOrigin}
            slotProps={infoMenuSlotProps}
        >
            <Box px={2} pb={1}>
                <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Settings
                </Typography>
            </Box>

            <MenuItem>
                <ListItemText
                    primary="Enter key to submit"
                    secondary={`When enabled, use ${keyComboToString("Enter")} to submit, and ${keyComboToString("Shift", "Enter")} to create a new line.`}
                    secondaryTypographyProps={secondaryTypographyProps}
                />
                <Switch
                    checked={enterWillSubmit}
                    onChange={() => onToggleEnterWillSubmit()}
                    variant="default"
                    size="sm"
                />
            </MenuItem>

            {mergedFeatures.allowFormatting && (
                <MenuItem>
                    <ListItemText
                        primary="Show formatting toolbar"
                        secondary="Display text formatting options above the input area."
                        secondaryTypographyProps={secondaryTypographyProps}
                    />
                    <Switch
                        checked={showToolbar}
                        onChange={() => onToggleToolbar()}
                        variant="default"
                        size="sm"
                    />
                </MenuItem>
            )}
            {mergedFeatures.allowSpellcheck && (
                <MenuItem>
                    <ListItemText
                        primary="Enable spellcheck"
                        secondary="When enabled, the browser will check for spelling errors in the input."
                        secondaryTypographyProps={secondaryTypographyProps}
                    />
                    <Switch
                        checked={spellcheck}
                        onChange={() => onToggleSpellcheck()}
                        variant="default"
                        size="sm"
                    />
                </MenuItem>
            )}
            <Divider sx={dividerStyle} />
            <Box px={2}>
                <FormTip
                    element={infoMenuTipData}
                    isEditing={false}
                    onUpdate={noop}
                    onDelete={noop}
                />
            </Box>
        </Menu>
    );
});
InfoMenu.displayName = "InfoMenu";
