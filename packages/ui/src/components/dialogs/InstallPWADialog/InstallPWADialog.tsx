import { useTheme } from "@mui/material";
import Box from "@mui/material/Box";
import Dialog from "@mui/material/Dialog";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import Tab from "@mui/material/Tab";
import Tabs from "@mui/material/Tabs";
import Typography from "@mui/material/Typography";
import { useState } from "react";
import { IconCommon, IconService } from "../../../icons/Icons.js";

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

function TabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;

    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`install-tabpanel-${index}`}
            aria-labelledby={`install-tab-${index}`}
            {...other}
        >
            {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
        </div>
    );
}

interface InstallPWADialogProps {
    open: boolean;
    onClose: () => void;
}

export function InstallPWADialog({ open, onClose }: InstallPWADialogProps) {
    const { palette } = useTheme();
    const [tabValue, setTabValue] = useState(0);

    const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
        setTabValue(newValue);
    };

    const stepStyle = {
        mb: 2,
        p: 2,
        borderRadius: 2,
        background: palette.background.paper,
        border: `1px solid ${palette.divider}`,
    };

    const iconStyle = {
        verticalAlign: "middle",
        display: "inline-flex",
        mx: 0.5,
    };

    return (
        <Dialog
            open={open}
            onClose={onClose}
            maxWidth="sm"
            fullWidth
            PaperProps={{
                sx: {
                    borderRadius: 3,
                    background: palette.background.default,
                },
            }}
        >
            <DialogTitle sx={{ 
                display: "flex", 
                alignItems: "center", 
                justifyContent: "space-between",
                borderBottom: `1px solid ${palette.divider}`,
                pb: 2,
            }}>
                <Typography variant="h5" component="h2" fontWeight="bold">
                    Install Vrooli App
                </Typography>
                <IconButton onClick={onClose} size="small">
                    <IconCommon name="X" />
                </IconButton>
            </DialogTitle>
            <DialogContent sx={{ pt: 3 }}>
                <Typography variant="body1" sx={{ mb: 1, mt: 2, color: palette.text.secondary }}>
                    Add Vrooli to your home screen for the best experience. It works just like a native app!
                </Typography>
                <Typography variant="body2" sx={{ mb: 3, color: palette.text.secondary, fontStyle: "italic" }}>
                    📱 Native apps coming soon to the App Store and Google Play Store
                </Typography>

                <Tabs 
                    value={tabValue} 
                    onChange={handleTabChange}
                    variant="fullWidth"
                    sx={{
                        borderBottom: `1px solid ${palette.divider}`,
                        "& .MuiTab-root": {
                            textTransform: "none",
                            fontSize: "1rem",
                        },
                    }}
                >
                    <Tab 
                        label="iPhone/iPad" 
                        icon={<IconService name="Apple" size={20} />}
                        iconPosition="start"
                    />
                    <Tab 
                        label="Android" 
                        icon={<IconService name="Android" size={20} />}
                        iconPosition="start"
                    />
                    <Tab 
                        label="Desktop" 
                        icon={<IconCommon name="Laptop" size={20} />}
                        iconPosition="start"
                    />
                </Tabs>

                <TabPanel value={tabValue} index={0}>
                    <Stack spacing={2}>
                        <Box sx={stepStyle}>
                            <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
                                <Box sx={{ ...iconStyle, color: palette.primary.main }}>1.</Box>
                                Open in Safari
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Make sure you're using Safari browser (not Chrome or other browsers).
                            </Typography>
                        </Box>

                        <Box sx={stepStyle}>
                            <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
                                <Box sx={{ ...iconStyle, color: palette.primary.main }}>2.</Box>
                                Tap the Share Button
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Look for the 
                                <IconCommon name="Share" size={16} sx={iconStyle} />
                                share icon at the bottom of your screen (iPhone) or top (iPad).
                            </Typography>
                        </Box>

                        <Box sx={stepStyle}>
                            <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
                                <Box sx={{ ...iconStyle, color: palette.primary.main }}>3.</Box>
                                Add to Home Screen
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Scroll down and tap "Add to Home Screen" 
                                <IconCommon name="Plus" size={16} sx={iconStyle} />
                                then tap "Add" in the top right.
                            </Typography>
                        </Box>

                        <Typography variant="caption" sx={{ 
                            mt: 2, 
                            p: 2, 
                            borderRadius: 2,
                            background: palette.info.main + "20",
                            color: palette.info.main,
                            display: "block",
                        }}>
                            💡 Tip: The app icon will appear on your home screen just like any other app!
                        </Typography>
                    </Stack>
                </TabPanel>

                <TabPanel value={tabValue} index={1}>
                    <Stack spacing={2}>
                        <Box sx={stepStyle}>
                            <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
                                <Box sx={{ ...iconStyle, color: palette.primary.main }}>1.</Box>
                                Open in Chrome
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Make sure you're using Chrome browser for the best experience.
                            </Typography>
                        </Box>

                        <Box sx={stepStyle}>
                            <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
                                <Box sx={{ ...iconStyle, color: palette.primary.main }}>2.</Box>
                                Tap the Menu
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Tap the three dots 
                                <IconCommon name="MoreVert" size={16} sx={iconStyle} />
                                in the top right corner.
                            </Typography>
                        </Box>

                        <Box sx={stepStyle}>
                            <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
                                <Box sx={{ ...iconStyle, color: palette.primary.main }}>3.</Box>
                                Add to Home Screen
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Select "Add to Home screen" or "Install app", then tap "Add" or "Install".
                            </Typography>
                        </Box>

                        <Typography variant="caption" sx={{ 
                            mt: 2, 
                            p: 2, 
                            borderRadius: 2,
                            background: palette.success.main + "20",
                            color: palette.success.main,
                            display: "block",
                        }}>
                            ✓ You may see an "Install" banner at the bottom of your screen - just tap it!
                        </Typography>
                    </Stack>
                </TabPanel>

                <TabPanel value={tabValue} index={2}>
                    <Stack spacing={2}>
                        <Typography variant="h6" gutterBottom>
                            Chrome / Edge
                        </Typography>
                        
                        <Box sx={stepStyle}>
                            <Typography variant="body1" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
                                <Box sx={{ ...iconStyle, color: palette.primary.main }}>1.</Box>
                                Look for the Install Icon
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Check the right side of your address bar for an install icon 
                                <IconCommon name="Download" size={16} sx={iconStyle} />
                                or 
                                <IconCommon name="Plus" size={16} sx={iconStyle} />
                            </Typography>
                        </Box>

                        <Box sx={stepStyle}>
                            <Typography variant="body1" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
                                <Box sx={{ ...iconStyle, color: palette.primary.main }}>2.</Box>
                                Click Install
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Click the icon and select "Install" when prompted. The app will open in its own window.
                            </Typography>
                        </Box>

                        <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>
                            Alternative Method
                        </Typography>

                        <Box sx={stepStyle}>
                            <Typography variant="body2" color="text.secondary">
                                <strong>Chrome:</strong> Click the three dots menu → "Save and share" → "Install Vrooli..."
                            </Typography>
                            <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                                <strong>Edge:</strong> Click the three dots menu → "Apps" → "Install this site as an app"
                            </Typography>
                        </Box>
                    </Stack>
                </TabPanel>
            </DialogContent>
        </Dialog>
    );
}