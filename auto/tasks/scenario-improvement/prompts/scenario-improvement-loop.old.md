

Here are some other things you should know:


- `vrooli setup` converts scenarios to generated apps, but DOES NOT run them (and may also skip conversions if the generated app was modified or there is a bug with the caching logic). `vrooli develop` runs the apps, but DOES NOT convert them. But try not to run EITHER of these, as you should only focus on ONE scenario of your choice.
- Many resources are not working right now, and you are FORBIDDEN from trying to fix them or adding new ones. You may start them if they are turned off, and you may use their features. But DO NOT modify the actual resource code itself, or turn off resources unless they're using excessive memory (only applies to whisper, probably).
- If you would like to run an n8n workflow, the n8n resource has webhook/api issues right now. You have to instead use the browserless workaround like this (example): `export N8N_EMAIL="matthalloran8@gmail.com" export N8N_PASSWORD="UQv@9r%%UGMmEJZC" vrooli resource browserless execute-workflow "nZwMYTAQAYUATglq" http://localhost:5678 60000 '{"text": "test"}'`. However, at the time of starting this task (iteration #0), the only n8n workflow that's verified as working is ollama.json (the one used in the example above). Fixing workflows is definitely something you can decide to do in your iteration, but DO NOT attempt to fix the n8n resource. **NOTE:** Shared workflows must be defined in the project-level .vrooli/service.json file and `vrooli setup` called, if you want the workflow to be upserted AND set to active. For scenario-level workflows, they *should* be injected when you do `vrooli app start <scenario_name>` (after you `vrooli scenario convert <scenario_name>`), though this hasn't been tested. As a fallback, you can run the *generated app*'s `./scripts/manage.sh setup` or call the injection engine directly.
-
- DO NOT EVER USE GIT!!! Unless you are simplfy doing a `git status` or `git diff`, which is fine
- DO NOT EVER modify scenario-improvement-loop.md, scenario-improvement-loop.sh, or manage-scenario-loop.sh. They are for running this loop, so modifying them is dangerous. If they are broken or missing, we put backups in /tmp.
- If you want to check the UI, consider using browserless to take a screenshot, and reading that.
- Please follow the SAME structure as other scenarios, where service.json is inside a .vrooli/ folder, data that gets injected into resources is inside an initialization/ folder, UI elements are in a ui/ folder, tests are handled by scenario-test.yaml, and so on. It's important to keep scenarios consistent.
- One additional thing I'd like scenarios to have which many may be missing is a small README.md file. These should be no more than 100 lines, with information on what the scenario is, why it's useful, what other scenarios it relies on/adds capabilities for, the UX style we're going for, etc.
- When designing a UI, please choose a style that's appropriate for the scenario. For example, study-buddy may have a generally clean aestethic but also cute and fun, such as a Lofi Girl type of vibe. retro-game-launcher would obviously ben 80s arcade retro style, as implied by the name. system-monitor could have a dark and green hackery or matrix vibe. app-debugger, product-manager, roi-fit-analysis, and other boring adult scenarios would likely be a more standard modern web ui. Take into account the type of scenario and how you think it'll be used when deciding on design decisions. Having them unique and varied is beneficial for scenarios which are more consumer-facing, while business-facing scenarios should be more professional. Scenarios which are used internally or for development can be up to you. 

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
- Changing the scenario's initialization/ folder structure? Could be a good idea. Scenarios used to be structured like initialization/automation/n8n/workflow.json. We've been moving toward getting rid of the categories, and using initialization/n8n/workflow.json instead. Much cleaner.
- Adding/fixing scenario's initialization/ files? Likely a good idea.
- Adding a bunch of code to the go API? Likely not a good idea. Consider moving logic to n8n/node-red/huginn/etc. workflows.
- Adding a bunch of code to the CLI? NOT a good idea. They are meant to be lightweight wrappers around the API.
- Adding a lot of code in general? Might not be a good idea. Scenarios are meant to be small and single-purposed.
- Updating an n8n workflow to take advantage of one of the shared workflows in the project-level initialization/ folder? GREAT idea. Many workflows in there were created *because* they are genuinely useful for reducing the complexity required to build scenarios. For example, the ollama.json workflow helps us reliably use ollama in all other workflows. rate-limiter.json is meant as a general rate limiting tool. embedding-generator.json is great for semantic search boxes, and so on. initialization/n8n/README.md has the most detailed information in the project-level shared n8n workflows.





Remember, some scenarios are used to increase the power and capability of Vrooli itself, which means they can call each other using their clis. For example, morning-vision-walk would likely rely heavily on stream-of-consciousness-analyzer, product-manager-agent, and maybe even life-coach. Other scenarios are demonstrations of what we can do and apps that might generate revenue. This includes scenarios which are intended to be deployed as a SaaS app or to a customer on Upwork, as well as scenarios which would be used internally by businesses or individuals with a local Vrooli server. The possibilities are only bounded by our current set of resources, which should hopefully be improved soon (but NOT in this chat).
