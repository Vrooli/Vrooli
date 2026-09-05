# agentharness

`agentharness` is the resource-facing coding-agent policy surface. It owns
offline policy evaluation, permission-document projection, model discovery
commands, and coding-role commands. Catalog contracts consumed by the control
plane live in `cli-core/agentcatalog`.

The package is intentionally not scenario-adoptable: resource runtimes and
internal platform commands are its only declared consumers.
