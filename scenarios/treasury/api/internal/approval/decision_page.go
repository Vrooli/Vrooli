package approval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"

	"treasury/internal/budget"
)

// Limits is how a held charge sits against the budget that checks it, read
// when the decision is raised.
type Limits struct {
	// HeldBecause is the authorization's remediation in words, when it has one.
	HeldBecause string
	Headroom    budget.Headroom
}

// LimitsSource reads a held charge's limits.
type LimitsSource func(context.Context, Request) (Limits, error)

// money renders minor units in the charge's currency: 110 USD minor units is
// "$1.10", not "110 USD".
func money(minor int64, currency string) string {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	amount := fmt.Sprintf("%d.%02d", minor/100, minor%100)
	switch strings.ToUpper(currency) {
	case "USD":
		return sign + "$" + amount
	case "EUR":
		return sign + "€" + amount
	case "GBP":
		return sign + "£" + amount
	}
	return sign + amount + " " + strings.ToUpper(currency)
}

// payee names a counterparty briefly; a long wallet address keeps its ends.
func payee(counterparty string) string {
	if strings.HasPrefix(counterparty, "0x") && len(counterparty) > 14 {
		return "wallet " + counterparty[:6] + "…" + counterparty[len(counterparty)-4:]
	}
	return counterparty
}

func approvalDecisionFacts(request Request, limits *Limits) operatordecision.Facts {
	amount := money(request.AmountMinor, request.Currency)
	question := fmt.Sprintf("Approve one held payment of %s to %s?", amount, payee(request.Counterparty))
	if limits != nil && limits.Headroom.PerTransactionCapMinor > 0 && request.AmountMinor > limits.Headroom.PerTransactionCapMinor {
		question += fmt.Sprintf(" It is over the automatic per-payment limit of %s.", money(limits.Headroom.PerTransactionCapMinor, request.Currency))
	}
	deadline := request.ExpiresAt.UTC()
	facts := operatordecision.Facts{
		Subject:  fmt.Sprintf("Approve a %s payment?", amount),
		Question: question,
		Options: []operatordecision.Option{
			{Key: "approve", Label: fmt.Sprintf("Approve: pay %s once, if the budget still allows it when you answer", amount)},
			{Key: "decline", Label: "Decline: cancel this payment and release its hold"},
		},
		Reversible: false,
		Urgency:    "high",
		// One open decision per held charge, so a re-send with fresher limits
		// revises it in place instead of opening a second one.
		SubjectKey: "treasury-approval:" + operatordecision.CorrelationID("", request.ID),
		BigPicture: fmt.Sprintf("An agent wants Treasury to pay %s to %s. Treasury held the payment because it needs a person's approval.\n\n"+
			"If you approve, Treasury pays this one charge after checking again that the budget, funding and freeze switches still allow it. Approving never raises a limit or allows a later payment. If you decline, or nobody answers by %s, nothing is paid and the hold is released.",
			amount, payee(request.Counterparty), deadline.Format("15:04 MST on 2 Jan")),
		TechnicalDetails: fmt.Sprintf("Authorization: %s. Approval: %s. Amount: %d minor units %s. Counterparty: %s. Book: %s. Mandate: %s. Requesting agent: %s. Expires: %s. This answer cannot change spending policy or authorize another transaction.",
			request.AuthorizationID, request.ID, request.AmountMinor, request.Currency, request.Counterparty, request.BookID, request.MandateID, request.RequestingAgent, deadline.Format(time.RFC3339)),
		Deadline: deadline,
		Correlation: map[string]string{
			"approval_id":      request.ID,
			"authorization_id": request.AuthorizationID,
			"book_id":          request.BookID,
			"mandate_id":       request.MandateID,
			"requesting_agent": request.RequestingAgent,
			"amount_minor":     fmt.Sprint(request.AmountMinor),
			"currency":         request.Currency,
			"counterparty":     request.Counterparty,
		},
	}
	page := operatordecision.NewPage(fmt.Sprintf("Pay %s to %s?", amount, payee(request.Counterparty))).
		Lead("An agent asked Treasury to make this payment. Treasury held it for you instead of paying automatically.").
		Facts(
			operatordecision.Row{Label: "Amount", Value: amount},
			operatordecision.Row{Label: "To", Value: request.Counterparty},
			operatordecision.Row{Label: "Asked by", Value: request.RequestingAgent},
			operatordecision.Row{Label: "Answer by", Value: deadline.Format("15:04 MST, 2 Jan 2006")},
		)
	if limits != nil {
		h := limits.Headroom
		cur := request.Currency
		if limits.HeldBecause != "" {
			page.Callout("warning", "Why it was held:", limits.HeldBecause+".")
		}
		rows := [][]string{}
		if h.PerTransactionCapMinor > 0 {
			state := "Within"
			if request.AmountMinor > h.PerTransactionCapMinor {
				state = "Over by " + money(request.AmountMinor-h.PerTransactionCapMinor, cur)
			}
			rows = append(rows, []string{"Automatic limit per payment", money(h.PerTransactionCapMinor, cur), state})
		}
		if h.PeriodicCapMinor > 0 {
			rows = append(rows, []string{"This period's budget", fmt.Sprintf("%s used of %s", money(h.PeriodUsedMinor, cur), money(h.PeriodicCapMinor, cur)), "Left after paying: " + money(h.PeriodRemainingMinor-request.AmountMinor, cur)})
		}
		if h.TotalCapMinor > 0 {
			rows = append(rows, []string{"Total budget", fmt.Sprintf("%s used of %s", money(h.TotalUsedMinor, cur), money(h.TotalCapMinor, cur)), "Left after paying: " + money(h.TotalRemainingMinor-request.AmountMinor, cur)})
		}
		if h.FundingRequired {
			rows = append(rows, []string{"Funding received", money(h.FundingCreditedMinor, cur), "Left after paying: " + money(h.FundingRemainingMinor-request.AmountMinor, cur)})
		}
		page.Section("Against the budget", "").Table([]string{"Limit", "Now", "With this payment"}, rows)
	}
	facts.Explainer = page.Callout("", "Only this payment:", "approving does not change any limit, mandate or later payment.").
		Options(facts, map[string]string{
			"approve": "Treasury checks the budget, funding and freeze switches once more, then pays. If any check fails, nothing is paid and you are told.",
			"decline": "Nothing is paid. The hold is released and the agent is told no.",
		}).
		IfNoAnswer(fmt.Sprintf("At %s the hold lapses: nothing is paid.", deadline.Format("15:04 MST"))).
		HTML()
	return facts
}
