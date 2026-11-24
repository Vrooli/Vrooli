# Agent Manager client

`agentmanager-client-go` shares Agent Manager service discovery and its generated
Connect client across scenario APIs. `Client.Connect` resolves the Agent Manager
address once per operation group and returns the common profile/task/run methods.
Scenario policy and domain-specific request composition remain with each caller.

The first migrated consumer is Content Desk. Its editorial prompt and allowed
actions stay in Content Desk; transport discovery and the Connect procedure
calls are shared here.
