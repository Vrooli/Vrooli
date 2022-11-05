/**
 * Handles giving reputation to users when some action is performed.
 */


const ReportAcceptWeight = [5, -20]; // report creator gains 5, owner of reported object loses 10
const ReportRejectWeight = -3;
const PullRequestAcceptWeight = 10;
const PullRequestRejectWeight = -1;
const IssueResolveWeight = 10;
const QuestionAnswerWeight = 5;
const CreatePublicApiWeight = 2;
const CreatePublicOrganizationWeight = 2;
const CreatePublicProjectWeight = 2;
const CreatePublicRoutineWeight = 2;
const CreatePublicSmartContractWeight = 2;
const CreatePublicStandardWeight = 2;

/**
 * Generates the reputation which should be rewarded to a user for receiving a vote. 
 * Given v0 is the original vote count and v1 is the new vote count:
 * - If v0 === v1, returns 0
 * - If v0 is from -9 to 9 and v1 > v0, then the user gets 1 reputation
 * - If v0 is from -9 to 9 and v1 < v0, then the user loses 1 reputation
 * - If v0 is from -99 to -10 or 10 to 99 and v1 > v0, then the user gets 1 reputation if v1 is a multiple of 10, otherwise 0
 * - If v0 is from -99 to -10 or 10 to 99 and v1 < v0, then the user loses 1 reputation if v1 is a multiple of 10, otherwise 0
 * - If v0 is from -999 to -100 or 100 to 999 and v1 > v0, then the user gets 1 reputation if v1 is a multiple of 100, otherwise 0
 * - If v0 is from -999 to -100 or 100 to 999 and v1 < v0, then the user loses 1 reputation if v1 is a multiple of 100, otherwise 0
 * , and so on.
 */
export const reputationDeltaVote = (v0: number, v1: number): number => {
    if (v0 === v1) return 0;
    // Determine how many zeros follow the leading digit
    const magnitude = v0 === 0 ? 1 : Math.floor(Math.log10(Math.abs(v0))) ;
    // Use modulo to check if the new vote count is a multiple of 10^(magnitude)
    const isMultiple = v1 % Math.pow(10, magnitude) === 0;
    // If vote increased, give reputation if it's a multiple of 10^(magnitude)
    if (v1 > v0) return isMultiple ? 1 : 0;
    // Otherwise vote must have decreased. Lose reputation if it's a multiple of 10^(magnitude)
    return isMultiple ? -1 : 0;
}

/**
 * Generates the reputation which should be rewarded to a user for receiving a star.
 * Same as votes, but double the magnitude.
 */
export const reputationDeltaStar = (v0: number, v1: number): number => {
    return reputationDeltaVote(v0, v1) * 2;
}

/**
 * Generates the reputation which should be rewarded to a user for contributing to a report. 
 * Should receive a reputation point every 10 contributions
 * @param totalContributions The total number of contributions you've made to any report
 */
export const reputationDeltaReportContribute = (totalContributions: number): number => {
    return Math.floor(totalContributions / 10);
}