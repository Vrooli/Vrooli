import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, description, id, intPositiveOrOne, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { quizQuestionValidation } from "./quizQuestion.js";

export const quizTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const quizValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
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
    ], [["projectConnect", "routineConnect", true]], d),
    update: (d) => yupObj({
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
    ], [], d),
};
