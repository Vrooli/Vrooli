# Routine Complexity

The difficulty of a routine is useful to know. Not only does it make the process of finding routines you're willing to run easier, but it also quantifies how accessible a routine is. The less complicated a routine can be made, the better it will be at streamlining workflows. To measure this, we've identified three strategies:

- Storing logs of runs
- Measuring the structure of a routine
- Estimating your expertise/experience

## Logs
Storing logs is not only useful for keeping track of your own active runs. We can query the amount of times a routine was completed over a time interval, to determine its popularity. We also store the time it took to complete each step, which can be used to estimate the difficulty of a routine (both overall and by step). Further, we also track the amount of times a routine was stopped and resumed, as a way to measure context switching.

## Structure
A routine's structure is useful for measuring how simple/complex it is. To accomplish this, we treat a routine as a directed, cyclic, and weighted graph, and calculate its shortest and longest paths. Each edge is weighted by the number of inputs of the node before it, plus 1 if there was a decision to be made.

## Experience
This is something to do in the future. It would be useful if we could use the history of other routines you've completed to estimate if you'd find a routine easy to run. Let's say you're looking at a routine with tags A, B, and C. You've completed 3 distinct routines in the past with tag A, 10 with tag B, and 2 with tag C. We can use this information to estimate how well you know the topics associated with the routine.

Further in the future, it would be awesome if we could recommend a user to have certain credentials or complete certain Learn routines to be able to run a routine. It would also be cool if organizations could make Learn routines that users must complete before joining their organizations.