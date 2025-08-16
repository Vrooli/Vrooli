You are an expert software engineer and automation visionary. Your task is to help me improve the set of scenarios in scripts/scenarios/core/. These scenarios are crucial for increasing the capabilities of the Vrooli platform. I will be away for the night, and am asking you to make one addition or improvement to our scenarios. I will be asking you this in a loop every 5 minutes, so make sure to keep your changes concise and localized to one area. DO NOT make any changes to our main scripts, or any other files outside of scripts/scenarios/core/ besides potentially the root initialization/ folder. Following this rule limits the chance of you messing things up for the subsequent iterations.

Here are some other things you should know:
- Use a file /tmp/vrooli-scenario-improvement.md to store information for your next iteration. Keep this file small and concise. If it is over 1000 lines, clean up some old notes. You should store no more than 10 lines of notes per iteration.
- Use `vrooli help` to see all the commands available to you. Prefer using the vrooli CLI over the bash scripts directly.
- We recently (and may still) have an issue with `vrooli develop` which causes excessive CPU usage. Be wary of this. I'm not sure *exactly* where the issue is. It may be part of manage.sh or another shared script file, in which case each app you start will continue to hog CPU threads at 100% until you kill the process. You are not forbidden from using `vrooli develop`, but just investigate and clean up any issues you find BEFORE any `vrooli develop` calls.
- To reliably convert one scenario, run `vrooli scenario convert <scenario-name> --force`. To run the converted app, run `vrooli scenario start <scenario-name>`. To stop, press CTRL+C from the same terminal, or .
- `vrooli setup` converts scenarios to generated apps, but DOES NOT run them (and may also skip conversions if the generated app was modified or there is a bug with the caching logic). `vrooli develop` runs the apps, but DOES NOT convert them. But try not to run EITHER of these, as you should only focus on one scenario of your choice.
- Most resoures are not working right now, and you are FORBIDDEN from trying to fix them or adding new ones. If you would like to run a workflow, you may use the browserless workaround like this (example): `N8N_EMAIL="matthalloran8@gmail.com" N8N_PASSWORD="UQv@9r%%UGMmEJZC" vrooli resource browserless execute-workflow <workflow_id> http://localhost:5678 3000 '{"text":"Hello world", "model":"nomic-embed-text"}'` (this itself may also be broken btw). NOTE: Shared workflows must be defined in the project-level .vrooli/service.json file and `vrooli setup` called, if you want the workflow to be upserted AND set to active. For scenario-level workflows, I think you can run the *generated app*'s `./scripts/manage.sh setup` or call the injection engine directly.
- When writing n8n workflows, follow the same high standards we use for the existing workflows in initialization/n8n/. This includes having both a webhook and manual trigger, with the manual trigger going into a node that provides defaults (the webhook trigger should NOT go through that node). It also includes using the ollama.json workflow instead of calling ollama directly.
- DO NOT EVER USE GIT!!! Unless you are simplfy doing a `git status` or `git diff`, which is fine
- DO NOT EVERY modify scenario-improvement-loop.md, scenario-improvement-loop.sh, or manage-scenario-loop.sh. They are for running this loop, so modifying them is dangerous. If they are broken or missing, we put backups in /tmp.
- If you want to check the UI, consider using browserless to take a screenshot, and reading that
- Please follow the SAME structure as other scenarios, where service.json is inside a .vrooli/ folder, data that gets injected into resources is inside an initialization/ folder, UI elements are in a ui/ folder, tests are handled by scenario-test.yaml, and so on. It's important to keep scenarios consistent

Before doing anything, read up on:
- /tmp/vrooli-scenario-improvement.md to see what you've done in the past
- The main README.md file in the root of the project
- scripts/scenarios/README.md folder to understand how scenarios work
- scripts/scenarios/catalog.json to understand the current scenarios
- scripts/scenarios/tools/app-structure.json to understand how scenarios are copied to the generated apps. Note that they copy some files from the root of the project
- scripts/resources/README.md to understand what resources are and how scenarios use them
- The (untested, so unsure of accuracy) prompt for adding and fixing scenarios in scripts/scenarios/core/prompt-manager/initialization/prompts/features/add-fix-scenario.md
- initialization/README.md at the root of the project to understand the current set of shared resource data that are injected directly by vrooli, and can be used by any scenario

Then, take a minute to understand the current state of the scenarios. They are in various states of completion, I'm sure. Pick a scenario you think is either poor or important, and improve it. Here are some rules to make sure your changes are good:
- Changing something *outside* of the scenario's folder? I HIGHLY advise against it. The only time this might be a good idea is when adding a new workflow to the root initialization/ folder. But that should only be done if you think it's beneficial for other scenarios to use the workflow (which could very well be the case).
- Drastically changing the scenario's .vrooli/service.json? Probably not a good idea. If you add or remove initialization data or change what resources are used, then you may need to update the service.json file. Otherwise, it's probably fine as-is or can be tweaked manually by me.
- Changing the scenario's initialization/ folder structure? Could be a good idea. Most scenarios at iteration 0 are structured like initialization/automation/n8n/workflow.json. We're slowly moving toward getting rid of the categories, and using initialization/n8n/workflow.json instead. Much cleaner.
- Adding/fixing scenario's initialization/ files? Likely a good idea.
- Adding a bunch of code to the go API? Likely not a good idea. Consider moving logic to n8n/node-red/huginn/etc. workflows.
- Switching from windmill to javascript, or vice-versa? NOT a good idea. We should keep the UIs as-is for now
- Adding a bunch of code to the CLI? NOT a good idea. They are meant to be lightweight wrappers around the API.
- Adding a lot of code in general? Might not be a good idea. Scenarios are meant to be small and single-purposed.
- Updating an n8n workflow to take advantage of one of the shared workflows? GREAT idea

If all scenarios look pretty good, you may consider adding a new scenario. For adding new scenarios, you may only pick from this list (in no particular order):
- app-debugger
- n8-workflow-generator
- prompt-performance-evaluator
- video-downloader
- news-aggregator-bias-analysis
- qr-code-generator
- typing-test
- invoice-generator
- app-to-ios (updated *generated* apps to have a platform/ios folder with a functional ios app)
- app-to-electron (similar to app-to-ios, but for Electron apps)
- app-onboarding-manager (analyzes a scenario [that's meant to be an app] for insufficient user onboarding, and improves)
- produce-manager-agent
- dream-analyzer
- life-coach
- chore-tracking-and-reward-bank
- nutrition-tracker
- app-scroller (scroll through running apps like you're on Tiktok)
- roi-fit-analysis
- seo-optimizer
- study-buddy
- local-info-scout

Remember, some scenarios are used to increase the power and capability of Vrooli itself, and others are demonstrations of what we can do and apps that might generate revenue. At the first iteration (may not be true now), the only UI types generated apps had are windmill UIs and javascript ones. I prefer javascript ones, as they can be fully customized. In fact, one of the goals of scenarios is to abstract and push out nearly every part of an app, so coding agents like yourself can focus on building beautiful single or few page apps. Feel free to make them stand out and be unique. I want them to feel fun and experimental, like the old internet days. Having a bunch of apps with the same blank layout does not spark joy.

Ultra think. You may use the web if needed.