import { Box, Button, Checkbox, Divider, FormControlLabel, FormHelperText, Grid } from "@mui/material";
import { Field, Formik, useField } from "formik";
import { useTranslation } from "react-i18next";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsContent, SettingsTopBar } from "../../components/navigation/SettingsTopBar.js";
import { MarkdownDisplay } from "../../components/text/MarkdownDisplay.js";
import { Title } from "../../components/text/Title.js";
import { BaseForm } from "../../forms/BaseForm/BaseForm.js";
import { HeartFilledIcon, RoutineIcon, TeamIcon, UserIcon } from "../../icons/common.js";
import { useLocation } from "../../route/router.js";
import { SettingsDataFormProps, SettingsDataViewProps } from "./types.js";

//TODO add section for managing uploaded files, with information about how much storage is used and a way to delete files
function DataOption({
    disabled,
    help,
    label,
    name,
}: {
    disabled: boolean,
    help?: string,
    label: string,
    name: string,
}) {
    return (
        <Box>
            <FormControlLabel
                control={<Field
                    name={name}
                    type="checkbox"
                    as={Checkbox}
                    size="large"
                    color="secondary"
                />}
                disabled={disabled}
                label={label}
            />
            <FormHelperText>
                {help}
            </FormHelperText>
        </Box>
    );
}

function SettingsDataForm({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsDataFormProps) {
    const { t } = useTranslation();

    const [allField] = useField<boolean>("all");

    return (
        <>
            <BaseForm
                display={display}
                isLoading={isLoading}
                style={{ margin: "auto" }}
            >
                <Box display="flex" flexDirection="column" sx={{ gap: 1, maxWidth: "600px", margin: "auto" }}>
                    <DataOption
                        disabled={false}
                        label="All"
                        name="all"
                        help="All data associated with your account"
                    />
                    <Divider sx={{ marginTop: 2 }} />
                    <Box display="flex" flexDirection="row" sx={{ gap: 1, marginRight: "auto" }}>
                        <Title
                            Icon={UserIcon}
                            title={"Personal"}
                            variant="subsection"
                            sxs={{ stack: { paddingLeft: 0 } }}
                        />
                        <Checkbox
                            size="large"
                            color="secondary"
                            onChange={(e) => {
                                const { checked } = e.target;
                                props.setFieldValue("account", checked);
                                props.setFieldValue("projects", checked);
                                props.setFieldValue("reminders", checked);
                                props.setFieldValue("notes", checked);
                                props.setFieldValue("runs", checked);
                                props.setFieldValue("schedules", checked);
                                props.setFieldValue("bookmarks", checked);
                                props.setFieldValue("views", checked);
                            }}
                        />
                    </Box>
                    <DataOption
                        disabled={allField.value}
                        help="Profile, emails, wallets, awards, focus modes, payment history, api usage, and user stats"
                        label="Account"
                        name="account"
                    />
                    <DataOption
                        disabled={allField.value}
                        label="Projects"
                        name="projects"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="This is always private, and deleted when your account is deleted"
                        label="Reminders"
                        name="reminders"
                    />
                    <DataOption
                        disabled={allField.value}
                        label="Notes"
                        name="notes"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="History, inputs, and other data related to runs of routines and projects"
                        label="Runs"
                        name="runs"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="This is always private, and deleted when your account is deleted"
                        label="Schedules"
                        name="schedules"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="These are always deleted when your account is deleted"
                        label="Bookmarks"
                        name="bookmarks"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="This is always private, and deleted when your account is deleted"
                        label="Views"
                        name="views"
                    />
                    <Divider sx={{ marginTop: 2 }} />
                    <Box display="flex" flexDirection="row" sx={{ gap: 1, marginRight: "auto" }}>
                        <Title
                            Icon={HeartFilledIcon}
                            title={"Engagement & Contributions"}
                            variant="subsection"
                            sxs={{ stack: { paddingLeft: 0 } }}
                        />
                        <Checkbox
                            size="large"
                            color="secondary"
                            onChange={(e) => {
                                const { checked } = e.target;
                                props.setFieldValue("comments", checked);
                                props.setFieldValue("issues", checked);
                                props.setFieldValue("pullRequests", checked);
                                props.setFieldValue("questions", checked);
                                props.setFieldValue("questionAnswers", checked);
                                props.setFieldValue("reactions", checked);
                                props.setFieldValue("reports", checked);
                            }}
                        />
                    </Box>
                    <DataOption
                        disabled={allField.value}
                        help="Comments you've created"
                        label="Comments"
                        name="comments"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="Issues you've created"
                        label="Issues"
                        name="issues"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="Pull requests you've opened"
                        label="Pull Requests"
                        name="pullRequests"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="Also deletes answers on your questions"
                        label="Questions"
                        name="questions"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="Every answer you've submitted to a question"
                        label="Question Answers"
                        name="questionAnswers"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="Votes and emoji reactions"
                        label="Reactions"
                        name="reactions"
                    />
                    <DataOption
                        disabled={allField.value}
                        label="Reports"
                        name="reports"
                    />
                    <Divider sx={{ marginTop: 2 }} />
                    <Box display="flex" flexDirection="row" sx={{ gap: 1, marginRight: "auto" }}>
                        <Title
                            Icon={TeamIcon}
                            title={"Collaborative"}
                            variant="subsection"
                            sxs={{ stack: { paddingLeft: 0 } }}
                        />
                        <Checkbox
                            size="large"
                            color="secondary"
                            onChange={(e) => {
                                const { checked } = e.target;
                                props.setFieldValue("teams", checked);
                                props.setFieldValue("chats", checked);
                                props.setFieldValue("bots", checked);
                            }}
                        />
                    </Box>
                    <DataOption
                        disabled={allField.value}
                        help="Will not delete teams with other admins"
                        label="Teams"
                        name="teams"
                    />
                    <DataOption
                        disabled={allField.value}
                        help="Will not delete group chats with other admins"
                        label="Chats"
                        name="chats"
                    />
                    <DataOption
                        disabled={allField.value}
                        label="Bots"
                        name="bots"
                    />
                    <Divider sx={{ marginTop: 2 }} />
                    <Box display="flex" flexDirection="row" sx={{ gap: 1, marginRight: "auto" }}>
                        <Title
                            Icon={RoutineIcon}
                            title={"Automations"}
                            variant="subsection"
                            sxs={{ stack: { paddingLeft: 0 } }}
                        />
                        <Checkbox
                            size="large"
                            color="secondary"
                            onChange={(e) => {
                                const { checked } = e.target;
                                props.setFieldValue("routines", checked);
                                props.setFieldValue("apis", checked);
                                props.setFieldValue("standards", checked);
                                props.setFieldValue("codes", checked);
                            }}
                        />
                    </Box>
                    <DataOption
                        disabled={allField.value}
                        label="Routines"
                        name="routines"
                    />
                    <DataOption
                        disabled={allField.value}
                        label="Apis"
                        name="apis"
                    />
                    <DataOption
                        disabled={allField.value}
                        label="Standards"
                        name="standards"
                    />
                    <DataOption
                        disabled={allField.value}
                        label="Codes"
                        name="codes"
                    />
                </Box>
            </BaseForm>
            <Grid container spacing={2} sx={{ margin: "auto", maxWidth: { xs: "400px", md: "630px" } }}>
                <Grid item xs={12} md={4}>
                    <Button
                        disabled={isLoading}
                        fullWidth
                        variant="contained"
                        color="secondary"
                        onClick={() => {
                            props.setFieldValue("requestType", "Download");
                            props.submitForm();
                        }}
                    >{t("Download")}</Button>
                </Grid>
                <Grid item xs={12} md={4}>
                    <Button
                        disabled={isLoading}
                        fullWidth
                        variant="outlined"
                        onClick={() => {
                            props.setFieldValue("requestType", "DownloadAndDelete");
                            props.submitForm();
                        }}
                        sx={{
                            borderColor: "error.main",
                            color: "error.main",
                        }}
                    >{t("DownloadAndDelete")}</Button>
                </Grid>
                <Grid item xs={12} md={4}>
                    <Button
                        disabled={isLoading}
                        fullWidth
                        variant="outlined"
                        onClick={() => {
                            props.setFieldValue("requestType", "Delete");
                            props.submitForm();
                        }}
                        sx={{
                            borderColor: "error.main",
                            color: "error.main",
                        }}
                    >{t("Delete")}</Button>
                </Grid>
            </Grid>
        </>
    );
}

export function SettingsDataView({
    display,
    onClose,
}: SettingsDataViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Data")}
            />
            <SettingsContent>
                <SettingsList />
                <Box m="auto" p={1}>
                    <Box
                        display="flex"
                        flexDirection="column"
                        justifyContent="center"
                        alignItems="center"
                        margin="auto"
                        sx={{
                            width: "min(100%, 600px)",
                            marginBottom: 4,
                        }}>
                        <MarkdownDisplay content={t("DataSettingsPageText")} />
                    </Box>
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            requestType: "Download",
                            all: false,
                            account: false, // Profile, emails, wallets, awards, focus modes, payment history, api usage, user stats
                            apis: false,
                            bookmarks: false,
                            bots: false,
                            chats: false,
                            codes: false,
                            comments: false,
                            issues: false,
                            notes: false,
                            //posts: false,
                            pullRequests: false,
                            projects: false,
                            questions: false,
                            questionAnswers: false,
                            // quizzes: false,
                            // quizResponses: false, // Includes attempts, answers, etc.
                            reactions: false,
                            reminders: false,
                            reports: false,
                            routines: false,
                            runs: false,
                            schedules: false,
                            standards: false,
                            // tags: false,
                            teams: false, // Includes meetings and roles
                            views: false,
                        }}
                        onSubmit={(values, helpers) => {
                            console.log("onsubmit", values);
                            //TODO make sure they have at least one validated email address to send a download link to, if not just deleting
                            // TODO make request
                        }}
                    >
                        {(formik) => <SettingsDataForm
                            display={display}
                            isLoading={false} //TODO
                            onCancel={formik.resetForm}
                            {...formik}
                        />}
                    </Formik>
                </Box>
            </SettingsContent>
        </>
    );
}
