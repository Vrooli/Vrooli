import { Button, styled } from "@mui/material";
import { IconCommon } from "../../../icons/Icons.js";

const StyledButton = styled(Button)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    display: "flex",
    marginTop: theme.spacing(2),
    textAlign: "center",
    textTransform: "none",
}));

export function SearchExistingButton({
    href,
    text,
}: {
    href: string;
    text: string
}) {
    return (
        <StyledButton
            href={href}
            variant="text"
            endIcon={<IconCommon
                decorative
                name="Search"
            />}
        >
            {text}
        </StyledButton>
    );
}
