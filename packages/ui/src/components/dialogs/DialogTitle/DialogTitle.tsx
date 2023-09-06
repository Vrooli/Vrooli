import { Box, DialogTitle as MuiDialogTitle, IconButton, useTheme } from "@mui/material";
import { Title } from "components/text/Title/Title";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { CloseIcon } from "icons";
import { forwardRef } from "react";
import { useLocation } from "route";
import { noSelect } from "styles";
import { tryOnClose } from "utils/navigation/urlTools";
import { DialogTitleProps } from "../types";

export const DialogTitle = forwardRef(({
    below,
    id,
    onClose,
    startComponent,
    ...titleData
}: DialogTitleProps, ref) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const isLeftHanded = useIsLeftHanded();

    return (
        <Box ref={ref} sx={{
            background: palette.primary.dark,
            color: palette.primary.contrastText,
            position: "sticky",
            top: 0,
            zIndex: 2,
            ...titleData.sxs?.root,
        }}>
            <MuiDialogTitle
                id={id}
                sx={{
                    ...noSelect,
                    display: "flex",
                    flexDirection: isLeftHanded ? "row-reverse" : "row",
                    alignItems: "center",
                    padding: 2,
                    textAlign: "center",
                    fontSize: { xs: "1.5rem", sm: "2rem" },
                }}
            >
                {startComponent}
                <Title
                    variant="subheader"
                    {...titleData}
                    sxs={{
                        stack: {
                            ...(isLeftHanded ? { marginRight: "auto" } : { marginLeft: "auto" }),
                            padding: 0,
                        },
                    }}
                />
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={() => { tryOnClose(onClose, setLocation); }}
                    sx={isLeftHanded ? { marginRight: "auto" } : { marginLeft: "auto" }}
                >
                    <CloseIcon fill={palette.primary.contrastText} />
                </IconButton>
            </MuiDialogTitle>
            {below}
        </Box>
    );
});
