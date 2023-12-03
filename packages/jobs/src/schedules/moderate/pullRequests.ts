/**
 * Handles the moderation of content through pull requests. It works like this:
 * 1. A user creates a pull request for some object. If the request is to fix a report, the report 
 * must be attached to the pull request.
 * 2. If the pull request does not reference a report, and the object has an owner:
 *     - A notification is sent to the object owner to review the pull request, with the option to accept or reject it.
 * 2. If the pull request is references report or is on an object without an owner, users can vote on whether the pull request should be accepted or rejected.
 * 3. Once a pull request is accepted/rejected, a notification is sent to the requestor. The requestor's reputation is updated accordingly.
 */
export { };
//TODO
