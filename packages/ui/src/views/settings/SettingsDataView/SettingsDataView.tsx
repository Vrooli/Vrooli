import { LINKS } from "@local/shared";
import { Box, Button, Checkbox, FormControlLabel, FormHelperText, Grid, Stack } from "@mui/material";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { SettingsDataFormProps, SettingsDataViewProps } from "../types";

const DataOption = ({
    disabled,
    help,
    label,
    name,
}: {
    disabled: boolean,
    help?: string,
    label: string,
    name: string,
}) => {
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
};

const SettingsDataForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsDataFormProps) => {
    const { t } = useTranslation();

    const [allField] = useField<boolean>("all");

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                style={{ margin: "auto" }}
            >
                <Box display="flex" flexDirection="column" sx={{ gap: 1, maxWidth: "600px", margin: "auto" }}>
                    <DataOption
                        disabled={false}
                        label="All"
                        name="all"
                    />
                    <Title title={"Personal"} variant="subheader" sxs={{ stack: { paddingLeft: 0, marginRight: "auto" } }} />
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
                    <Title title={"Engagement & Contributions"} variant="subheader" sxs={{ stack: { paddingLeft: 0, marginRight: "auto" } }} />
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
                    <Title title={"Collaborative"} variant="subheader" sxs={{ stack: { paddingLeft: 0, marginRight: "auto" } }} />
                    <DataOption
                        disabled={allField.value}
                        help="Will not delete teams with other admins"
                        label="Teams"
                        name="organizations"
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
                    <Title title={"Automations"} variant="subheader" sxs={{ stack: { paddingLeft: 0, marginRight: "auto" } }} />
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
                        label="Smart Contracts"
                        name="smartContracts"
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
};

export const SettingsDataView = ({
    isOpen,
    onClose,
}: SettingsDataViewProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const [, setLocation] = useLocation();

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Data")}
            />
            <Stack direction="row" mt={2} sx={{ paddingBottom: pagePaddingBottom }}>
                <SettingsList />
                <Box m="auto" p={1}>
                    <Box
                        display="flex"
                        flexDirection="column"
                        justifyContent="center"
                        alignItems="center"
                        margin="auto"
                        sx={{
                            gap: 2,
                            width: "min(100%, 600px)",
                            marginBottom: 4,
                        }}>
                        <MarkdownDisplay content={t("DataSettingsPageText")} />
                        <Button variant="outlined" color="primary" onClick={() => { setLocation(LINKS.SettingsPrivacy); }}>{t("DataSettingsPageText2")}</Button>
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
                            comments: false,
                            issues: false,
                            notes: false,
                            organizations: false, // Includes meetings and roles
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
                            smartContracts: false,
                            standards: false,
                            // tags: false,
                            views: false,
                        }}
                        onSubmit={(values, helpers) => {
                            console.log("onsubmit", values);
                            //TODO
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
            </Stack>
        </>
    );
};
