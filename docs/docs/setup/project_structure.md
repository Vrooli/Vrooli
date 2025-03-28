# Project Structure
This is a high-level overview of the project structure of the Vrooli codebase. For more information on a specific directory. Check out each directory yourself to get a better understanding of what's in there.

.  
├── [assets]({{ config.extra.repo_code }}/assets) - Data displayed in docs  
├── [docs]({{ config.extra.repo_code }}/docs) - Data for documentation site, hosted at vrooli.com/docs  
└── [packages]({{ config.extra.repo_code }}/packages) - Core website code, in a monorepo setup  
├── [server]({{ config.extra.repo_code }}/packages/server) - The "behind the scenes" code  
│   └── [src]({{ config.extra.repo_code }}/packages/server/src)  
│   ├── [auth]({{ config.extra.repo_code }}/packages/server/src/auth) - Authentication-related code, such as cardano-serialization and API endpoint authentication  
│   ├── [db]({{ config.extra.repo_code }}/packages/server/src/db)  
│   │   ├── [migrations]({{ config.extra.repo_code }}/packages/server/src/db/migrations) - Database version control  
│   │   ├── [seeds]({{ config.extra.repo_code }}/packages/server/src/db/seeds) - For populating database with real or "mock" data  
│   │   └── [schema.prisma]({{ config.extra.repo_code }}/packages/server/src/db/schema.prisma) - Defines the database structure  
│   ├── [events]({{ config.extra.repo_code }}/packages/server/src/events) - Handles tracking awards, triggering notifications, and other   event-based actions
│   ├── [middleware]({{ config.extra.repo_code }}/packages/server/src/middleware) - Express middleware  
│   ├── [models]({{ config.extra.repo_code }}/packages/server/src/models) - Compositional components representing objects in the database  
│   ├── [notify]({{ config.extra.repo_code }}/packages/server/src/notify) - Everything related to notifications. Emailing, push, etc.  
│   └── [utils]({{ config.extra.repo_code }}/packages/server/src/utils) - Miscellaneous utility functions  
│   └── [package.json]({{ config.extra.repo_code }}/packages/server/package.json) - Dependencies and useful scripts  
├── [shared]({{ config.extra.repo_code }}/packages/shared) - Data shared between packages  
└── [ui]({{ config.extra.repo_code }}/packages/ui) - What the user sees  
├── [public]({{ config.extra.repo_code }}/packages/ui/public) - OpenGraph, Progressive Web App, Trusted Web Activity, and other metadata about the website  
└── [src]({{ config.extra.repo_code }}/packages/ui/src)  
├── [assets]({{ config.extra.repo_code }}/packages/ui/src/assets)  
    │   ├── [font]({{ config.extra.repo_code }}/packages/ui/src/assets/font) - Custom fonts  
    │   └── [img]({{ config.extra.repo_code }}/packages/ui/src/assets/img) - Images displayed on site, that aren't user-uploaded or from Open Graph  
    ├── [components]({{ config.extra.repo_code }}/packages/ui/src/components) - Reusable React components  
    ├── [forms]({{ config.extra.repo_code }}/packages/ui/src/forms) - User input forms  
    ├── [pages]({{ config.extra.repo_code }}/packages/ui/src/pages) - Website pages  
    └── [utils]({{ config.extra.repo_code }}/packages/ui/src/utils) - Miscellaneous utility functions  
└── [package.json]({{ config.extra.repo_code }}/packages/ui/package.json) - Dependencies and useful scripts  
└── [.env-example]({{ config.extra.repo_code }}/.env-example) - Environment variables  

