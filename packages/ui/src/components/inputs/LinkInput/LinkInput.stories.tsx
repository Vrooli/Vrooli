import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import RadioGroup from "@mui/material/RadioGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Radio from "@mui/material/Radio";
import TextField from "@mui/material/TextField";
import Chip from "@mui/material/Chip";
import { Formik, Form } from "formik";
import { LinkInput } from "./LinkInput.js";
import { Switch } from "../Switch/Switch.js";
import type { FindObjectType } from "../../dialogs/types.js";

const meta: Meta<typeof LinkInput> = {
    title: "Components/Inputs/LinkInput",
    component: LinkInput,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive LinkInput Playground
export const LinkInputShowcase: Story = {
    render: () => {
        const [autoFocus, setAutoFocus] = useState(false);
        const [fullWidth, setFullWidth] = useState(true);
        const [disabled, setDisabled] = useState(false);
        const [label, setLabel] = useState("Link");
        const [placeholder, setPlaceholder] = useState("https://example.com");
        const [limitToEnabled, setLimitToEnabled] = useState(false);

        const availableTypes: FindObjectType[] = [
            "Api", "ApiVersion", "Bookmark", "BookmarkList", "Chat", "ChatInvite", "ChatMessage", 
            "ChatParticipant", "Comment", "Issue", "Meeting", "MeetingInvite", "Member", "MemberInvite", 
            "Notification", "Project", "ProjectVersion", "PullRequest", "Question", "QuestionAnswer", 
            "Quiz", "QuizAttempt", "QuizQuestion", "QuizQuestionResponse", "Reaction", "Report", 
            "Resource", "ResourceVersion", "Routine", "RoutineVersion", "Run", "RunProject", 
            "RunRoutine", "Schedule", "Standard", "StandardVersion", "Tag", "Team", "User",
        ];
        
        const [selectedTypes, setSelectedTypes] = useState<FindObjectType[]>(["Project", "Routine", "Standard"]);

        const handleTypeToggle = (type: FindObjectType) => {
            setSelectedTypes(prev => 
                prev.includes(type) 
                    ? prev.filter(t => t !== type)
                    : [...prev, type],
            );
        };

        return (
            <Box sx={{ 
                p: 2, 
                height: "100vh", 
                overflow: "auto",
                bgcolor: "background.default", 
            }}>
                <Box sx={{ 
                    display: "flex", 
                    gap: 2, 
                    flexDirection: "column",
                    maxWidth: 1400, 
                    mx: "auto", 
                }}>
                    {/* Controls Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        height: "fit-content",
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>LinkInput Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Label Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label</FormLabel>
                                <TextField
                                    value={label}
                                    onChange={(e) => setLabel(e.target.value)}
                                    size="small"
                                    placeholder="Enter label text..."
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Placeholder Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Placeholder</FormLabel>
                                <TextField
                                    value={placeholder}
                                    onChange={(e) => setPlaceholder(e.target.value)}
                                    size="small"
                                    placeholder="Enter placeholder text..."
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* AutoFocus Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={autoFocus}
                                    onChange={(checked) => setAutoFocus(checked)}
                                    size="sm"
                                    label="Auto Focus"
                                    labelPosition="right"
                                />
                            </FormControl>

                            {/* Full Width Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={fullWidth}
                                    onChange={(checked) => setFullWidth(checked)}
                                    size="sm"
                                    label="Full Width"
                                    labelPosition="right"
                                />
                            </FormControl>

                            {/* Disabled Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={disabled}
                                    onChange={(checked) => setDisabled(checked)}
                                    size="sm"
                                    label="Disabled"
                                    labelPosition="right"
                                />
                            </FormControl>

                            {/* Limit To Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={limitToEnabled}
                                    onChange={(checked) => setLimitToEnabled(checked)}
                                    size="sm"
                                    label="Limit Object Types"
                                    labelPosition="right"
                                />
                            </FormControl>
                        </Box>

                        {/* Object Types Selection (when limitTo is enabled) */}
                        {limitToEnabled && (
                            <Box sx={{ mt: 3 }}>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>
                                    Allowed Object Types (click to toggle):
                                </Typography>
                                <Box sx={{ 
                                    display: "flex", 
                                    flexWrap: "wrap", 
                                    gap: 1, 
                                    maxHeight: 200,
                                    overflow: "auto",
                                }}>
                                    {availableTypes.map(type => (
                                        <Chip
                                            key={type}
                                            label={type}
                                            onClick={() => handleTypeToggle(type)}
                                            color={selectedTypes.includes(type) ? "primary" : "default"}
                                            variant={selectedTypes.includes(type) ? "filled" : "outlined"}
                                            size="small"
                                        />
                                    ))}
                                </Box>
                            </Box>
                        )}
                    </Box>

                    {/* Live Demo */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Live Demo</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            Enter a URL or click the search button to find objects within the application. 
                            {limitToEnabled && ` Search is limited to: ${selectedTypes.join(", ")}.`}
                        </Typography>
                        
                        <Formik
                            initialValues={{
                                link: "",
                            }}
                            onSubmit={(values) => {
                                console.log("Form submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ maxWidth: 600 }}>
                                    <LinkInput
                                        name="link"
                                        label={label}
                                        placeholder={placeholder}
                                        autoFocus={autoFocus}
                                        fullWidth={fullWidth}
                                        disabled={disabled}
                                        limitTo={limitToEnabled ? selectedTypes : undefined}
                                        onObjectData={(data) => {
                                            console.log("Object data received:", data);
                                        }}
                                    />
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Link Types Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Different Link Types</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            The LinkInput component handles both external URLs and internal application links.
                        </Typography>
                        
                        <Formik
                            initialValues={{
                                externalLink: "https://github.com",
                                youtubeLink: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
                                internalLink: "",
                                emailLink: "mailto:contact@example.com",
                                telLink: "tel:+1234567890",
                            }}
                            onSubmit={(values) => {
                                console.log("Link types submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
                                    gap: 3, 
                                }}>
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            External Website
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Links to external websites
                                        </Typography>
                                        <LinkInput
                                            name="externalLink"
                                            label="External Link"
                                            placeholder="https://example.com"
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            YouTube Video
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Direct links to videos
                                        </Typography>
                                        <LinkInput
                                            name="youtubeLink"
                                            label="YouTube Link"
                                            placeholder="https://youtube.com/watch?v=..."
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Internal Link
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Click search to find app objects
                                        </Typography>
                                        <LinkInput
                                            name="internalLink"
                                            label="Internal Link"
                                            placeholder="Use search to find objects"
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Email Link
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Mailto links for email addresses
                                        </Typography>
                                        <LinkInput
                                            name="emailLink"
                                            label="Email Link"
                                            placeholder="mailto:example@domain.com"
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Object Type Limitations Demo */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Object Type Limitations</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            The limitTo prop restricts which object types can be selected from the search dialog.
                        </Typography>
                        
                        <Formik
                            initialValues={{
                                projectsOnly: "",
                                usersTeamsOnly: "",
                                everythingAllowed: "",
                            }}
                            onSubmit={(values) => {
                                console.log("Limitations demo submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Projects Only
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Limited to Project objects
                                        </Typography>
                                        <LinkInput
                                            name="projectsOnly"
                                            label="Project Link"
                                            placeholder="Search for projects"
                                            limitTo={["Project", "ProjectVersion"]}
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Users & Teams
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Limited to User and Team objects
                                        </Typography>
                                        <LinkInput
                                            name="usersTeamsOnly"
                                            label="User/Team Link"
                                            placeholder="Search for users or teams"
                                            limitTo={["User", "Team"]}
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            All Objects
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            No limitations (default)
                                        </Typography>
                                        <LinkInput
                                            name="everythingAllowed"
                                            label="Any Link"
                                            placeholder="Search for anything"
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* States Showcase */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Different States</Typography>
                        
                        <Formik
                            initialValues={{
                                normalState: "",
                                disabledState: "https://example.com",
                                errorState: "not-a-valid-url",
                            }}
                            validate={(values) => {
                                const errors: any = {};
                                if (values.errorState && !values.errorState.match(/^https?:\/\/.+/)) {
                                    errors.errorState = "Please enter a valid URL starting with http:// or https://";
                                }
                                return errors;
                            }}
                            onSubmit={(values) => {
                                console.log("States demo submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    {/* Normal State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Normal State
                                        </Typography>
                                        <LinkInput
                                            name="normalState"
                                            label="Normal Link"
                                            placeholder="Enter any link"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Disabled State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Disabled State
                                        </Typography>
                                        <LinkInput
                                            name="disabledState"
                                            label="Disabled Link"
                                            disabled
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Error State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Error State
                                        </Typography>
                                        <LinkInput
                                            name="errorState"
                                            label="Link with Error"
                                            placeholder="Enter a valid URL"
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Features */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Features</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
                            gap: 2, 
                        }}>
                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🔍 Object Search
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Click the search button to find and link to objects within the application.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🌐 External URLs
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Supports any external URL including websites, emails, and phone numbers.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🏷️ Object Preview
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Shows title and description for internal application links when available.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🎯 Type Filtering
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Use limitTo prop to restrict search to specific object types.
                                </Typography>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
