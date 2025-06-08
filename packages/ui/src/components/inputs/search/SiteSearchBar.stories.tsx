import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import { useTheme } from "@mui/material";
import { useState } from "react";
import { BasicSearchBar, PaperSearchBar, SiteSearchBarPaper } from "./SiteSearchBar.js";

export default {
    title: "Components/Inputs/Search/SiteSearchBarPaper",
    component: SiteSearchBarPaper,
};

const outerStyle = {
    width: "min(800px, 100%)",
    padding: "20px",
    border: "1px solid #ccc",
} as const;
// Wraps the target component in a div, and displays relevant information underneath it.
function Outer({
    children,
    inputValue,
}: {
    children: React.ReactNode;
    inputValue: string;
}) {
    return (
        <div style={outerStyle}>
            {children}
            <div>
                <p>Input Value: {inputValue}</p>
            </div>
        </div>
    );
}

// Rounded box to put the input value in, to represent how it'd look in a dialog
function MockDialog({ children }: { children: React.ReactNode }) {
    const { palette, spacing } = useTheme();
    return (
        <Box width="min(400px, 100%)" height="400px" borderRadius={spacing(3)} sx={{ backgroundColor: palette.background.paper }}>
            {children}
        </Box>
    );
}

export function Basic() {
    const [value, setValue] = useState("");

    return (
        <Outer inputValue={value}>
            <MockDialog>
                <BasicSearchBar
                    onChange={setValue}
                    placeholder="Search..."
                    value={value}
                />
            </MockDialog>
        </Outer>
    );
}
Basic.parameters = {
    docs: {
        description: {
            story: "Displays the basic search bar.",
        },
    },
};

export function ParentUpdatesBasic() {
    const [value, setValue] = useState("Starting with this text");

    const textOptions = ["hello", "Jeff was here", "I'm a test"];
    function setRandomText() {
        setValue(textOptions[Math.floor(Math.random() * textOptions.length)]);
    }

    return (
        <Outer inputValue={value}>
            <MockDialog>
                <BasicSearchBar
                    onChange={setValue}
                    value={value}
                />
            </MockDialog>
            <Button onClick={setRandomText}>Change Value</Button>
        </Outer>
    );
}
ParentUpdatesBasic.parameters = {
    docs: {
        description: {
            story: "The parent updates the value of the input.",
        },
    },
};

export function Paper() {
    const [value, setValue] = useState("");

    return (
        <Outer inputValue={value}>
            <MockDialog>
                <PaperSearchBar
                    onChange={setValue}
                    value={value}
                />
            </MockDialog>
        </Outer>
    );
}
Paper.parameters = {
    docs: {
        description: {
            story: "Displays the paper search bar.",
        },
    },
};

export function ParentUpdatesPaper() {
    const [value, setValue] = useState("Starting with this text");

    const textOptions = ["hello", "Jeff was here", "I'm a test"];
    function setRandomText() {
        setValue(textOptions[Math.floor(Math.random() * textOptions.length)]);
    }

    return (
        <Outer inputValue={value}>
            <MockDialog>
                <PaperSearchBar
                    onChange={setValue}
                    value={value}
                />
            </MockDialog>
            <Button onClick={setRandomText}>Change Value</Button>
        </Outer>
    );
}
ParentUpdatesPaper.parameters = {
    docs: {
        description: {
            story: "The parent updates the value of the input.",
        },
    },
};
