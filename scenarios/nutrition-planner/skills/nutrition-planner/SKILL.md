---
name: "nutrition-planner"
description: "Plan what to eat, shop and cook: build a meal plan around your diet, kitchen, effort, budget and nutrition goals, keep your recipes, and make a grocery shopping list (Nooch)."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["nutrition-planner", "nooch", "meal-plan", "recipes", "groceries", "shopping-list", "diet", "cooking", "nutrition"]
  requires:
    scenarios: ["nutrition-planner"]
    commands: ["nutrition-planner workspace list"]
  origin:
    kind: "authored"
---

## nutrition-planner

Workspaces hold recipes and plans.

| You want | Command |
|---|---|
| List workspaces | `nutrition-planner workspace list` |
| Recipes | `nutrition-planner recipe list`, `nutrition-planner recipe create` |

Run `nutrition-planner <group> <command> --help` for flags.

The CLI covers workspaces and recipes only; meal plans and shopping lists are not CLI commands yet.
