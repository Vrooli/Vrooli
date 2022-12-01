# Database Design
Prisma can sometimes delete comments in the schema.prisma file, so this document provides information about the database design.


## General
- `citext` - Provides case-insensitive text search. Especially useful for emails.
- `stars`/`views` - Stars and views are stored as integers, so we don't have to count the number of relationships every time we want these common fields.
- `lanugage` - Languages are stored using their ISO 639-2 language code
- `forks` - Forks are used to suggest changes to a versioned object. When a fork is created, it initializes new root (i.e. not versioned) data, and copies the parent version's data

## api
- Stores information about external APIs, such as their name, description, details (like description but longer and supports markdown), version, link, type (GraphQL, OpenAPIv2, etc.), tags, link, schema, and more.
- Unique to link
- Schema stored as a stringified JSON object, either in GraphQL schema documentation or OpenAPIv3 format (or maybe more types)

## api_key
API keys for accessing Vrooli's API. Not to be confused with the api table, which specifies information about external APIs listed on Vrooli.

## comment
Comments can be submitted by either a user or an organization. They can be associated with a project, routine, or standard.  
- `userId` - Comment posted BY a user
- `organizationId` - Comment posted BY an organization
- `projectId` - Comment posted TO a project
- `routineId` - Comment posted TO a routine
- `standardId` - Comment posted TO a standard

## handle
Handles refer to ADA Handles


## node
Nodes describe the structure of a routine orchestration. Nodes come in 4 varieties:  
1. END - A node that ends the routine. Must not have a next node.
2. REDIRECT - A redirect to a node earlier in the routine. These are created automatically when someone makes a link to an earlier object
3. ROUTINE_LIST - A list of routines
4. START - The start node

- `columnIndex`/`rowIndex` - Specify the location of the node, so it can be displayed in the same place on the screen as it was when it was last saved.


## node_link
Links two nodes together. Any nodes that are not linked can still be associated with a routine, but they will be put in an "unlinked" state and the routine will not be able to be run


## node_link_when
A specific condition for a link to be available. If all links fail their conditions, then the user cannot proceed.


## organization
An organization is any group of one or more users that work together to accomplish a common goal/goals. In the crypto world, these are preferably Decentralized Autonomous Organizations (DAOs). But any organization can use this platform.
- `isOpenToNewMembers` - True if looking for open-source contributors or team members, paid or unpaid  

## project
A project is "owned" by an organization and/or user.


## resource
A resource is a link or address that provides context to whatever object it is associated with. This may be a user's social media links, a project's website, a routine's instruction video, etc.

A resource can be of the following types:  
- AdaHandle - Must start with "$". See https://adahandle.com/  
- Cardano RECEIVE address - Must start with "addr:"  
- An IPFS address - Must start with "ipfs:"  
- A typical url - Must start with "http://", "https://", or "www."  


## routine
The most important object in the system. A routine is a way to describe the process for completing some task. It may consist of subroutines, which are themselves routines.    

A routine is owned by either a user or organization. It can be transferred to another user or organization.

- `complexity` - complexity = 1 + (complexity of each subroutine) along the longest path
- `isAutomatable` - Indicates if the routine can be run automatically, if all conditions are met
- `isComplete` - If the routine is complete, or in a working state. Determined by the user
- `isInternal` -  Indicates if the routine should appear in searches, or if it is only meant to be a subnode of another routine
- `simplicity` - Opposite of completixy - 1 + (complexity of each subroutine) along the shortest path
- `timesStarted` - Count of times the routine was started. Includes times started as subroutine, but excludes runs by the owner
- `timesCompleted` - Count of times the routine was completed. Includes times completed as subroutine, but excludes runs by the owner


## routine_input_translation
NOTE: Name is not translated, as it is used as an identifier to query data the user has entered.


## routine_output_translation
NOTE: Name is not translated, as it is used as an identifier to query data the routine has generated.


## run
Stores information about individual runs of a routine. 

Can be multiple per routine/user combination.

- `completedComplexity` - Complexity of routine which was already completed. Used to measure progress more accurately than just the number of nodes completed.
- `contextSwitches` - Measures context switches. 0 means that the routine was executed completely without leaving
- `timeElapsed` - Time spent working on the run, in seconds

## run_step
Individual step of a run. Used to store progress, and additional metrics for each step.

Steps may be executed multiple times (i.e. loops), so duplicate steps are allowed.

- `order` - Execution order of the step. Routines don't need to be completed linearly 


## TODO scheduling runs
Schedules a routine to run at a specific time - once or repeatedly.  

Can be triggered based on time or conditions.


## standard
Data standard for a routine input or output

A user or organization cannot update a standard once it has been published (except for its description and tags). Therefore, they must create the standard with a new version number.

NOTE: standard name is not translated because it is used as a unique identifier. Think of standard names like CIP proposals. There isn't a translation for CIP-0030 - that's just its name

- `isInternal` - Indicates if the standard should appear in searches. Internal standards are not deleted, and have to creator. This make routine duplication easier, and opens the possibility to suggesting popular internal standard structures when users are creating them.

## standard_translation
- `jsonVariable` - If standard is a JSON type, then it may contain variables that have labels and helper text


## star
Objects can only be starred by a user, not an organization.

Stars can be applied to organizations, projects, routines, standards, tags, and users


## tag
Tags are used to categorize an object.


## user_tag_hidden
Allows users to not only hide tags from their searches, but also to blur them out.


## wallet
- `stakingAddress` - Synonymous with reward address
- `publicAddress` - Public address of wallet. Not required because there can be many public addresses for a single wallet
- `name` - User-defined name for wallet


## ResourceUsedFor
Examples of correct use for each resource type:  
- `Community` - Discord invite link
- `Context` - Pitch video link
- `Developer` - GitHub readme link with an alternative approach to completing the routine
- `Donation` - Patreon link
- `ExternalService` - Fill out form link
- `Feed` - Social media feed
- `Install` - Install a required program link
- `Learning` - White Paper link
- `Notes` - Notion board, notes app
- `OfficialWebsite` - Website or Github link
- `Proposal` - IdeaScale link
- `Related` - Anything that's related to the project, but doesn't fit into any of the other categories
- `Researching` - Knowledge base, research paper
- `Scheduling` - Google calendar, tasking app
- `Social` - Twitter, Facebook, Instagram
- `Tutorial` - Youtube video link


## ResourceListUsedFor
- `Custom` - User-created resource list, which can be deleted
- `Display` - e.g. Oranization view, Profile page
- `Learn` - Learn dashboard
- `Research` - Research dashboard
- `Develop` - Develop dashboard


## RunStatus
- `Scheduled` - Routine is scheduled to run
- `InProgress` - Routine is currently running
- `Completed` - Routine has completed running, and was successful
- `Failed` - Routine has completed running, but failed
- `Cancelled` - Routine was cancelled


## RunStepStatus
Run steps are only created as the routine is being run, so there is no scheduling.  
- `InProgress` - Step is currently running
- `Completed` - Step has completed running
- `Skipped` - Step was skipped. Only allowed if step is optional