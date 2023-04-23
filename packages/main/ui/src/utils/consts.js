import { InputType } from "@local/consts";
export const Forms = {
    ForgotPassword: "forgot-password",
    LogIn: "logIn",
    Profile: "profile",
    ResetPassword: "reset-password",
    SignUp: "signUp",
};
export var Status;
(function (Status) {
    Status["Incomplete"] = "Incomplete";
    Status["Invalid"] = "Invalid";
    Status["Valid"] = "Valid";
})(Status || (Status = {}));
export var BuildAction;
(function (BuildAction) {
    BuildAction["AddIncomingLink"] = "AddIncomingLink";
    BuildAction["AddOutgoingLink"] = "AddOutgoingLink";
    BuildAction["AddSubroutine"] = "AddSubroutine";
    BuildAction["EditSubroutine"] = "EditSubroutine";
    BuildAction["DeleteSubroutine"] = "DeleteSubroutine";
    BuildAction["OpenSubroutine"] = "OpenSubroutine";
    BuildAction["DeleteNode"] = "DeleteNode";
    BuildAction["UnlinkNode"] = "UnlinkNode";
    BuildAction["AddEndAfterNode"] = "AddEndAfterNode";
    BuildAction["AddListAfterNode"] = "AddListAfterNode";
    BuildAction["AddListBeforeNode"] = "AddListBeforeNode";
    BuildAction["MoveNode"] = "MoveNode";
})(BuildAction || (BuildAction = {}));
export var BuildRunState;
(function (BuildRunState) {
    BuildRunState[BuildRunState["Paused"] = 0] = "Paused";
    BuildRunState[BuildRunState["Running"] = 1] = "Running";
    BuildRunState[BuildRunState["Stopped"] = 2] = "Stopped";
})(BuildRunState || (BuildRunState = {}));
export var RoutineStepType;
(function (RoutineStepType) {
    RoutineStepType["RoutineList"] = "RoutineList";
    RoutineStepType["Decision"] = "Decision";
    RoutineStepType["Subroutine"] = "Subroutine";
})(RoutineStepType || (RoutineStepType = {}));
export var ResourceType;
(function (ResourceType) {
    ResourceType["Url"] = "Url";
    ResourceType["Wallet"] = "Wallet";
    ResourceType["Handle"] = "Handle";
})(ResourceType || (ResourceType = {}));
export const InputTypeOptions = [
    {
        label: "Text",
        value: InputType.TextField,
    },
    {
        label: "JSON",
        value: InputType.JSON,
    },
    {
        label: "Integer",
        value: InputType.IntegerInput,
    },
    {
        label: "Radio (Select One)",
        value: InputType.Radio,
    },
    {
        label: "Checkbox (Select any)",
        value: InputType.Checkbox,
    },
    {
        label: "Switch (On/Off)",
        value: InputType.Switch,
    },
    {
        label: "Markdown",
        value: InputType.Markdown,
    },
];
//# sourceMappingURL=consts.js.map