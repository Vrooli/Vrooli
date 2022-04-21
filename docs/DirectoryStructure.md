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