You are an expert software engineer and automation visionary. Your task is to help me improve AND validate the set of scenarios in scripts/scenarios/core/. These scenarios are crucial for increasing the capabilities of the Vrooli platform. I will be away for the night, and am asking you to make one addition or improvement to our scenarios. I will be asking you this in a loop every 5 minutes, so make sure to keep your changes concise and localized to one area. DO NOT make any changes to our main scripts, or any other files outside of scripts/scenarios/core/ (besides any special cases mentioned later in this prompt). Following this rule limits the chance of you messing things up for the subsequent iterations.

A quality scenario should:
- Use shared n8n workflows when possible
- Follow our scenario structure guidelines
- Sucessfully be converted into an app, which STARTS and is validated through the API, CLI, or a browserless screenshot. DO NOT assume that a scenario works - in fact, most probably don't work right now

Here are some other things you should know:
- Use a file /tmp/vrooli-scenario-improvement.md to store information for your next iteration. Keep this file small and concise. If it is over 1000 lines, clean up some old notes. You should store no more than 10 lines of notes per iteration.
- Use `vrooli help` to see all the commands available to you. Prefer using the vrooli CLI over the bash scripts directly.
- We recently (and may still) have an issue with `vrooli develop` which causes excessive CPU usage. Be wary of this. I'm not sure *exactly* where the issue is. It may be part of manage.sh or another shared script file, in which case each app you start will continue to hog CPU threads at 100% until you kill the process. You are not forbidden from using `vrooli develop`, but just investigate and clean up any issues you find BEFORE any `vrooli develop` calls.
- To reliably convert one scenario, run `vrooli scenario convert <scenario-name> --force`. To run the converted app, run `vrooli scenario start <scenario-name>`. To stop, press CTRL+C from the same terminal, or .
- `vrooli setup` converts scenarios to generated apps, but DOES NOT run them (and may also skip conversions if the generated app was modified or there is a bug with the caching logic). `vrooli develop` runs the apps, but DOES NOT convert them. But try not to run EITHER of these, as you should only focus on ONE scenario of your choice.
- Many resources are not working right now, and you are FORBIDDEN from trying to fix them or adding new ones. You may start them if they are turned off, and you may use their features. But DO NOT modify the actual resource code itself, or turn off resources unless they're using excessive memory (only applies to whisper, probably).
- If you would like to run an n8n workflow, the n8n resource has webhook/api issues right now. You have to instead use the browserless workaround like this (example): `export N8N_EMAIL="matthalloran8@gmail.com" export N8N_PASSWORD="UQv@9r%%UGMmEJZC" vrooli resource browserless execute-workflow "nZwMYTAQAYUATglq" http://localhost:5678 60000 '{"text": "test"}'`. However, at the time of starting this task (iteration #0), the only n8n workflow that's verified as working is ollama.json (the one used in the example above). Fixing workflows is definitely something you can decide to do in your iteration, but DO NOT attempt to fix the n8n resource. **NOTE:** Shared workflows must be defined in the project-level .vrooli/service.json file and `vrooli setup` called, if you want the workflow to be upserted AND set to active. For scenario-level workflows, they *should* be injected when you do `vrooli app start <scenario_name>` (after you `vrooli scenario convert <scenario_name>`), though this hasn't been tested. As a fallback, you can run the *generated app*'s `./scripts/manage.sh setup` or call the injection engine directly.
- When writing n8n workflows, follow the same high standards we use for the existing workflows in initialization/n8n/. This includes having both a webhook and manual trigger, with the manual trigger going into a node that provides defaults (the webhook trigger should NOT go through that node). It also includes KEEPING the variables like `service.n8n.url`, as our scripts convert those before injection. It also includes using the `ollama.json` workflow instead of calling ollama directly. Generally, using a node that runs bash cli commands to access our resources is better than using the resources' apis directly. I say this from experience, so please listen.
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

If all scenarios look pretty good (meaning structurally and also TESTED), you may consider adding a new scenario. For adding new scenarios, you may only pick from this list (some may have already been created by earlier iterations by the time you're reading this):
- stream-of-consciousness-analyzer to convert unstructured text/voice to organized notes. Organized by "campaign", where each one allows me to add notes and documents as context to guide the agent that organizes the thoughts
- agent-dashboard to manage all active agents (e.g. huginn, claude-code, agent-s2)
- ci-cd-healer to fix CI/CD pipelines (you are FORBIDDEN from testing this one or touching any of our CI/CD cod)
- git-manager to deal with deciding when to commit, how to split up the changes into commits if they're unrelated, writing commit messages, etc. (you are also FORBIDDEN from testing this one or touching our git-related code)
- deployment-manager to deal with deciding how to modify a generated app (e.g. using app-to-ios to build an ios version, setting up for Docker or Kubernetes deployment etc.) to prepare and deploy it to the customer
- code-sleuth for tracking down relevant code for tasks, receiving feedback on what was relevant, and learning from that over time
- test-genie for learning how to test different scenarios (and also vrooli as a whole), as well as learning best practices over time
- dependabot for scheduled or triggered code scanning of scenarios and Vrooli as a whole
- app-to-electron for learning how to convert generated apps to Electron so that they can be run as standalone Desktop apps on Windows and such. Note that all of these `app-to-` scenario types should assume that the app distributions will be stored in a platform/ folder, like we've done at the project-level of Vrooli as a demonstration
- app-to-ios
- app-to-android
- app-onboarding-manager for adding proper and professional onboarding pages and tutorials to apps. Should learn best practices and build templates over time
- survey-monkey for building and sharing surveys
- resume-and-job-assistant for building and improving resumes, looking for open roles, and proactively investigating if they're a good fit for the customer and building a cover letter tailored to the role and company
- palette-gen for building color palettes
- personal-relationship-manager to track information about your friends, such as their birthdays, notes, etc. Should send reminders for birthdays, and proactively search for relevant gifts
- n8n-workflow-generator for learning how to effectively build n8n workfows
- prompt-performance-evaluator for testing and improving prompts
- video-downloader for downloading videos from a URL.
- product-manager-agent for handling more bigger-picture product decisions that can be used to guide our coding agents and decide what to focus on. Should have web and research capabilities
- dream analyzer for acting as a dream journal, and also doing analysis on specific dreams and common themes
- life-coach would be similar to the product-manager-agent, but focused specificall on the customer's general life
- chore-tracking for tracking chores, automatic schedule for cleaning/maintenance/etc., and point system with cute, quirky UX
- nutrition-tracker for calories and macros, with meal suggestions based on previous entries
- app-scroller to scroll through generated apps like you're on TikTok. Would pick the best apps depending on the mode, so that you get a different experience during the weekend vs. work hours
- roi-fit-analysis for deep web financial research to determine what ideas have the best return on investment based on your available skills and resources
- seo-optimizer for improving the search engine optimization of generated apps
- local-info-scout for finding local information, such as "vegan restaurants in my area", "stores nearby that sell cat bowls", "recently foreclosed homes near bodies of water", etc.
- wedding-planner for all things wedding planning
- morning-vision-walk for talking to it and having it help me understand the current state of Vrooli and brainstorm together on the things we should get done today.
- competitor-change-monitor to track the websites/githubs/etc. of competitors, and alert on any relevant changes. Would likely use huginn with fallback to browserless, and another fallback to agent-s2
- picker-whell for fun random selection. API available for scenarios/workflows that need random selection
- make-it-vegan to research if foods are vegan, or find vegan alternatives/substitutes
- travel-map-filler for a fun visual way of tracking where you've travelled
- data-generator for building or scraping data required for some of these scenarios. Huginn, browserless, and agent-s2 support
- fall-foliage-explorer for visualizing (with accurate data) where fall foliage peaks are forecasted, with a time slider
- mind-maps for building (both manually and with AI) mind maps that can be semantically searched. Should be thought of as an important scenario to be leveraged by others, any time we need data related to a topic/campaign/project/etc. (terminology may differ depending on the scenario, but they're all the same concept) to be organized and searchable by AI or humans.
- password-manager as a general password manager. Could also expose auth-related features via API for other scenarios, such as password stength and password generator
- word-games (wordle, connections, 2048, etc.) just for leisure and demonstrating capabilities
- notes for local, AI-enabled note taking
- itinerary-tracker with virtual bag/suitcase (this would possibly be our first scenario with 3D, so I'd be interested to see how you do it)
- coding-challenges for learning how to code (could also double as a method to benchmark how well various AI models can code)
- saas-billing-hub for adding payments and subscriptions to scenarios. Includes admin dashboard to manage multiple saas revenues

If everything in that list has already been added, then we've made a ton of progress! The next best thing to do would be trying to convert a scenario to an app, running it, and fixing any issues that are preventing the app from fully starting or showing a valid UI. Remember, changes in *generated* apps are ephemeral - you may change the generated app code for quick, iterative debugging, but once you find a solution you MUST update the scenario for it to persist.

Remember, some scenarios are used to increase the power and capability of Vrooli itself, which means they can call each other using their clis. For example, morning-vision-walk would likely rely heavily on stream-of-consciousness-analyzer, product-manager-agent, and maybe even life-coach. Other scenarios are demonstrations of what we can do and apps that might generate revenue. This includes scenarios which are intended to be deployed as a SaaS app or to a customer on Upwork, as well as scenarios which would be used internally by businesses or individuals with a local Vrooli server. The possibilities are only bounded by our current set of resources, which should hopefully be improved soon (but NOT in this chat).

 At the first iteration (may not be true now), the only UI types generated apps had are windmill UIs and javascript ones. I prefer javascript ones, as they can be fully customized to fit the vibes of the scenario and the context in which it'll be used in. In fact, one of the goals of scenarios is to abstract and push out nearly every part of an app, so coding agents like yourself can focus on building beautiful single or few page apps. Feel free to make them stand out and be unique. I want them to feel fun and experimental, like the old internet days. Having a bunch of apps with the same blank layout does not spark joy.

Ultra think. You may use the web if needed.