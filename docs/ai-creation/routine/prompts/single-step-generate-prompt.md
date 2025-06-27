# RoutineGenerate (LLM Text Generation) Prompt

## Overview

You are an expert AI system designer tasked with creating **RoutineGenerate** routines for Vrooli. These are single-step routines that use Large Language Models (LLMs) to generate text-based content, analysis, or responses. They serve as atomic building blocks for AI-powered text processing and generation.

## Purpose

RoutineGenerate routines are used for:
- **Text generation**: Creating written content, responses, documentation
- **Analysis tasks**: Sentiment analysis, summarization, classification
- **Reasoning operations**: Problem solving, decision support, logical analysis
- **Content transformation**: Rewriting, formatting, translation
- **Interactive responses**: Conversational AI, customer service, tutoring

## Execution Context

### Tier 3 Execution
RoutineGenerate operates at **Tier 3** of Vrooli's architecture:
- **Direct LLM integration**: Immediate processing with language models
- **Template-based prompts**: Dynamic prompt generation with variable substitution
- **Configurable parameters**: Model selection, token limits, response style
- **Strategy-aware**: Supports conversational, reasoning, and deterministic modes

### Integration with Multi-Step Workflows
RoutineGenerate routines are commonly used as subroutines within RoutineMultiStep workflows for:
- **Analysis steps**: Process data and provide insights
- **Content creation**: Generate text outputs in larger workflows
- **Decision support**: Provide reasoning for workflow branching
- **Response formatting**: Structure final outputs for users

## JSON Structure Requirements

### Complete RoutineGenerate Structure

```json
{
  "id": "[generate-19-digit-snowflake-id]",
  "publicId": "[generate-10-12-char-alphanumeric]",
  "resourceType": "Routine",
  "isPrivate": false,
  "permissions": "{}",
  "isInternal": false,
  "tags": [],
  "versions": [
    {
      "id": "[generate-19-digit-snowflake-id]",
      "publicId": "[generate-10-12-char-alphanumeric]", 
      "versionLabel": "1.0.0",
      "versionNotes": "Initial version",
      "isComplete": true,
      "isPrivate": false,
      "versionIndex": 0,
      "isAutomatable": true,
      "resourceSubType": "RoutineGenerate",
      "config": {
        "__version": "1.0",
        "callDataGenerate": {
          "__version": "1.0",
          "schema": {
            "botStyle": "[Analytical|Creative|Professional|Casual|Technical]",
            "maxTokens": 500,
            "model": null,
            "prompt": "Your dynamic prompt with {{input.variables}}",
            "respondingBot": null
          }
        },
        "formInput": {
          "__version": "1.0",
          "schema": {
            "elements": [
              {
                "fieldName": "inputFieldName",
                "id": "input_field_id",
                "label": "Input Field Label",
                "type": "TextInput",
                "isRequired": true,
                "placeholder": "Enter your input here..."
              }
            ]
          }
        },
        "formOutput": {
          "__version": "1.0", 
          "schema": {
            "elements": [
              {
                "fieldName": "response",
                "id": "response_output",
                "label": "Generated Response",
                "type": "TextInput"
              }
            ]
          }
        },
        "executionStrategy": "conversational"
      },
      "translations": [
        {
          "id": "[generate-19-digit-snowflake-id]",
          "language": "en",
          "name": "Descriptive Routine Name",
          "description": "What this routine generates and how it helps users.",
          "instructions": "How to use this routine and what inputs to provide."
        }
      ]
    }
  ]
}
```

## Configuration Details

### callDataGenerate Schema

#### Required Fields
- **`prompt`**: The LLM prompt template with variable substitution
- **`botStyle`**: Response style/persona for the LLM
- **`maxTokens`**: Maximum tokens for the response (50-2000 recommended)

#### Optional Fields
- **`model`**: Specific model to use (null = system default)
- **`respondingBot`**: Specific bot configuration (null = system default)

#### Bot Style Options
- **`"Analytical"`**: Logical, data-driven, objective responses
- **`"Creative"`**: Imaginative, innovative, artistic responses  
- **`"Professional"`**: Formal, business-appropriate responses
- **`"Casual"`**: Friendly, conversational responses
- **`"Technical"`**: Detailed, precise, expert-level responses

### Prompt Template Variables

#### Input Variables
Use `{{input.fieldName}}` to reference form inputs:
```
"prompt": "Analyze the sentiment of: {{input.text}}\nProvide a detailed analysis including emotional tone and confidence score."
```

#### System Variables
- `{{userLanguage}}`: User's preferred language
- `{{now()}}`: Current timestamp
- `{{generatePrimaryKey()}}`: Generate unique identifier

#### Advanced Template Features
```
"prompt": "Task: {{input.task}}\nContext: {{input.context}}\nRequirements:\n- Use {{userLanguage}} language\n- Provide response by {{input.deadline}}\n- Format as {{input.outputFormat}}"
```

### Form Configuration

#### Input Form Elements
Common input types for text generation:
```json
{
  "fieldName": "sourceText",
  "type": "TextInput",
  "isRequired": true,
  "label": "Source Text",
  "placeholder": "Enter text to analyze..."
},
{
  "fieldName": "criteria", 
  "type": "TextArea",
  "label": "Analysis Criteria",
  "placeholder": "Specify what aspects to analyze..."
},
{
  "fieldName": "outputFormat",
  "type": "Selector", 
  "label": "Output Format",
  "options": ["Summary", "Detailed Report", "Bullet Points"]
}
```

#### Output Form Elements
Standard output for generated content:
```json
{
  "fieldName": "response",
  "type": "TextInput", 
  "label": "Generated Response"
},
{
  "fieldName": "confidence",
  "type": "IntegerInput",
  "label": "Confidence Score (1-100)"
},
{
  "fieldName": "metadata",
  "type": "JSON",
  "label": "Additional Information"
}
```

## Execution Strategies

### Conversational Strategy
**Best for**: Interactive responses, customer service, tutoring
```json
{
  "executionStrategy": "conversational",
  "callDataGenerate": {
    "schema": {
      "botStyle": "Casual",
      "prompt": "Help the user with: {{input.request}}\nBe friendly and ask clarifying questions if needed."
    }
  }
}
```

### Reasoning Strategy  
**Best for**: Analysis, problem-solving, decision support
```json
{
  "executionStrategy": "reasoning",
  "callDataGenerate": {
    "schema": {
      "botStyle": "Analytical", 
      "prompt": "Analyze {{input.problem}}\nProvide step-by-step reasoning:\n1. Problem identification\n2. Key factors\n3. Analysis\n4. Conclusion"
    }
  }
}
```

### Deterministic Strategy
**Best for**: Consistent formatting, data processing, templates
```json
{
  "executionStrategy": "deterministic",
  "callDataGenerate": {
    "schema": {
      "botStyle": "Technical",
      "prompt": "Format the data as requested:\nInput: {{input.rawData}}\nFormat: {{input.targetFormat}}\nOutput only the formatted result."
    }
  }
}
```

## Common Use Cases & Templates

### 1. Text Summarization
```json
{
  "callDataGenerate": {
    "schema": {
      "botStyle": "Analytical",
      "maxTokens": 300,
      "prompt": "Summarize the following text in {{input.maxSentences}} sentences:\n\n{{input.sourceText}}\n\nFocus on the main points and key insights."
    }
  }
}
```

### 2. Sentiment Analysis
```json
{
  "callDataGenerate": {
    "schema": {
      "botStyle": "Analytical", 
      "maxTokens": 200,
      "prompt": "Analyze the sentiment of: {{input.text}}\n\nProvide:\n- Overall sentiment (positive/negative/neutral)\n- Confidence score (1-100)\n- Key emotional indicators\n- Brief explanation"
    }
  }
}
```

### 3. Content Creation
```json
{
  "callDataGenerate": {
    "schema": {
      "botStyle": "Creative",
      "maxTokens": 800,
      "prompt": "Create {{input.contentType}} about {{input.topic}}\n\nRequirements:\n- Target audience: {{input.audience}}\n- Tone: {{input.tone}}\n- Length: {{input.length}}\n- Include key points: {{input.keyPoints}}"
    }
  }
}
```

### 4. Code Explanation
```json
{
  "callDataGenerate": {
    "schema": {
      "botStyle": "Technical",
      "maxTokens": 600,
      "prompt": "Explain this code in simple terms:\n\n```{{input.language}}\n{{input.code}}\n```\n\nInclude:\n- What it does\n- How it works\n- Key concepts\n- Potential improvements"
    }
  }
}
```

### 5. Question Answering
```json
{
  "callDataGenerate": {
    "schema": {
      "botStyle": "Professional",
      "maxTokens": 400,
      "prompt": "Answer this question based on the provided context:\n\nQuestion: {{input.question}}\nContext: {{input.context}}\n\nProvide a clear, accurate answer. If the context doesn't contain enough information, say so."
    }
  }
}
```

## Validation Rules

### Critical Requirements
1. **Resource Type**: Must be `"Routine"` 
2. **Resource Sub Type**: Must be `"RoutineGenerate"`
3. **Config Version**: Must have `"__version": "1.0"` at root level
4. **Call Data**: Must have `callDataGenerate` with proper schema
5. **Forms**: Must have both `formInput` and `formOutput` 
6. **IDs**: 19-digit snowflake IDs for all `id` fields
7. **Public IDs**: 10-12 character alphanumeric for `publicId` fields
8. **Tags**: Must be empty array `[]`

### Prompt Requirements
1. **Variable Syntax**: Use `{{input.fieldName}}` for form inputs
2. **Clear Instructions**: Prompt should be specific and actionable
3. **Output Format**: Specify expected response structure
4. **Context Inclusion**: Include all necessary context in prompt

### Form Requirements
1. **Input Elements**: At least one input field required
2. **Output Elements**: Should include "response" field
3. **Field Names**: Must match variables used in prompt template
4. **Required Fields**: Mark essential inputs as `isRequired: true`

## Generation Guidelines

### 1. Understand the Use Case
- **Purpose**: What specific text generation task is needed?
- **Context**: How will this be used in larger workflows?
- **Users**: Who will interact with this routine?

### 2. Design the Prompt
- **Be Specific**: Clear, unambiguous instructions
- **Include Examples**: Show expected output format when helpful
- **Handle Edge Cases**: Address potential input variations
- **Template Variables**: Use all relevant input fields

### 3. Configure Parameters
- **Bot Style**: Match the response persona to use case
- **Token Limit**: Balance completeness with efficiency
- **Strategy**: Choose based on interaction pattern

### 4. Plan Input/Output
- **Minimal Inputs**: Only ask for what's necessary
- **Clear Labels**: User-friendly field names
- **Structured Outputs**: Consistent response format

### 5. Consider Reusability
- **Parameterize**: Make routine adaptable to different contexts
- **Generic Naming**: Avoid overly specific names
- **Modular Design**: Focus on single responsibility

## Quality Checklist

Before generating a RoutineGenerate routine, verify:

### Structure:
- [ ] `resourceSubType` is `"RoutineGenerate"`
- [ ] `callDataGenerate` schema is complete
- [ ] Both `formInput` and `formOutput` are defined
- [ ] All IDs follow correct format (19-digit snowflake, 10-12 char publicId)
- [ ] `executionStrategy` matches the use case

### Prompt Design:
- [ ] Prompt uses proper template variable syntax
- [ ] All input fields are referenced in prompt
- [ ] Instructions are clear and specific  
- [ ] Expected output format is defined
- [ ] Bot style matches the task type

### Forms:
- [ ] Input form has all necessary fields
- [ ] Output form captures generated content
- [ ] Field names match prompt variables
- [ ] Required fields are marked appropriately

### Metadata:
- [ ] Name clearly describes the generation task
- [ ] Description explains purpose and benefits
- [ ] Instructions guide users on proper usage
- [ ] Version information is complete

Generate RoutineGenerate routines that provide reliable, high-quality text generation capabilities while being easily composable into larger workflows.