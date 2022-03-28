![Vrooli Logo with motto](./docs/assets/readme-display.png)

A collaborative tool for visualizing and automating tasks

*Note: This README assumes you know the basics of how [Project Catalyst](https://projectcatalyst.org/resources/what-is-project-catalyst) works.*

**What is a visual work routine?** A *routine* is the process for completing a specific task. A *work routine* completes productive tasks, such as creating a business outline, deciding on a budget, and generating a proposal. A *visual work routine* is an interactive tool that guides users through the process of completing a task, with descriptions and guides along every step of the way.

**Why use Vrooli?** Not only does Vrooli keep you organized and focused on your goals, but it also provides the same guidance to the rest of the community. Work routines can be built on top of each other, which simplifies the process of creating ever more complex routines.

**How can Vrooli be used to inspire entrepreneurs?** The simple process of stepping through a routine gives entrepreneurs assurance that they are on the right track. Seeing a routine's progress also provides a sense of accomplishment and motivation. Complicated or incomplete sections of popular routines can become great business proposals.

**How can Vrooli be used to automate tasks?** Vrooli has visions of becoming the "glue" of the automated world. If you're using routines to plan everything, connecting data and computation is the next logical step. The [project roadmap](#roadmap) details the timeline for this functionality.

## Quick Links
- [White paper (plus long-term vision for Vrooli and the future of society)](https://docs.google.com/document/d/1BO3Q5ADavWKaK0RSdeDVXfYOHtBF-cMr7S-XxKiGJ8g/edit?usp=sharing)
- [Explaining the problem and our solution](https://matthalloran8.medium.com/the-next-generation-of-global-collaboration-a4839766e29e)
- [First project proposal](https://cardano.ideascale.com/a/dtd/Community-Made-Interactive-Guides/367058-48088)
- [Join the discussion on Discord!](https://discord.gg/VyrDFzbmmF)
- [Follow us on Twitter!](https://twitter.com/VrooliOfficial)

## Roadmap
 - Q1 2022 
    - Website launch. Users can create, comment, and vote on basic routines. 
    - Routines consist of a flowchart of steps. Each step can have a description and a list of useful resources.
 - Q2 2022
    - Routine visualizer improved.
    - Routines have the ability to reference other routines.
    - Routines can be created using the template of any existing routine.
    - "Request a Routine" section added.
 - Q3 2022
    - API fully defined for routine metadata. Supports all existing features, plus information required for data storage and automation.
    - Development started for connecting routines to specific user interfaces. This functionality allows for users to step through routines much easier, and is an important precursor for automation.
    - Integration with DIDs, to act as a reputation system.
- Q4 2022
    - Routines can connect to IPNS (similar to IPFS, but updatable) for data storage.
    - Ability for routines to trigger a smart contract.
    - Release of routine interface functionality.
- 2023 and beyond
    - Ability for users to create their own routine interfaces.
    - Decentralize all the things!
    - Continual improvements and bug fixes, to ensure Vrooli is as best as can be.

## Join the Team
Vrooli's vision is bright, but a lot of work needs to be done to get there. Please feel free to contribute to the project, however you can. All work is appreciated💙 

If you'd like to work with me on this project, or create your own proposal for a feature on the roadmap, don't hesitate to reach out! My contact links can be found [here](https://matthalloran.info).

## Development Stack
| Dependency  | Purpose  |  Version  |
|---|---|---|
| [ReactJS](https://reactjs.org/)  | UI  |  `^17.0.2` |
| [MaterialUI](https://material-ui.com/)  | UI Styling  |  `^5.0.0-beta.0`  |
| [Apollo](https://www.apollographql.com/)  | API |  `^2.25.0` |
| [ExpressJs](https://expressjs.com/)  |  Backend Server  | `^4.17.1` |
| [PostgreSQL](https://www.postgresql.org/)  | Database  | `postgres:13` |
| [Redis](https://redis.io/) | Task Queueing | `redis` |

## Directory Structure
* [docs](./docs) - Stores additional guides, besides this one.
    * [assets](./docs/assets) - Data displayed in docs 
* [packages](./packages) - Core website code, in a monorepo setup
    * [server](./packages/server) - The "behind the scenes" code
        * [src](./packages/server/src)
            * [auth](./packages/server/src/auth) - Authentication-related code, such as cardano-serialization and GraphQL authentication
            * [db](./packages/server/src/db)
                * [migrations](./packages/server/src/db/migrations) - Database version control
                * [seeds](./packages/server/src/db/seeds) - For populating database with real or "mock" data
                * [schema.prisma](./packages/server/src/db/schema.prisma) - Defines the database structure
            * [models](./packages/server/src/models) - Compositional components representing objects in the database
            * [schema](./packages/server/src/schema) - GraphQL schema
                * [index.ts](./packages/server/src/schema/index.ts) - Merges schema files into one executable schema
                * [root.ts](./packages/server/src/schema/root.ts) - Contains core type definitions and resolvers
            * [utils](./packages/server/src/utils) - Miscellaneous utility functions
            * [worker](./packages/server/src/worker) - Processes which can be run asynchronously using Redis
                * [email](./packages/server/src/worker/email) - Send emails
                * [sms](./packages/server/src/worker/sms) - Send text messages
        * [package.json](./packages/server/package.json) - Dependencies and useful scripts
    * [shared](./packages/shared) - Data shared between packages  
    * [ui](./packages/ui) - What the user sees
        * [.storybook](./packages/ui/.storybook) - Configuration for [Storybook](https://storybook.js.org/), which allows components to be edited without a working frontend
        * [public](./packages/ui/public) - OpenGraph, Progressive Web App, Trusted Web Activity, and other metadata about the website
        * [src](./packages/ui/src)
            * [assets](./packages/ui/src/assets)
                * [font](./packages/ui/src/assets/font) - Custom fonts
                * [img](./packages/ui/src/assets/img) - Images displayed on site, that aren't user-uploaded or from Open Graph
            * [components](./packages/ui/src/components) - Reusable React components
            * [forms](./packages/ui/src/forms) - User input forms
            * [graphql](./packages/ui/src/graphql)
                * [fragment](./packages/ui/src/graphql/fragment) - Fragments for reducing duplication of query and mutation return selections
                * [generated](./packages/ui/src/graphql/generated) - Code automatically generated from `yarn graphql-generate` script
                * [mutation](./packages/ui/src/graphql/mutation) - GraphQL mutations
                * [query](./packages/ui/src/graphql/query) - GraphQL queries
                * [utils](./packages/ui/src/graphql/utils) - GraphQL-specific utility functions
            * [pages](./packages/ui/src/pages) - Website pages
            * [utils](./packages/ui/src/utils) - Miscellaneous utility functions
        * [package.json](./packages/ui/package.json) - Dependencies and useful scripts
* [.env-example](./.env-example) - Environment variables. Rename to `.env` before starting

## Workflow setup
See [this guide](https://github.com/MattHalloran/ReactGraphQLTemplate#how-to-start) for setting up the development workflow and general development tips.

If you'd like to work with me on this project, or create your own proposal for a feature on the roadmap, don't hesitate to reach out! My contact links can be found [here](https://matthalloran.info).
