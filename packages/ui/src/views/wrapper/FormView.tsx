import { Box, Container, useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { ViewDisplayType } from "views/types";

interface Props {
    autocomplete?: string;
    children: JSX.Element;
    display?: ViewDisplayType;
    maxWidth?: string | number;
    title: string;
}

export const FormView = ({
    autocomplete = "on",
    children,
    display = "page",
    maxWidth = "90%",
    title,
}: Props) => {
    const { palette } = useTheme();

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    title,
                }}
            />
            <Box sx={{
                backgroundColor: palette.background.paper,
                display: "grid",
                position: "relative",
                boxShadow: "0px 2px 4px -1px rgb(0 0 0 / 20%), 0px 4px 5px 0px rgb(0 0 0 / 14%), 0px 1px 10px 0px rgb(0 0 0 / 12%)",
                minWidth: "300px",
                maxWidth: "min(100%, 700px)",
                borderRadius: "10px",
                overflow: "hidden",
                left: "50%",
                transform: "translateX(-50%)",
                marginBottom: "20px",
            }}>
                <Container>
                    {children}
                </Container>
            </Box>
        </>
    );
};
