import { toMutation, toQuery, toSearch } from './utils';

export const endpoints = {
    api: () => {
        const { apiPartial } = require('./partial/api');
        return {
            findOne: toQuery('api', 'FindByIdInput', apiPartial, 'full'),
            findMany: toQuery('apis', 'ApiSearchInput', ...toSearch(apiPartial)),
            create: toMutation('apiCreate', 'ApiCreateInput', apiPartial, 'full'),
            update: toMutation('apiUpdate', 'ApiUpdateInput', apiPartial, 'full'),
        }
    },
    apiKey: () => {
        const { apiKeyPartial } = require('./partial/apiKey');
        const { successPartial } = require('./partial/success');
        return {
            create: toMutation('apiKeyCreate', 'ApiKeyCreateInput', apiKeyPartial, 'full'),
            update: toMutation('apiKeyUpdate', 'ApiKeyUpdateInput', apiKeyPartial, 'full'),
            deleteOne: toMutation('apiKeyDeleteOne', 'ApiKeyDeleteOneInput', successPartial, 'full'),
            validate: toMutation('apiKeyValidate', 'ApiKeyValidateInput', apiKeyPartial, 'full'),
        }
    },
    apiVersion: () => {
        const { apiVersionPartial } = require('./partial/apiVersion');
        return {
            findOne: toQuery('apiVersion', 'FindByIdInput', apiVersionPartial, 'full'),
            findMany: toQuery('apiVersions', 'ApiVersionSearchInput', ...toSearch(apiVersionPartial)),
            create: toMutation('apiVersionCreate', 'ApiVersionCreateInput', apiVersionPartial, 'full'),
            update: toMutation('apiVersionUpdate', 'ApiVersionUpdateInput', apiVersionPartial, 'full'),
        }
    },
    auth: () => {
        const { sessionPartial } = require('./partial/session');
        const { successPartial } = require('./partial/success');
        const { walletCompletePartial } = require('./partial/walletComplete');
        return {
            emailLogIn: toMutation('emailLogIn', 'EmailLogInInput', sessionPartial, 'full'),
            emailSignUp: toMutation('emailSignUp', 'EmailSignUpInput', sessionPartial, 'full'),
            emailRequestPasswordChange: toMutation('emailRequestPasswordChange', 'EmailRequestPasswordChangeInput', successPartial, 'full'),
            emailResetPassword: toMutation('emailResetPassword', 'EmailResetPasswordInput', sessionPartial, 'full'),
            guestLogIn: toMutation('guestLogIn', null, sessionPartial, 'full'),
            logOut: toMutation('logOut', 'LogOutInput', sessionPartial, 'full'),
            validateSession: toMutation('validateSession', 'ValidateSessionInput', sessionPartial, 'full'),
            switchCurrentAccount: toMutation('switchCurrentAccount', 'SwitchCurrentAccountInput', sessionPartial, 'full'),
            walletInit: toMutation('walletInit', 'WalletInitInput'),
            walletComplete: toMutation('walletComplete', 'WalletCompleteInput', walletCompletePartial, 'full'),
        }
    },
    comment: () => {
        const { commentPartial, commentThreadPartial } = require('./partial/comment');
        return {
            findOne: toQuery('comment', 'FindByIdInput', commentPartial, 'full'),
            findMany: toQuery('comments', 'CommentSearchInput', commentThreadPartial, 'full'),
            create: toMutation('commentCreate', 'CommentCreateInput', commentPartial, 'full'),
            update: toMutation('commentUpdate', 'CommentUpdateInput', commentPartial, 'full'),
        }
    },
    copy: () => {
        const { copyResultPartial } = require('./partial/copyResult');
        return {
            copy: toMutation('copy', 'CopyInput', copyResultPartial, 'full'),
        }
    },
    deleteOneOrMany: () => {
        const { successPartial } = require('./partial/success');
        const { countPartial } = require('./partial/count');
        return {
            deleteOne: toMutation('deleteOne', 'DeleteOneInput', successPartial, 'full'),
            deleteMany: toMutation('deleteMany', 'DeleteManyInput', countPartial, 'full'),
        }
    },
    email: () => {
        const { emailPartial } = require('./partial/email');
        const { successPartial } = require('./partial/success');
        return {
            create: toMutation('emailCreate', 'PhoneCreateInput', emailPartial, 'full'),
            verify: toMutation('sendVerificationEmail', 'SendVerificationEmailInput', successPartial, 'full'),
        }
    },
    feed: () => {
        const { developResultPartial, learnResultPartial, popularResultPartial, researchResultPartial } = require('./partial/feed');
        return {
            popular: toQuery('popular', 'PopularInput', popularResultPartial, 'list'),
            learn: toQuery('learn', null, learnResultPartial, 'list'),
            research: toQuery('research', null, researchResultPartial, 'list'),
            develop: toQuery('develop', null, developResultPartial, 'list'),
        }
    },
    history: () => {
        const { historyResultPartial } = require('./partial/historyResult');
        return {
            history: toQuery('history', 'HistoryInput', historyResultPartial, 'list'),
        }
    },
    issue: () => {
        const { issuePartial } = require('./partial/issue');
        return {
            findOne: toQuery('issue', 'FindByIdInput', issuePartial, 'full'),
            findMany: toQuery('issues', 'IssueSearchInput', ...toSearch(issuePartial)),
            create: toMutation('issueCreate', 'IssueCreateInput', issuePartial, 'full'),
            update: toMutation('issueUpdate', 'IssueUpdateInput', issuePartial, 'full'),
            close: toMutation('issueClose', 'IssueCloseInput', issuePartial, 'full'),
        }
    },
    label: () => {
        const { labelPartial } = require('./partial/label');
        return {
            findOne: toQuery('label', 'FindByIdInput', labelPartial, 'full'),
            findMany: toQuery('labels', 'LabelSearchInput', ...toSearch(labelPartial)),
            create: toMutation('labelCreate', 'LabelCreateInput', labelPartial, 'full'),
            update: toMutation('labelUpdate', 'LabelUpdateInput', labelPartial, 'full'),
        }
    },
    meeting: () => {
        const { meetingPartial } = require('./partial/meeting');
        return {
            findOne: toQuery('meeting', 'FindByIdInput', meetingPartial, 'full'),
            findMany: toQuery('meetings', 'MeetingSearchInput', ...toSearch(meetingPartial)),
            create: toMutation('meetingCreate', 'MeetingCreateInput', meetingPartial, 'full'),
            update: toMutation('meetingUpdate', 'MeetingUpdateInput', meetingPartial, 'full'),
        }
    },
    meetingInvite: () => {
        const { meetingInvitePartial } = require('./partial/meetingInvite');
        return {
            findOne: toQuery('meetingInvite', 'FindByIdInput', meetingInvitePartial, 'full'),
            findMany: toQuery('meetingInvites', 'MeetingInviteSearchInput', ...toSearch(meetingInvitePartial)),
            create: toMutation('meetingInviteCreate', 'MeetingInviteCreateInput', meetingInvitePartial, 'full'),
            update: toMutation('meetingInviteUpdate', 'MeetingInviteUpdateInput', meetingInvitePartial, 'full'),
            accept: toMutation('meetingInviteAccept', 'FindByIdInput', meetingInvitePartial, 'full'),
            decline: toMutation('meetingInviteDecline', 'FindByIdInput', meetingInvitePartial, 'full'),
        }
    },
    member: () => {
        const { memberPartial } = require('./partial/member');
        return {
            findOne: toQuery('member', 'FindByIdInput', memberPartial, 'full'),
            findMany: toQuery('members', 'MemberSearchInput', ...toSearch(memberPartial)),
            update: toMutation('memberUpdate', 'MemberUpdateInput', memberPartial, 'full'),
        }
    },
    memberInvite: () => {
        const { memberInvitePartial } = require('./partial/memberInvite');
        return {
            findOne: toQuery('memberInvite', 'FindByIdInput', memberInvitePartial, 'full'),
            findMany: toQuery('memberInvites', 'MemberInviteSearchInput', ...toSearch(memberInvitePartial)),
            create: toMutation('memberInviteCreate', 'MemberInviteCreateInput', memberInvitePartial, 'full'),
            update: toMutation('memberInviteUpdate', 'MemberInviteUpdateInput', memberInvitePartial, 'full'),
            accept: toMutation('memberInviteAccept', 'FindByIdInput', memberInvitePartial, 'full'),
            decline: toMutation('memberInviteDecline', 'FindByIdInput', memberInvitePartial, 'full'),
        }
    },
    node: () => {
        const { nodePartial } = require('./partial/node');
        return {
            create: toMutation('nodeCreate', 'NodeCreateInput', nodePartial, 'full'),
            update: toMutation('nodeUpdate', 'NodeUpdateInput', nodePartial, 'full'),
        }
    },
    note: () => {
        const { notePartial } = require('./partial/note');
        return {
            findOne: toQuery('note', 'FindByIdInput', notePartial, 'full'),
            findMany: toQuery('notes', 'NoteSearchInput', ...toSearch(notePartial)),
            create: toMutation('noteCreate', 'NoteCreateInput', notePartial, 'full'),
            update: toMutation('noteUpdate', 'NoteUpdateInput', notePartial, 'full'),
        }
    },
    noteVersion: () => {
        const { noteVersionPartial } = require('./partial/noteVersion');
        return {
            findOne: toQuery('noteVersion', 'FindByIdInput', noteVersionPartial, 'full'),
            findMany: toQuery('noteVersions', 'NoteVersionSearchInput', ...toSearch(noteVersionPartial)),
            create: toMutation('noteVersionCreate', 'NoteVersionCreateInput', noteVersionPartial, 'full'),
            update: toMutation('noteVersionUpdate', 'NoteVersionUpdateInput', noteVersionPartial, 'full'),
        }
    },
    notification: () => {
        const { notificationPartial, notificationSettingsPartial } = require('./partial/notification');
        const { successPartial } = require('./partial/success');
        const { countPartial } = require('./partial/count');
        return {
            findOne: toQuery('notification', 'FindByIdInput', notificationPartial, 'full'),
            findMany: toQuery('notifications', 'NotificationSearchInput', ...toSearch(notificationPartial)),
            markAsRead: toMutation('notificationMarkAsRead', 'FindByIdInput', successPartial, 'full'),
            update: toMutation('notificationMarkAllAsRead', null, countPartial, 'full'),
            settingsUpdate: toMutation('notificationSettingsUpdate', 'NotificationSettingsUpdateInput', notificationSettingsPartial, 'full'),
        }
    },
    notificationSubscription: () => {
        const { notificationSubscriptionPartial } = require('./partial/notificationSubscription');
        return {
            findOne: toQuery('notificationSubscription', 'FindByIdInput', notificationSubscriptionPartial, 'full'),
            findMany: toQuery('notificationSubscriptions', 'NotificationSubscriptionSearchInput', ...toSearch(notificationSubscriptionPartial)),
            create: toMutation('notificationSubscriptionCreate', 'NotificationSubscriptionCreateInput', notificationSubscriptionPartial, 'full'),
            update: toMutation('notificationSubscriptionUpdate', 'NotificationSubscriptionUpdateInput', notificationSubscriptionPartial, 'full'),
        }
    },
    organization: () => {
        const { organizationPartial } = require('./partial/organization');
        return {
            findOne: toQuery('organization', 'FindByIdInput', organizationPartial, 'full'),
            findMany: toQuery('organizations', 'OrganizationSearchInput', ...toSearch(organizationPartial)),
            create: toMutation('organizationCreate', 'OrganizationCreateInput', organizationPartial, 'full'),
            update: toMutation('organizationUpdate', 'OrganizationUpdateInput', organizationPartial, 'full'),
        }
    },
    phone: () => {
        const { phonePartial } = require('./partial/phone');
        const { successPartial } = require('./partial/success');
        return {
            create: toMutation('phoneCreate', 'PhoneCreateInput', phonePartial, 'full'),
            update: toMutation('sendVerificationText', 'SendVerificationTextInput', successPartial, 'full'),
        }
    },
    post: () => {
        const { postPartial } = require('./partial/post');
        return {
            findOne: toQuery('post', 'FindByIdInput', postPartial, 'full'),
            findMany: toQuery('posts', 'PostSearchInput', ...toSearch(postPartial)),
            create: toMutation('postCreate', 'PostCreateInput', postPartial, 'full'),
            update: toMutation('postUpdate', 'PostUpdateInput', postPartial, 'full'),
        }
    },
    project: () => {
        const { projectPartial } = require('./partial/project');
        return {
            findOne: toQuery('project', 'FindByIdInput', projectPartial, 'full'),
            findMany: toQuery('projects', 'ProjectSearchInput', ...toSearch(projectPartial)),
            create: toMutation('projectCreate', 'ProjectCreateInput', projectPartial, 'full'),
            update: toMutation('projectUpdate', 'ProjectUpdateInput', projectPartial, 'full'),
        }
    },
    projectOrOrganization: () => {
        const { projectOrOrganizationPartial } = require('./partial/projectOrOrganization');
        return {
            findMany: toQuery('projectOrOrganizations', 'ProjectOrOrganizationSearchInput', ...toSearch(projectOrOrganizationPartial)),
        }
    },
    projectOrRoutine: () => {
        const { projectOrRoutinePartial } = require('./partial/projectOrRoutine');
        return {
            findMany: toQuery('projectOrRoutines', 'ProjectOrRoutineSearchInput', ...toSearch(projectOrRoutinePartial)),
        }
    },
    projectVersion: () => {
        const { projectVersionPartial } = require('./partial/projectVersion');
        return {
            findOne: toQuery('projectVersion', 'FindByIdInput', projectVersionPartial, 'full'),
            findMany: toQuery('projectVersions', 'ProjectVersionSearchInput', ...toSearch(projectVersionPartial)),
            create: toMutation('projectVersionCreate', 'ProjectVersionCreateInput', projectVersionPartial, 'full'),
            update: toMutation('projectVersionUpdate', 'ProjectVersionUpdateInput', projectVersionPartial, 'full'),
        }
    },
    pullRequest: () => {
        const { pullRequestPartial } = require('./partial/pullRequest');
        return {
            findOne: toQuery('pullRequest', 'FindByIdInput', pullRequestPartial, 'full'),
            findMany: toQuery('pullRequests', 'PullRequestSearchInput', ...toSearch(pullRequestPartial)),
            create: toMutation('pullRequestCreate', 'PullRequestCreateInput', pullRequestPartial, 'full'),
            update: toMutation('pullRequestUpdate', 'PullRequestUpdateInput', pullRequestPartial, 'full'),
            accept: toMutation('pullRequestAcdept', 'FindByIdInput', pullRequestPartial, 'full'),
            reject: toMutation('pullRequestReject', 'FindByIdInput', pullRequestPartial, 'full'),
        }
    },
    pushDevice: () => {
        const { pushDevicePartial } = require('./partial/pushDevice');
        return {
            findMany: toQuery('pushDevices', 'PushDeviceSearchInput', ...toSearch(pushDevicePartial)),
            create: toMutation('pushDeviceCreate', 'PushDeviceCreateInput', pushDevicePartial, 'full'),
            update: toMutation('pushDeviceUpdate', 'PushDeviceUpdateInput', pushDevicePartial, 'full'),
        }
    },
    question: () => {
        const { questionPartial } = require('./partial/question');
        return {
            findOne: toQuery('question', 'FindByIdInput', questionPartial, 'full'),
            findMany: toQuery('questions', 'QuestionSearchInput', ...toSearch(questionPartial)),
            create: toMutation('questionCreate', 'QuestionCreateInput', questionPartial, 'full'),
            update: toMutation('questionUpdate', 'QuestionUpdateInput', questionPartial, 'full'),
        }
    },
    questionAnswer: () => {
        const { questionAnswerPartial } = require('./partial/questionAnswer');
        return {
            findOne: toQuery('questionAnswer', 'FindByIdInput', questionAnswerPartial, 'full'),
            findMany: toQuery('questionAnswers', 'QuestionAnswerSearchInput', ...toSearch(questionAnswerPartial)),
            create: toMutation('questionAnswerCreate', 'QuestionAnswerCreateInput', questionAnswerPartial, 'full'),
            update: toMutation('questionAnswerUpdate', 'QuestionAnswerUpdateInput', questionAnswerPartial, 'full'),
            accept: toMutation('questionAnswerMarkAsAccepted', 'FindByIdInput', questionAnswerPartial, 'full'),
        }
    },
    quiz: () => {
        const { quizPartial } = require('./partial/quiz');
        return {
            findOne: toQuery('quiz', 'FindByIdInput', quizPartial, 'full'),
            findMany: toQuery('quizzes', 'QuizSearchInput', ...toSearch(quizPartial)),
            create: toMutation('quizCreate', 'QuizCreateInput', quizPartial, 'full'),
            update: toMutation('quizUpdate', 'QuizUpdateInput', quizPartial, 'full'),
        }
    },
    quizAttempt: () => {
        const { quizAttemptPartial } = require('./partial/quizAttempt');
        return {
            findOne: toQuery('quizAttempt', 'FindByIdInput', quizAttemptPartial, 'full'),
            findMany: toQuery('quizAttempts', 'QuizAttemptSearchInput', ...toSearch(quizAttemptPartial)),
            create: toMutation('quizAttemptCreate', 'QuizAttemptCreateInput', quizAttemptPartial, 'full'),
            update: toMutation('quizAttemptUpdate', 'QuizAttemptUpdateInput', quizAttemptPartial, 'full'),
        }
    },
    quizQuestion: () => {
        const { quizQuestionPartial } = require('./partial/quizQuestion');
        return {
            findOne: toQuery('quizQuestion', 'FindByIdInput', quizQuestionPartial, 'full'),
            findMany: toQuery('quizQuestions', 'QuizQuestionSearchInput', ...toSearch(quizQuestionPartial)),
        }
    },
    quizQuestionResponse: () => {
        const { quizQuestionResponsePartial } = require('./partial/quizQuestionResponse');
        return {
            findOne: toQuery('quizQuestionResponse', 'FindByIdInput', quizQuestionResponsePartial, 'full'),
            findMany: toQuery('quizQuestionResponses', 'QuizQuestionResponseSearchInput', ...toSearch(quizQuestionResponsePartial)),
            create: toMutation('quizQuestionResponseCreate', 'QuizQuestionResponseCreateInput', quizQuestionResponsePartial, 'full'),
            update: toMutation('quizQuestionResponseUpdate', 'QuizQuestionResponseUpdateInput', quizQuestionResponsePartial, 'full'),
        }
    },
    reminder: () => {
        const { reminderPartial } = require('./partial/reminder');
        return {
            findOne: toQuery('reminder', 'FindByIdInput', reminderPartial, 'full'),
            findMany: toQuery('reminders', 'ReminderSearchInput', ...toSearch(reminderPartial)),
            create: toMutation('reminderCreate', 'ReminderCreateInput', reminderPartial, 'full'),
            update: toMutation('reminderUpdate', 'ReminderUpdateInput', reminderPartial, 'full'),
        }
    },
    reminderList: () => {
        const { reminderListPartial } = require('./partial/reminderList');
        return {
            findOne: toQuery('reminderList', 'FindByIdInput', reminderListPartial, 'full'),
            findMany: toQuery('reminderLists', 'ReminderListSearchInput', ...toSearch(reminderListPartial)),
            create: toMutation('reminderListCreate', 'ReminderListCreateInput', reminderListPartial, 'full'),
            update: toMutation('reminderListUpdate', 'ReminderListUpdateInput', reminderListPartial, 'full'),
        }
    },
    report: () => {
        const { reportPartial } = require('./partial/report');
        return {
            findOne: toQuery('report', 'FindByIdInput', reportPartial, 'full'),
            findMany: toQuery('reports', 'ReportSearchInput', ...toSearch(reportPartial)),
            create: toMutation('reportCreate', 'ReportCreateInput', reportPartial, 'full'),
            update: toMutation('reportUpdate', 'ReportUpdateInput', reportPartial, 'full'),
        }
    },
    reportResponse: () => {
        const { reportResponsePartial } = require('./partial/reportResponse');
        return {
            findOne: toQuery('reportResponse', 'FindByIdInput', reportResponsePartial, 'full'),
            findMany: toQuery('reportResponses', 'ReportResponseSearchInput', ...toSearch(reportResponsePartial)),
            create: toMutation('reportResponseCreate', 'ReportResponseCreateInput', reportResponsePartial, 'full'),
            update: toMutation('reportResponseUpdate', 'ReportResponseUpdateInput', reportResponsePartial, 'full'),
        }
    },
    reputationHistory: () => {
        const { reputationHistoryPartial } = require('./partial/reputationHistory');
        return {
            findOne: toQuery('reputationHistory', 'FindByIdInput', reputationHistoryPartial, 'full'),
            findMany: toQuery('reputationHistories', 'ReputationHistorySearchInput', ...toSearch(reputationHistoryPartial)),
        }
    },
    resource: () => {
        const { resourcePartial } = require('./partial/resource');
        return {
            findOne: toQuery('resource', 'FindByIdInput', resourcePartial, 'full'),
            findMany: toQuery('resources', 'ResourceSearchInput', ...toSearch(resourcePartial)),
            create: toMutation('resourceCreate', 'ResourceCreateInput', resourcePartial, 'full'),
            update: toMutation('resourceUpdate', 'ResourceUpdateInput', resourcePartial, 'full'),
        }
    },
    resourceList: () => {
        const { resourceListPartial } = require('./partial/resourceList');
        return {
            findOne: toQuery('resourceList', 'FindByIdInput', resourceListPartial, 'full'),
            findMany: toQuery('resourceLists', 'ResourceListSearchInput', ...toSearch(resourceListPartial)),
            create: toMutation('resourceListCreate', 'ResourceListCreateInput', resourceListPartial, 'full'),
            update: toMutation('resourceListUpdate', 'ResourceListUpdateInput', resourceListPartial, 'full'),
        }
    },
    role: () => {
        const { rolePartial } = require('./partial/role');
        return {
            findOne: toQuery('role', 'FindByIdInput', rolePartial, 'full'),
            findMany: toQuery('roles', 'RoleSearchInput', ...toSearch(rolePartial)),
            create: toMutation('roleCreate', 'RoleCreateInput', rolePartial, 'full'),
            update: toMutation('roleUpdate', 'RoleUpdateInput', rolePartial, 'full'),
        }
    },
    routine: () => {
        const { routinePartial } = require('./partial/routine');
        return {
            findOne: toQuery('routine', 'FindByIdInput', routinePartial, 'full'),
            findMany: toQuery('routines', 'RoutineSearchInput', ...toSearch(routinePartial)),
            create: toMutation('routineCreate', 'RoutineCreateInput', routinePartial, 'full'),
            update: toMutation('routineUpdate', 'RoutineUpdateInput', routinePartial, 'full'),
        }
    },
    routineVersion: () => {
        const { routineVersionPartial } = require('./partial/routineVersion');
        return {
            findOne: toQuery('routineVersion', 'FindByIdInput', routineVersionPartial, 'full'),
            findMany: toQuery('routineVersions', 'RoutineVersionSearchInput', ...toSearch(routineVersionPartial)),
            create: toMutation('routineVersionCreate', 'RoutineVersionCreateInput', routineVersionPartial, 'full'),
            update: toMutation('routineVersionUpdate', 'RoutineVersionUpdateInput', routineVersionPartial, 'full'),
        }
    },
    runProject: () => {
        const { runProjectPartial } = require('./partial/runProject');
        const { countPartial } = require('./partial/count');
        return {
            findOne: toQuery('runProject', 'FindByIdInput', runProjectPartial, 'full'),
            findMany: toQuery('runProjects', 'RunProjectSearchInput', ...toSearch(runProjectPartial)),
            create: toMutation('runProjectCreate', 'RunProjectCreateInput', runProjectPartial, 'full'),
            update: toMutation('runProjectUpdate', 'RunProjectUpdateInput', runProjectPartial, 'full'),
            deleteAll: toMutation('runProjectDeleteAll', null, countPartial, 'full'),
            complete: toMutation('runProjectComplete', 'RunProjectCompleteInput', runProjectPartial, 'full'),
            cancel: toMutation('runProjectCancel', 'RunProjectCancelInput', runProjectPartial, 'full'),
        }
    },
    runProjectOrRunRoutine: () => {
        const { runProjectOrRunRoutinePartial } = require('./partial/runProjectOrRunRoutine');
        return {
            findMany: toQuery('runProjectOrRunRoutines', 'RunProjectOrRunRoutineSearchInput', ...toSearch(runProjectOrRunRoutinePartial)),
        }
    },
    runProjectSchedule: () => {
        const { runProjectSchedulePartial } = require('./partial/runProjectSchedule');
        return {
            findOne: toQuery('runProjectSchedule', 'FindByIdInput', runProjectSchedulePartial, 'full'),
            findMany: toQuery('runProjectSchedules', 'RunProjectScheduleSearchInput', ...toSearch(runProjectSchedulePartial)),
            create: toMutation('runProjectScheduleCreate', 'RunProjectScheduleCreateInput', runProjectSchedulePartial, 'full'),
            update: toMutation('runProjectScheduleUpdate', 'RunProjectScheduleUpdateInput', runProjectSchedulePartial, 'full'),
        }
    },
    runRoutine: () => {
        const { runRoutinePartial } = require('./partial/runRoutine');
        const { countPartial } = require('./partial/count');
        return {
            findOne: toQuery('runRoutine', 'FindByIdInput', runRoutinePartial, 'full'),
            findMany: toQuery('runRoutines', 'RunRoutineSearchInput', ...toSearch(runRoutinePartial)),
            create: toMutation('runRoutineCreate', 'RunRoutineCreateInput', runRoutinePartial, 'full'),
            update: toMutation('runRoutineUpdate', 'RunRoutineUpdateInput', runRoutinePartial, 'full'),
            deleteAll: toMutation('runRoutineDeleteAll', null, countPartial, 'full'),
            complete: toMutation('runRoutineComplete', 'RunRoutineCompleteInput', runRoutinePartial, 'full'),
            cancel: toMutation('runRoutineCancel', 'RunRoutineCancelInput', runRoutinePartial, 'full'),
        }
    },
    runRoutineInput: () => {
        const { runRoutineInputPartial } = require('./partial/runRoutineInput');
        return {
            findMany: toQuery('runRoutineInputs', 'RunRoutineInputSearchInput', ...toSearch(runRoutineInputPartial)),
        }
    },
    runRoutineSchedule: () => {
        const { runRoutineSchedulePartial } = require('./partial/runRoutineSchedule');
        return {
            findOne: toQuery('runRoutineSchedule', 'FindByIdInput', runRoutineSchedulePartial, 'full'),
            findMany: toQuery('runRoutineSchedules', 'RunRoutineScheduleSearchInput', ...toSearch(runRoutineSchedulePartial)),
            create: toMutation('runRoutineScheduleCreate', 'RunRoutineScheduleCreateInput', runRoutineSchedulePartial, 'full'),
            update: toMutation('runRoutineScheduleUpdate', 'RunRoutineScheduleUpdateInput', runRoutineSchedulePartial, 'full'),
        }
    },
    smartContract: () => {
        const { smartContractPartial } = require('./partial/smartContract');
        return {
            findOne: toQuery('smartContract', 'FindByIdInput', smartContractPartial, 'full'),
            findMany: toQuery('smartContracts', 'SmartContractSearchInput', ...toSearch(smartContractPartial)),
            create: toMutation('smartContractCreate', 'SmartContractCreateInput', smartContractPartial, 'full'),
            update: toMutation('smartContractUpdate', 'SmartContractUpdateInput', smartContractPartial, 'full'),
        }
    },
    smartContractVersion: () => {
        const { smartContractVersionPartial } = require('./partial/smartContractVersion');
        return {
            findOne: toQuery('smartContractVersion', 'FindByIdInput', smartContractVersionPartial, 'full'),
            findMany: toQuery('smartContractVersions', 'SmartContractVersionSearchInput', ...toSearch(smartContractVersionPartial)),
            create: toMutation('smartContractVersionCreate', 'SmartContractVersionCreateInput', smartContractVersionPartial, 'full'),
            update: toMutation('smartContractVersionUpdate', 'SmartContractVersionUpdateInput', smartContractVersionPartial, 'full'),
        }
    },
    standard: () => {
        const { standardPartial } = require('./partial/standard');
        return {
            findOne: toQuery('standard', 'FindByIdInput', standardPartial, 'full'),
            findMany: toQuery('standards', 'StandardSearchInput', ...toSearch(standardPartial)),
            create: toMutation('standardCreate', 'StandardCreateInput', standardPartial, 'full'),
            update: toMutation('standardUpdate', 'StandardUpdateInput', standardPartial, 'full'),
        }
    },
    standardVersion: () => {
        const { standardVersionPartial } = require('./partial/standardVersion');
        return {
            findOne: toQuery('standardVersion', 'FindByIdInput', standardVersionPartial, 'full'),
            findMany: toQuery('standardVersions', 'StandardVersionSearchInput', ...toSearch(standardVersionPartial)),
            create: toMutation('standardVersionCreate', 'StandardVersionCreateInput', standardVersionPartial, 'full'),
            update: toMutation('standardVersionUpdate', 'StandardVersionUpdateInput', standardVersionPartial, 'full'),
        }
    },
    star: () => {
        const { starPartial } = require('./partial/star');
        const { successPartial } = require('./partial/success');
        return {
            findMany: toQuery('stars', 'StarSearchInput', ...toSearch(starPartial)),
            star: toMutation('star', 'StarInput', successPartial, 'full'),
        }
    },
    statsApi: () => {
        const { statsApiPartial } = require('./partial/statsApi');
        return {
            findMany: toQuery('statsApi', 'StatsApiSearchInput', ...toSearch(statsApiPartial)),
        }
    },
    statsOrganization: () => {
        const { statsOrganizationPartial } = require('./partial/statsOrganization');
        return {
            findMany: toQuery('statsOrganization', 'StatsOrganizationSearchInput', ...toSearch(statsOrganizationPartial)),
        }
    },
    statsProject: () => {
        const { statsProjectPartial } = require('./partial/statsProject');
        return {
            findMany: toQuery('statsProject', 'StatsProjectSearchInput', ...toSearch(statsProjectPartial)),
        }
    },
    statsQuiz: () => {
        const { statsQuizPartial } = require('./partial/statsQuiz');
        return {
            findMany: toQuery('statsQuiz', 'StatsQuizSearchInput', ...toSearch(statsQuizPartial)),
        }
    },
    statsRoutine: () => {
        const { statsRoutinePartial } = require('./partial/statsRoutine');
        return {
            findMany: toQuery('statsRoutine', 'StatsRoutineSearchInput', ...toSearch(statsRoutinePartial)),
        }
    },
    statsSite: () => {
        const { statsSitePartial } = require('./partial/statsSite');
        return {
            findMany: toQuery('statsSite', 'StatsSiteSearchInput', ...toSearch(statsSitePartial)),
        }
    },
    statsSmartContract: () => {
        const { statsSmartContractPartial } = require('./partial/statsSmartContract');
        return {
            findMany: toQuery('statsSmartContract', 'StatsSmartContractSearchInput', ...toSearch(statsSmartContractPartial)),
        }
    },
    statsStandard: () => {
        const { statsStandardPartial } = require('./partial/statsStandard');
        return {
            findMany: toQuery('statsStandard', 'StatsStandardSearchInput', ...toSearch(statsStandardPartial)),
        }
    },
    statsUser: () => {
        const { statsUserPartial } = require('./partial/statsUser');
        return {
            findMany: toQuery('statsUser', 'StatsUserSearchInput', ...toSearch(statsUserPartial)),
        }
    },
    tag: () => {
        const { tagPartial } = require('./partial/tag');
        return {
            findOne: toQuery('tag', 'FindByIdInput', tagPartial, 'full'),
            findMany: toQuery('tags', 'TagSearchInput', ...toSearch(tagPartial)),
            create: toMutation('tagCreate', 'TagCreateInput', tagPartial, 'full'),
            update: toMutation('tagUpdate', 'TagUpdateInput', tagPartial, 'full'),
        }
    },
    transfer: () => {
        const { transferPartial } = require('./partial/transfer');
        return {
            findOne: toQuery('transfer', 'FindByIdInput', transferPartial, 'full'),
            findMany: toQuery('transfers', 'TransferSearchInput', ...toSearch(transferPartial)),
            requestSend: toMutation('transferRequestSend', 'TransferRequestSendInput', transferPartial, 'full'),
            requestReceive: toMutation('transferRequestReceive', 'TransferRequestReceiveInput', transferPartial, 'full'),
            transferUpdate: toMutation('transferUpdate', 'TransferUpdateInput', transferPartial, 'full'),
            transferCancel: toMutation('transferCancel', 'FindByIdInput', transferPartial, 'full'),
            transferAccept: toMutation('transferAccept', 'FindByIdInput', transferPartial, 'full'),
            transferDeny: toMutation('transferDeny', 'TransferDenyInput', transferPartial, 'full'),
        }
    },
    translate: () => {
        return {
            translate: toQuery('translate', 'FindByIdInput', {} as any, 'full'),//translatePartial, 'full'),
        }
    },
    user: () => {
        const { userPartial, profilePartial } = require('./partial/user');
        const { successPartial } = require('./partial/success');
        return {
            profile: toQuery('profile', null, profilePartial, 'full'),
            findOne: toQuery('user', 'FindByIdInput', userPartial, 'full'),
            findMany: toQuery('users', 'UserSearchInput', ...toSearch(userPartial)),
            profileUpdate: toMutation('profileUpdate', 'ProfileUpdateInput', profilePartial, 'full'),
            profileEmailUpdate: toMutation('profileEmailUpdate', 'ProfileEmailUpdateInput', profilePartial, 'full'),
            deleteOne: toMutation('userDeleteOne', 'UserDeleteInput', successPartial, 'full'),
            exportData: toMutation('exportData', null),
        }
    },
    userSchedule: () => {
        const { userSchedulePartial } = require('./partial/userSchedule');
        return {
            findOne: toQuery('userSchedule', 'FindByIdInput', userSchedulePartial, 'full'),
            findMany: toQuery('userSchedules', 'UserScheduleSearchInput', ...toSearch(userSchedulePartial)),
            create: toMutation('userScheduleCreate', 'UserScheduleCreateInput', userSchedulePartial, 'full'),
            update: toMutation('userScheduleUpdate', 'UserScheduleUpdateInput', userSchedulePartial, 'full'),
        }
    },
    view: () => {
        const { viewPartial } = require('./partial/view');
        return {
            findMany: toQuery('views', 'ViewSearchInput', ...toSearch(viewPartial)),
        }
    },
    vote: () => {
        const { votePartial } = require('./partial/vote');
        const { successPartial } = require('./partial/success');
        return {
            findMany: toQuery('votes', 'VoteSearchInput', ...toSearch(votePartial)),
            vote: toMutation('vote', 'VoteInput', successPartial, 'full'),
        }
    },
    wallet: () => {
        const { walletPartial } = require('./partial/wallet');
        return {
            findHandles: toQuery('findHandles', 'FindHandlesInput'),
            update: toMutation('walletUpdate', 'WalletUpdateInput', walletPartial, 'full'),
        }
    }
}