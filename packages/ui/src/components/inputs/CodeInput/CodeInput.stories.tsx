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
import { Formik, Form } from "formik";
import { CodeLanguage } from "@vrooli/shared";
import { CodeInput, CodeInputBase } from "./CodeInput.js";
import { Switch } from "../Switch/Switch.js";

const meta: Meta<typeof CodeInput> = {
    title: "Components/Inputs/CodeInput",
    component: CodeInput,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Sample code for different languages
const sampleCode = {
    [CodeLanguage.Javascript]: `function fibonacci(n) {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}

console.log(fibonacci(10));`,
    [CodeLanguage.Python]: `def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n - 1) + fibonacci(n - 2)

print(fibonacci(10))`,
    [CodeLanguage.Json]: `{
    "name": "John Doe",
    "age": 30,
    "email": "john@example.com",
    "address": {
        "street": "123 Main St",
        "city": "Anytown",
        "zipCode": "12345"
    },
    "hobbies": ["reading", "swimming", "coding"]
}`,
    [CodeLanguage.Html]: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sample Page</title>
</head>
<body>
    <header>
        <h1>Welcome to My Website</h1>
    </header>
    <main>
        <p>This is a sample HTML page.</p>
    </main>
</body>
</html>`,
    [CodeLanguage.Css]: `.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

.header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 2rem;
    border-radius: 8px;
}

.button {
    background-color: #3498db;
    color: white;
    padding: 10px 20px;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.3s ease;
}

.button:hover {
    background-color: #2980b9;
}`,
    [CodeLanguage.Typescript]: `interface User {
    id: number;
    name: string;
    email: string;
    isActive: boolean;
}

class UserService {
    private users: User[] = [];

    addUser(user: Omit<User, 'id'>): User {
        const newUser: User = {
            ...user,
            id: this.users.length + 1
        };
        this.users.push(newUser);
        return newUser;
    }

    getUserById(id: number): User | undefined {
        return this.users.find(user => user.id === id);
    }
}`,
    [CodeLanguage.Shell]: `#!/bin/bash

# Update package lists
sudo apt update

# Install required packages
sudo apt install -y git nodejs npm

# Clone repository
git clone https://github.com/example/repo.git
cd repo

# Install dependencies
npm install

# Start the application
npm start`,
    [CodeLanguage.Sql]: `SELECT 
    u.id,
    u.username,
    u.email,
    COUNT(p.id) as post_count,
    MAX(p.created_at) as last_post_date
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
WHERE u.is_active = true
    AND u.created_at >= DATE_SUB(NOW(), INTERVAL 1 YEAR)
GROUP BY u.id, u.username, u.email
HAVING post_count > 0
ORDER BY post_count DESC, last_post_date DESC
LIMIT 10;`,
};

// Interactive CodeInput Playground
export const CodeInputShowcase: Story = {
    render: () => {
        const [disabled, setDisabled] = useState(false);
        const [selectedLanguage, setSelectedLanguage] = useState(CodeLanguage.Javascript);
        
        // Get a subset of popular languages for the demo
        const popularLanguages = [
            CodeLanguage.Javascript,
            CodeLanguage.Typescript,
            CodeLanguage.Python,
            CodeLanguage.Json,
            CodeLanguage.Html,
            CodeLanguage.Css,
            CodeLanguage.Shell,
            CodeLanguage.Sql,
        ];

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
                        <Typography variant="h5" sx={{ mb: 3 }}>CodeInput Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Language Selection */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Language</FormLabel>
                                <RadioGroup
                                    value={selectedLanguage}
                                    onChange={(e) => setSelectedLanguage(e.target.value as CodeLanguage)}
                                    sx={{ gap: 0.5, maxHeight: 200, overflow: "auto" }}
                                >
                                    {popularLanguages.map(lang => (
                                        <FormControlLabel 
                                            key={lang}
                                            value={lang} 
                                            control={<Radio size="small" />} 
                                            label={lang} 
                                            sx={{ m: 0 }} 
                                        />
                                    ))}
                                </RadioGroup>
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
                        </Box>
                    </Box>

                    {/* CodeInput Demo with Formik */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>CodeInput with Formik</Typography>
                        
                        <Formik
                            initialValues={{
                                codeLanguage: selectedLanguage,
                                content: sampleCode[selectedLanguage] || "",
                            }}
                            onSubmit={(values) => {
                                console.log("Form submitted:", values);
                            }}
                            enableReinitialize
                        >
                            {({ values, setFieldValue }) => {
                                // Update values when selectedLanguage changes
                                React.useEffect(() => {
                                    setFieldValue("codeLanguage", selectedLanguage);
                                    setFieldValue("content", sampleCode[selectedLanguage] || "");
                                }, [selectedLanguage, setFieldValue]);

                                return (
                                    <Form>
                                        <Box sx={{ mb: 2 }}>
                                            <Typography variant="body2" color="text.secondary">
                                                Current Language: <strong>{values.codeLanguage}</strong>
                                            </Typography>
                                            <Typography variant="body2" color="text.secondary">
                                                Content Length: <strong>{values.content.length} characters</strong>
                                            </Typography>
                                        </Box>
                                        <CodeInput
                                            name="content"
                                            codeLanguageField="codeLanguage"
                                            disabled={disabled}
                                        />
                                    </Form>
                                );
                            }}
                        </Formik>
                    </Box>

                    {/* CodeInputBase Demo (Direct Usage) */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>CodeInputBase (Direct Usage)</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            This demonstrates using CodeInputBase directly without Formik for more control.
                        </Typography>
                        
                        <CodeInputBase
                            name="directCodeInput"
                            codeLanguage={selectedLanguage}
                            content={sampleCode[selectedLanguage] || ""}
                            handleCodeLanguageChange={(newLang) => {
                                setSelectedLanguage(newLang);
                                console.log("Language changed to:", newLang);
                            }}
                            handleContentChange={(newContent) => {
                                console.log("Content changed:", newContent.slice(0, 100) + "...");
                            }}
                            disabled={disabled}
                        />
                    </Box>

                    {/* Language-Specific Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Language-Specific Examples</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", lg: "repeat(2, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* JSON Example with Validation */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>JSON with Validation</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Try editing the JSON to see real-time validation and error highlighting.
                                </Typography>
                                <CodeInputBase
                                    name="jsonExample"
                                    codeLanguage={CodeLanguage.Json}
                                    content={`{
    "valid": true,
    "message": "This is valid JSON",
    "array": [1, 2, 3]
}`}
                                    handleCodeLanguageChange={() => {}}
                                    handleContentChange={(content) => console.log("JSON updated:", content)}
                                    disabled={disabled}
                                />
                            </Box>

                            {/* Limited Language Selection */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Limited Language Options</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    This example limits available languages to web technologies only.
                                </Typography>
                                <CodeInputBase
                                    name="webExample"
                                    codeLanguage={CodeLanguage.Html}
                                    content={sampleCode[CodeLanguage.Html]}
                                    handleCodeLanguageChange={(newLang) => console.log("Web language changed:", newLang)}
                                    handleContentChange={(content) => console.log("Web content updated:", content.slice(0, 50) + "...")}
                                    limitTo={[CodeLanguage.Html, CodeLanguage.Css, CodeLanguage.Javascript]}
                                    disabled={disabled}
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Features Showcase */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Features Showcase</Typography>
                        
                        <Box sx={{ mb: 3 }}>
                            <Typography variant="h6" sx={{ mb: 2 }}>Key Features:</Typography>
                            <Box component="ul" sx={{ pl: 3, m: 0 }}>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Syntax Highlighting:</strong> Language-specific syntax highlighting for all supported languages
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Language Selection:</strong> Dropdown to change programming language (when multiple languages allowed)
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Code Validation:</strong> Real-time validation and error highlighting for supported languages (JSON, etc.)
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Word Wrap Toggle:</strong> Switch between wrapped and unwrapped text display
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Expand/Collapse:</strong> Minimize editor when not in use
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Copy to Clipboard:</strong> One-click copying of entire code content
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Undo/Redo:</strong> Standard text editing operations
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Format JSON:</strong> Automatic JSON formatting for JSON content
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    <strong>Dark/Light Theme:</strong> Automatic theme detection and switching
                                </Typography>
                            </Box>
                        </Box>

                        <Typography variant="h6" sx={{ mb: 2 }}>Try These Actions:</Typography>
                        <Box component="ul" sx={{ pl: 3, m: 0 }}>
                            <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                Change the language using the dropdown in the toolbar
                            </Typography>
                            <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                Toggle word wrap using the wrap icon
                            </Typography>
                            <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                Collapse/expand the editor using the arrow icons
                            </Typography>
                            <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                For JSON: try adding invalid syntax to see error highlighting
                            </Typography>
                            <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                Use the format button (for JSON) to pretty-print the code
                            </Typography>
                        </Box>
                    </Box>

                    {/* All Languages Grid */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Supported Languages</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            CodeInput supports {Object.values(CodeLanguage).length} programming languages with syntax highlighting and various features.
                        </Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "repeat(2, 1fr)", sm: "repeat(3, 1fr)", md: "repeat(4, 1fr)", lg: "repeat(6, 1fr)" },
                            gap: 1, 
                        }}>
                            {Object.values(CodeLanguage).map(lang => (
                                <Box 
                                    key={lang}
                                    sx={{ 
                                        p: 1,
                                        borderRadius: 1,
                                        bgcolor: lang === selectedLanguage ? "primary.main" : "grey.100",
                                        color: lang === selectedLanguage ? "primary.contrastText" : "text.primary",
                                        textAlign: "center",
                                        cursor: "pointer",
                                        transition: "all 0.2s ease",
                                        "&:hover": {
                                            bgcolor: lang === selectedLanguage ? "primary.dark" : "grey.200",
                                        },
                                    }}
                                    onClick={() => setSelectedLanguage(lang)}
                                >
                                    <Typography variant="caption" sx={{ fontWeight: lang === selectedLanguage ? "bold" : "normal" }}>
                                        {lang}
                                    </Typography>
                                </Box>
                            ))}
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};