import { bool, description, id, intPositiveOrOne, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { quizQuestionValidation } from "./quizQuestion";

export const quizTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
});

export const quizValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        maxAttempts: opt(intPositiveOrOne),
        randomizeQuestionOrder: opt(bool),
        revealCorrectAnswers: opt(bool),
        timeLimit: opt(intPositiveOrOne),
        pointsToPass: opt(intPositiveOrOne),
    }, [
        ["routine", ["Connect"], "one", "opt"],
        ["project", ["Connect"], "one", "opt"],
        ["quizQuestions", ["Create"], "many", "opt", quizQuestionValidation],
        ["translations", ["Create"], "many", "opt", quizTranslationValidation],
    ], [["projectConnect", "routineConnect"]], o),
    update: ({ o }) => yupObj({
        id: req(id),
        maxAttempts: opt(intPositiveOrOne),
        randomizeQuestionOrder: opt(bool),
        revealCorrectAnswers: opt(bool),
        timeLimit: opt(intPositiveOrOne),
        pointsToPass: opt(intPositiveOrOne),
    }, [
        ["routine", ["Connect", "Disconnect"], "one", "opt"],
        ["project", ["Connect", "Disconnect"], "one", "opt"],
        ["quizQuestions", ["Create", "Update", "Delete"], "many", "opt", quizQuestionValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", quizTranslationValidation],
    ], [], o),
};
