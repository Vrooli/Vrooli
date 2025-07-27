# 04 - AI Integration

Unlock the full power of Agent S2 with AI-driven automation.

## Prerequisites

1. Complete all previous sections
2. Set up AI credentials:
   ```bash
   export ANTHROPIC_API_KEY=your_key_here
   # or
   export OPENAI_API_KEY=your_key_here
   ```
3. Enable AI in Agent S2:
   ```bash
   export AGENT_S2_AI_ENABLED=true
   ```

## Examples in this section

### natural_language_tasks.py
Control your computer with natural language:
- "Click the blue button"
- "Type my email address"
- "Open the settings menu"

```bash
python natural_language_tasks.py
```

### visual_reasoning.py
Let AI understand and interact with what's on screen:
- Identify UI elements
- Read and understand text
- Make decisions based on visual context

```bash
python visual_reasoning.py
```

### autonomous_planning.py
Have AI plan and execute multi-step tasks:
- Break down complex goals
- Execute step-by-step plans
- Adapt to changing conditions

```bash
python autonomous_planning.py
```

### adaptive_automation.py
Create automations that adapt to different scenarios:
- Handle variations in UI
- Recover from unexpected states
- Learn from experience

```bash
python adaptive_automation.py
```

## AI Capabilities

### 1. Natural Language Understanding
- Interpret user commands in plain English
- Understand context and intent
- Handle ambiguous instructions

### 2. Visual Understanding
- Identify UI elements
- Read on-screen text
- Understand layouts and relationships

### 3. Planning and Reasoning
- Break down complex tasks
- Create step-by-step plans
- Make logical decisions

### 4. Adaptive Behavior
- Handle UI variations
- Recover from errors
- Improve over time

## Best Practices

1. **Be specific** - Clear instructions get better results
2. **Provide context** - Help the AI understand the goal
3. **Verify results** - AI can make mistakes, always verify
4. **Use fallbacks** - Have non-AI alternatives ready
5. **Monitor costs** - AI API calls can add up

## Examples of AI Commands

```python
# Simple commands
ai.perform_task("Click the Submit button")
ai.perform_task("Type 'Hello World' in the text field")

# Complex commands
ai.perform_task("Find the email from John and reply with 'Thanks!'")
ai.perform_task("Navigate to settings and enable dark mode")

# Multi-step plans
ai.generate_plan("Organize my desktop icons by type")
ai.execute_workflow("login", {"username": "user", "password": "pass"})
```

## Troubleshooting

### AI not available
- Check API keys are set
- Verify AI is enabled
- Check logs for initialization errors

### Poor results
- Provide more specific instructions
- Include context about the current state
- Try breaking down into smaller tasks

### Performance issues
- AI calls can be slow (1-5 seconds)
- Use core automation for time-critical tasks
- Consider caching AI decisions

## What's Next?

You've mastered Agent S2! Consider:
- Building your own automation workflows
- Contributing examples to the project
- Exploring advanced AI models and capabilities