package main

import (
	"encoding/json"
	"net/http"

	feedbackhttp "landing-page-business-suite-api/handlers/feedback"
)

var feedbackHandlerDependencies = feedbackhttp.Dependencies{
	DecodeJSON: func(w http.ResponseWriter, r *http.Request, target interface{}) bool {
		if err := json.NewDecoder(r.Body).Decode(target); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request format.", ApiErrorTypeValidation)
			return false
		}
		return true
	},
	PathInt:        getPathParamInt,
	WriteErrorType: writeJSONError,
	WriteJSON: func(w http.ResponseWriter, value interface{}) error {
		return json.NewEncoder(w).Encode(value)
	},
	LogError: logStructuredError,
}

type feedbackEmailNotifier struct {
	configStore  *ConfigStore
	emailService *EmailService
}

func (n feedbackEmailNotifier) Notify(feedback *FeedbackRequest) {
	go func() {
		branding := n.configStore.GetBranding()
		dbBranding := &SiteBranding{SiteName: branding.SiteName, SupportEmail: branding.SupportEmail, DefaultTitle: branding.DefaultTitle, ThemePrimaryColor: branding.ThemePrimaryColor}
		if err := n.emailService.SendFeedbackNotification(dbBranding, feedback); err != nil {
			logStructuredError("feedback_email_send_failed", map[string]interface{}{"error": err.Error()})
		}
	}()
}

// handleFeedbackCreateWithConfigStore composes feedback creation with the
// branding/email notification integration.
func handleFeedbackCreateWithConfigStore(svc *FeedbackService, cs *ConfigStore, emailSvc *EmailService) http.HandlerFunc {
	return feedbackhttp.Create(feedbackHandlerDependencies, svc, feedbackEmailNotifier{configStore: cs, emailService: emailSvc})
}

func handleFeedbackList(svc FeedbackServicer) http.HandlerFunc {
	return feedbackhttp.List(feedbackHandlerDependencies, svc)
}

func handleFeedbackGet(svc FeedbackServicer) http.HandlerFunc {
	return feedbackhttp.Get(feedbackHandlerDependencies, svc)
}

func handleFeedbackUpdateStatus(svc FeedbackServicer) http.HandlerFunc {
	return feedbackhttp.UpdateStatus(feedbackHandlerDependencies, svc)
}

func handleFeedbackDelete(svc FeedbackServicer) http.HandlerFunc {
	return feedbackhttp.Delete(feedbackHandlerDependencies, svc)
}

func handleFeedbackDeleteBulk(svc FeedbackServicer) http.HandlerFunc {
	return feedbackhttp.DeleteBulk(feedbackHandlerDependencies, svc)
}
