# Collision Avoidance

## Purpose
You work on one assigned resource or scenario. Avoid conflicts by staying within your boundaries and working around limitations gracefully.

## Core Principle: Stay in Your Lane
**Work only within your assigned resource/scenario directory.** Do not modify:
- Files in other resources/scenarios
- Core Vrooli CLI code
- Shared infrastructure files
- Other agents' assigned tasks

## Handling Limitations
When you encounter blockers:

**❌ DO NOT try to fix external issues:**
- Critical bugs in main Vrooli CLI
- Semantic knowledge system not working
- Other resources/scenarios broken

**✅ DO work around limitations:**
- Document the limitation in your response
- Find alternative approaches within your scope
- Focus on what you can control

## Managing Dependencies
**You MAY start/install existing resources/scenarios if:**
- They are required for your task
- They are not currently running
- Use: `vrooli resource NAME develop` or `vrooli scenario NAME develop`

**❌ NEVER stop/uninstall anything that's already running**
- Assume other agents depend on running services
- If something must be stopped, document why and let infrastructure handle it

## Simple Rules
1. **Stick to your assigned resource/scenario only**
2. **Work around external limitations - don't fix them**
3. **Can start dependencies - never stop them**
4. **Document blockers clearly**
5. **Focus on deliverables within your control**

This approach prevents conflicts while ensuring each agent can make meaningful progress on their assigned work.