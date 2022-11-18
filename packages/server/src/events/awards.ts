/**
 * Given a trigger and the trigger's data, determines if the user should receive an
 * award
 */

import { Notify } from "../notify";
import { PrismaType } from "../types";

/**
 * All award categories. An award must belong to only one category, 
 * and a category must have at least one award.
 */
export type AwardCategory = 'AccountAnniversary' |
    'AccountNew' |
    'ApiCreate' |
    'CommentCreate' |
    'IssueCreate' |
    'NoteCreate' |
    'ObjectStar' |
    'ObjectVote' |
    'OrganizationCreate' |
    'OrganizationJoin' |
    'PostCreate' |
    'ProjectCreate' |
    'PullRequestCreate' |
    'PullRequestComplete' |
    'QuestionAnswer' |
    'QuestionCreate' |
    'QuizPass' |
    'ReportEnd' |
    'ReportContribute' |
    'Reputation' |
    'RunRoutine' |
    'RunProject' |
    'RoutineCreate' |
    'SmartContractCreate' |
    'StandardCreate' |
    'Streak' |
    'UserInvite';

/**
 * Given an ordered list of numbers, returns the closest lower number in the list
 * @param num The number to find the closest lower number for
 * @param list The list of numbers to search, from lowest to highest
 * @returns The closest lower number in the list, or null if there is none
 */
function closestLower(num: number, list: number[]): number | null {
    for (let i = 0; i < list.length; i++) {
        if (list[i] > num)
            return list[i - 1] || null;
    }
    return null;
}

// Determines variant for awards. Example: 7-day streak, 100th routine completed, etc.
// If an award has a variant, returns the closest lower variant (i.e. the highest variant that's applicable)
export const awardVariants: { [key in AwardCategory]?: (count: number) => number | null } = {
    AccountAnniversary: (years: number) => years,
    // No variants for AccountNew
    AccountNew: () => null,
    Streak: (days: number) => closestLower(days, [7, 30, 100, 200, 365, 500, 750, 1000]),
    QuizPass: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000]),
    Reputation: (count: number) => closestLower(count, [10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]),
    ObjectStar: (count: number) => closestLower(count, [1, 100, 500]),
    ObjectVote: (count: number) => closestLower(count, [1, 100, 1000, 10000]),
    PullRequestCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500]),
    PullRequestComplete: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500]),
    ApiCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50]),
    CommentCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000]),
    IssueCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250]),
    NoteCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100]),
    OrganizationCreate: (count: number) => closestLower(count, [1, 2, 5, 10]),
    OrganizationJoin: (count: number) => closestLower(count, [1, 5, 10, 25]),
    PostCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000]),
    ProjectCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100]),
    QuestionAnswer: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000]),
    QuestionCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000]),
    ReportEnd: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100]),
    ReportContribute: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000]),
    RunRoutine: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]),
    RunProject: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000]),
    RoutineCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100, 250, 500, 1000]),
    SmartContractCreate: (count: number) => closestLower(count, [1, 5, 10, 25]),
    StandardCreate: (count: number) => closestLower(count, [1, 5, 10, 25, 50]),
    UserInvite: (count: number) => closestLower(count, [1, 5, 10, 25, 50, 100]),
};

/**
 * Helper function for generating name/description object
 */
const nad = (name: string | null, description: string | null) => ({ name, description });

/**
 * Maps award category/level to the award's name and description. Names should be interesting and unique.
 */
const awardNames: { [key in AwardCategory]: (count: number) => { name: string | null, description: string | null } } = {
    AccountAnniversary: (years: number) => nad(`${years}-${years === 1 ? 'year' : 'years'} anniversary`, `Be a member of the community for ${years} ${years === 1 ? 'year' : 'years'}`),
    AccountNew: () => nad('Baby steps', 'Create your account'),
    Streak: (days: number) => {
        if (days < 7) return nad(null, null);
        if (days < 30) return nad('Keep it up', `Complete a routine for 7 days in a row`);
        if (days < 100) return nad('Consistency is key', 'Complete a routine for 30 days in a row');
        if (days < 200) return nad('Streaking', 'Complete a routine for 100 days in a row');
        if (days < 365) return nad('Epic streak', 'Complete a routine for 200 days in a row');
        if (days < 500) return nad('Routine pro', 'Complete a routine for 365 days in a row');
        if (days < 750) return nad('Routine champion', 'Complete a routine for 500 days in a row');
        if (days < 1000) return nad('Routine legend', 'Complete a routine for 750 days in a row');
        return nad('Routine god', 'Complete a routine for 1000 days in a row');
    },
    QuizPass: (count: number) => {
        if (count < 5) return nad('Quiz taker', 'Complete a quiz');
        if (count < 10) return nad('Rising star', 'Complete 5 quizzes');
        if (count < 25) return nad('Smart cookie', 'Complete 10 quizzes');
        if (count < 50) return nad('Studious', 'Complete 25 quizzes');
        if (count < 100) return nad('Brainiac', 'Complete 50 quizzes');
        if (count < 250) return nad('Quiz pro', 'Complete 100 quizzes');
        if (count < 500) return nad('Quiz champion', 'Complete 250 quizzes');
        if (count < 1000) return nad('Quiz legend', 'Complete 500 quizzes');
        return nad('Quiz god', 'Complete 1000 quizzes');
    },
    Reputation: (count: number) => {
        if (count < 10) return nad(null, null);
        if (count < 25) return nad('Nice', 'Get 10 reputation points');
        if (count < 50) return nad('Pointy', 'Earn 25 reputation points');
        if (count < 100) return nad('Unsung hero', 'Earn 50 reputation points');
        if (count < 250) return nad('Big shot', 'Earn 100 reputation');
        if (count < 500) return nad('VIP', 'Earn 250 reputation');
        if (count < 1000) return nad('Rockstar', 'Earn 500 reputation');
        if (count < 2500) return nad('Reputation pro', 'Earn 1000 reputation');
        if (count < 5000) return nad('Reputation champion', 'Earn 2500 reputation');
        if (count < 10000) return nad('Reputation legend', 'Earn 5000 reputation');
        return nad('Reputation god', 'Earn 10000 reputation');
    },
    ObjectStar: (count: number) => {
        if (count < 100) return nad('Star gazer', 'Star an object');
        if (count < 500) return nad('For research purposes', 'Star 100 objects');
        return nad('Superstar', 'Star 500 objects');
    },
    ObjectVote: (count: number) => {
        if (count < 100) return nad('I voted', 'Vote on an object');
        if (count < 1000) return nad('Active voter', 'Vote on 100 objects');
        if (count < 10000) return nad('Opinionated', 'Vote on 1000 objects');
        return nad('Civic duty', 'Vote on 10000 objects');
    },
    PullRequestCreate: (count: number) => {
        if (count < 5) return nad('Making a difference', 'Create a pull request');
        if (count < 10) return nad('Good idea', 'Create 5 pull requests');
        if (count < 25) return nad('Contributor', 'Create 10 pull requests');
        if (count < 50) return nad('Super contributor', 'Create 25 pull requests');
        if (count < 100) return nad('Pull request pro', 'Create 50 pull requests');
        if (count < 250) return nad('Pull request champion', 'Create 100 pull requests');
        if (count < 500) return nad('Pull request legend', 'Create 250 pull requests');
        return nad('Pull request god', 'Create 500 pull requests');
    },
    PullRequestComplete: (count: number) => {
        if (count < 5) return nad('Problem solver', 'Complete a pull request');
        if (count < 10) return nad('Good job', 'Complete 5 pull requests');
        if (count < 25) return nad('Great job', 'Complete 10 pull requests');
        if (count < 50) return nad('Awesome job', 'Complete 25 pull requests');
        if (count < 100) return nad('Your opinions matter', 'Complete 50 pull requests');
        if (count < 250) return nad('Pull my finger', 'Complete 100 pull requests');
        if (count < 500) return nad('Pull it together', 'Complete 250 pull requests');
        return nad('How do you do it?', 'Complete 500 pull requests');
    },
    ApiCreate: (count: number) => {
        if (count < 5) return nad('/api/v1', 'Create an API');
        if (count < 10) return nad('/api/v2', 'Create 5 APIs');
        if (count < 25) return nad('/api/v3', 'Create 10 APIs');
        if (count < 50) return nad('/api/v4', 'Create 25 APIs');
        return nad('/api/v69', 'Create 50 APIs');
    },
    CommentCreate: (count: number) => {
        if (count < 5) return nad('Breaking the ice', 'Create a comment');
        if (count < 10) return nad('Small talk', 'Create 5 comments');
        if (count < 25) return nad('Chatty', 'Create 10 comments');
        if (count < 50) return nad('Interlocutor', 'Create 25 comments');
        if (count < 100) return nad('Comment pro', 'Create 50 comments');
        if (count < 250) return nad('Comment champion', 'Create 100 comments');
        if (count < 500) return nad('Comment legend', 'Create 250 comments');
        if (count < 1000) return nad('Comment god', 'Create 500 comments');
        return nad('Justin Y.', 'Create 1000 comments');
    },
    IssueCreate: (count: number) => {
        if (count < 5) return nad('Small problem', 'Create an issue');
        if (count < 10) return nad('I sense a flaw', 'Create 5 issues');
        if (count < 25) return nad('I spy', 'Create 10 issues');
        if (count < 50) return nad('Issue pro', 'Create 25 issues');
        if (count < 100) return nad('Issue champion', 'Create 50 issues');
        if (count < 250) return nad('Issue legend', 'Create 100 issues');
        return nad('Issue god', 'Create 250 issues');
    },
    NoteCreate: (count: number) => {
        if (count < 5) return nad('Note to self', 'Create a note');
        if (count < 10) return nad('Write that down', 'Create 5 notes');
        if (count < 25) return nad('Noted', 'Create 10 notes');
        if (count < 50) return nad('Better than sticky notes', 'Create 25 notes');
        if (count < 100) return nad('Scribe', 'Create 50 notes');
        return nad('Noteworthy', 'Create 100 notes');
    },
    OrganizationCreate: (count: number) => {
        if (count < 2) return nad('Organized', 'Create an organization');
        if (count < 5) return nad('Anotha one', 'Create 2 organizations');
        if (count < 10) return nad('Entrepreneur', 'Create 5 organizations');
        return nad('Elon Musk', 'Create 10 organizations');
    },
    OrganizationJoin: (count: number) => {
        if (count < 5) return nad('Participant', 'Join an organization');
        if (count < 10) return nad('Team player', 'Join 5 organizations');
        if (count < 25) return nad('Assemble!', 'Join 10 organizations');
        return nad('Teamwork makes the dream work', 'Join 25 organizations');
    },
    PostCreate: (count: number) => {
        if (count < 5) return nad('I have something to say', 'Create a post');
        if (count < 10) return nad('Hear ye', 'Create 5 posts');
        if (count < 25) return nad('Posty', 'Create 10 posts');
        if (count < 50) return nad('Public speaker', 'Create 25 posts');
        if (count < 100) return nad('Blogger', 'Create 50 posts');
        if (count < 250) return nad('Niche influencer', 'Create 100 posts');
        if (count < 500) return nad('Influencer', 'Create 250 posts');
        if (count < 1000) return nad('Major influencer', 'Create 500 posts');
        return nad('Mr. Beast', 'Create 1000 posts');
    },
    ProjectCreate: (count: number) => {
        if (count < 5) return nad('Big plans', 'Create a project');
        if (count < 10) return nad('Project manager', 'Create 5 projects');
        if (count < 25) return nad('Project pro', 'Create 10 projects');
        if (count < 50) return nad('Project champion', 'Create 25 projects');
        if (count < 100) return nad('Project legend', 'Create 50 projects');
        return nad('Project god', 'Create 100 projects');
    },
    QuestionAnswer: (count: number) => {
        if (count < 5) return nad('I can answer that', 'Answer a question');
        if (count < 10) return nad('Knowledgeable', 'Answer 5 questions');
        if (count < 25) return nad('Smartypants', 'Answer 10 questions');
        if (count < 50) return nad('Know-it-all', 'Answer 25 questions');
        if (count < 100) return nad('Answering machine', 'Answer 50 questions');
        if (count < 250) return nad('Answer pro', 'Answer 100 questions');
        if (count < 500) return nad('Answer champion', 'Answer 250 questions');
        if (count < 1000) return nad('Answer legend', 'Answer 500 questions');
        return nad('Answer god', 'Answer 1000 questions');
    },
    QuestionCreate: (count: number) => {
        if (count < 5) return nad('I have a question', 'Create a question');
        if (count < 10) return nad('Questionable', 'Create 5 questions');
        if (count < 25) return nad('Help pls', 'Create 10 questions');
        if (count < 50) return nad('Clarifier', 'Create 25 questions');
        if (count < 100) return nad('Assistance needed', 'Create 50 questions');
        if (count < 250) return nad('Question pro', 'Create 100 questions');
        if (count < 500) return nad('Question champion', 'Create 250 questions');
        if (count < 1000) return nad('Question legend', 'Create 500 questions');
        return nad('Question god', 'Create 1000 questions');
    },
    ReportEnd: (count: number) => {
        if (count < 5) return nad('Hall monitor', 'Create a report that passes');
        if (count < 10) return nad('Quality control', 'Create 5 reports that pass');
        if (count < 25) return nad('Report pro', 'Create 10 reports that pass');
        if (count < 50) return nad('Report champion', 'Create 25 reports that pass');
        if (count < 100) return nad('Report legend', 'Create 50 reports that pass');
        return nad('Report god', 'Create 100 reports that pass');
    },
    ReportContribute: (count: number) => {
        if (count < 5) return nad('Doing my part', 'Contribute to a report');
        if (count < 10) return nad('Helping out', 'Contribute to 5 reports');
        if (count < 25) return nad('Fixer upper', 'Contribute to 10 reports');
        if (count < 50) return nad('Maintenance crew', 'Contribute to 25 reports');
        if (count < 100) return nad('Inspector', 'Contribute to 50 reports');
        if (count < 250) return nad('Report pro', 'Contribute to 100 reports');
        if (count < 500) return nad('Report champion', 'Contribute to 250 reports');
        if (count < 1000) return nad('Report legend', 'Contribute to 500 reports');
        return nad('Report god', 'Contribute to 1000 reports');
    },
    RunRoutine: (count: number) => {
        if (count < 5) return nad('Hello, world!', 'Run a routine');
        if (count < 10) return nad('Productive', 'Run 5 routines');
        if (count < 25) return nad('Motivated', 'Run 10 routines');
        if (count < 50) return nad('Diligent', 'Run 25 routines');
        if (count < 100) return nad('Hard worker', 'Run 50 routines');
        if (count < 250) return nad('Routinely routining', 'Run 100 routines');
        if (count < 500) return nad('Sheeesh', 'Run 250 routines');
        if (count < 1000) return nad('Busy bee', 'Run 500 routines');
        if (count < 2500) return nad('Routine pro', 'Run 1000 routines');
        if (count < 5000) return nad('Routine champion', 'Run 2500 routines');
        if (count < 10000) return nad('Routine legend', 'Run 5000 routines');
        return nad('Routine god', 'Run 10000 routines');
    },
    RunProject: (count: number) => {
        if (count < 5) return nad('I did a learn', 'Run a project');
        if (count < 10) return nad('Elementary', 'Run 5 projects');
        if (count < 25) return nad('Good noodle', 'Run 10 projects');
        if (count < 50) return nad('Smarticle particle', 'Run 25 projects');
        if (count < 100) return nad('Scholar', 'Run 50 projects');
        if (count < 250) return nad('Learning pro', 'Run 100 projects');
        if (count < 500) return nad('Learning champion', 'Run 250 projects');
        if (count < 1000) return nad('Learning legend', 'Run 500 projects');
        return nad('Learning god', 'Run 1000 projects');
    },
    RoutineCreate: (count: number) => {
        if (count < 5) return nad('Getting started', 'Create a routine');
        if (count < 10) return nad('Routine rookie', 'Create 5 routines');
        if (count < 25) return nad('Routine enthusiast', 'Create 10 routines');
        if (count < 50) return nad('Routine lover', 'Create 25 routines');
        if (count < 100) return nad('Routine fanatic', 'Create 50 routines');
        if (count < 250) return nad('Routine pro', 'Create 100 routines');
        if (count < 500) return nad('Routine champion', 'Create 250 routines');
        if (count < 1000) return nad('Routine legend', 'Create 500 routines');
        return nad('Routine god', 'Create 1000 routines');
    },
    SmartContractCreate: (count: number) => {
        if (count < 5) return nad('Contractor', 'Create a smart contract');
        if (count < 10) return nad('dApp developer', 'Create 5 smart contracts');
        if (count < 25) return nad('Love to see it', 'Create 10 smart contracts');
        return nad('The hero we need', 'Create 25 smart contracts');
    },
    StandardCreate: (count: number) => {
        if (count < 5) return nad('You get a standard', 'Create a standard');
        if (count < 10) return nad('Standards pro', 'Create 5 standards');
        if (count < 25) return nad('Standards champion', 'Create 10 standards');
        if (count < 50) return nad('Standards legend', 'Create 25 standards');
        return nad('Standards god', 'Create 50 standards');
    },
    UserInvite: (count: number) => {
        if (count < 5) return nad('Spread the word', 'Invited user joined the platform');
        if (count < 10) return nad('Word of mouth', '5 invited users joined the platform');
        if (count < 25) return nad('Popular', '10 invited users joined the platform');
        if (count < 50) return nad('We love you', '25 invited users joined the platform');
        if (count < 100) return nad('You are the best', '50 invited users joined the platform');
        return nad('You are the bestest', '100 invited users joined the platform');
    },
}

/**
 * Checks if a user should receive an award
 * @param awardCategory The award category
 * @param previousCount The previous count of the award category
 * @param currentCount The current count of the award category
 * @returns True if the user should receive the award
 */
const shouldAward = (awardCategory: AwardCategory, previousCount: number, currentCount: number): boolean => {
    const variant = awardVariants[awardCategory];
    if (!variant)
        return false;

    const previousVariant = variant(previousCount);
    const currentVariant = variant(currentCount);
    return previousVariant !== currentVariant;
}

/**
 * Handles tracking awards for a user. If a new award is earned, a notification
 * can be sent to the user (push or email)
 */
export const Award = (prisma: PrismaType, userId: string) => ({
    /**
     * Upserts an award into the database. If the award progress reaches a new goal,
     * the user is notified
     * @param category The category of the award
     * @param newProgress The new progress of the award
     */
    update: async (category: AwardCategory, newProgress: number) => {
        // Upsert the award into the database, with progress incremented
        // by the new progress
        const award = await prisma.award.upsert({
            where: { userId_category: { userId, category } },
            update: { progress: { increment: newProgress } },
            create: { userId, category, progress: newProgress },
        });
        // Check if user should receive a new award (i.e. the progress has put them
        // into a new award tier)
        const isNewTier = shouldAward(category, award.progress - newProgress, award.progress);
        if (isNewTier) {
            // Send a notification to the user
            const { name, description } = awardNames[category](award.progress);
            if (name && description) Notify(prisma, userId).pushAward(name, description);
            // Set "timeCurrentTierCompleted" to the current time
            await prisma.award.update({
                where: { userId_category: { userId, category } },
                data: { timeCurrentTierCompleted: new Date() },
            });
        }
        return award;
    },
})