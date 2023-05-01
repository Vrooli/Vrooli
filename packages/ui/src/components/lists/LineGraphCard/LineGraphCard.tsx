import { Card, CardContent, Typography, useTheme } from "@mui/material";
import { LineGraph } from "components/graphs/LineGraph/LineGraph";
import { useDimensions } from "utils/hooks/useDimensions";
import { LineGraphCardProps } from "../types";

export const LineGraphCard = ({
    title,
    index,
    ...lineGraphProps
}: LineGraphCardProps) => {
    const { palette } = useTheme();
    const { dimensions, ref } = useDimensions();

    return (
        <Card ref={ref} sx={{
            width: "100%",
            height: "100%",
            boxShadow: 6,
            background: palette.primary.light,
            color: palette.primary.contrastText,
            borderRadius: "16px",
            margin: 0,
        }}>
            <CardContent
                sx={{
                    display: "contents",
                }}
            >
                <Typography
                    variant="h6"
                    component="h2"
                    textAlign="center"
                    sx={{
                        marginBottom: 2,
                        marginTop: 1,
                    }}
                >{title}</Typography>
                <LineGraph
                    dims={{
                        width: dimensions.width,
                        height: 250,
                    }}
                    {...lineGraphProps}
                />
            </CardContent>
        </Card>
    );
};
