/**
 * Handles the moderation of content through pull requests and report suggestions.
 * 
 * Reports work like this: 
 * 1. A user reports some object with a reason and optional details. The reporter can delete their own report at any time.
 * 2. Other users (cannot be report creator or object owner) suggest a moderation action. These include:
 *      - Deleting the object
 *      - Marking the report as a false report
 *      - Hiding the object until a new version is published that fixes the issue
 *      - Marking the report as a non-issue
 *      - Suspending the object owner (for how long depends on previous suspensions)
 * 3. When enough suggestions are made (accounting for the reputation of the report creator and suggestors), 
 *   the report is automatically accepted and the object is moderated accordingly.
 * 4. Notifications are sent to the relevant users when a decision is made, and reputation scores are updated.
 * 
 * Pull requests work like this:
 * 1. A user creates a pull request for some object. If the request is to fix a report, the report 
 * must be attached to the pull request.
 * 2. If the pull request does not reference a report, and the object has an owner:
 *     - A notification is sent to the object owner to review the pull request, with the option to accept or reject it.
 * 2. If the pull request is references report or is on an object without an owner, users can vote on whether the pull request should be accepted or rejected.
 * 3. Once a pull request is accepted/rejected, a notification is sent to the requestor. The requestor's reputation is updated accordingly.
 */

export {}