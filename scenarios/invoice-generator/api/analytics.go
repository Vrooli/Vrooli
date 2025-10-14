package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// DashboardSummary represents the main dashboard analytics
type DashboardSummary struct {
	TotalInvoices       int     `json:"total_invoices"`
	TotalRevenue        float64 `json:"total_revenue"`
	TotalPaid           float64 `json:"total_paid"`
	TotalPending        float64 `json:"total_pending"`
	TotalOverdue        float64 `json:"total_overdue"`
	AverageInvoiceValue float64 `json:"average_invoice_value"`
	AveragePaymentDays  float64 `json:"average_payment_days"`
	ActiveClients       int     `json:"active_clients"`
	OverdueInvoices     int     `json:"overdue_invoices"`
	MonthlyRevenue      float64 `json:"monthly_revenue"`
	MonthlyInvoices     int     `json:"monthly_invoices"`
}

// RevenueAnalytics represents revenue trends over time
type RevenueAnalytics struct {
	MonthlyRevenue []MonthlyRevenue `json:"monthly_revenue"`
	QuarterlyRevenue []QuarterlyRevenue `json:"quarterly_revenue"`
	YearlyRevenue []YearlyRevenue `json:"yearly_revenue"`
	RevenueGrowth float64 `json:"revenue_growth_percent"`
}

// MonthlyRevenue represents revenue for a specific month
type MonthlyRevenue struct {
	Month       string  `json:"month"`        // Format: 2025-01
	Revenue     float64 `json:"revenue"`
	Invoices    int     `json:"invoices"`
	AvgValue    float64 `json:"avg_value"`
	PaidCount   int     `json:"paid_count"`
	PendingCount int    `json:"pending_count"`
}

// QuarterlyRevenue represents revenue for a quarter
type QuarterlyRevenue struct {
	Quarter     string  `json:"quarter"`      // Format: 2025-Q1
	Revenue     float64 `json:"revenue"`
	Invoices    int     `json:"invoices"`
}

// YearlyRevenue represents revenue for a year
type YearlyRevenue struct {
	Year        string  `json:"year"`         // Format: 2025
	Revenue     float64 `json:"revenue"`
	Invoices    int     `json:"invoices"`
}

// ClientAnalytics represents client-related analytics
type ClientAnalytics struct {
	TopClients         []TopClient      `json:"top_clients"`
	ClientsWithBalance []ClientBalance  `json:"clients_with_balance"`
	TotalClients       int              `json:"total_clients"`
	ActiveClients      int              `json:"active_clients"`
}

// TopClient represents a client ranked by revenue
type TopClient struct {
	ClientID       string  `json:"client_id"`
	ClientName     string  `json:"client_name"`
	TotalRevenue   float64 `json:"total_revenue"`
	InvoiceCount   int     `json:"invoice_count"`
	AverageValue   float64 `json:"average_value"`
	LastInvoiceDate string `json:"last_invoice_date"`
}

// ClientBalance represents a client's outstanding balance
type ClientBalance struct {
	ClientID       string  `json:"client_id"`
	ClientName     string  `json:"client_name"`
	TotalBalance   float64 `json:"total_balance"`
	OverdueBalance float64 `json:"overdue_balance"`
	InvoiceCount   int     `json:"invoice_count"`
}

// InvoiceAnalytics represents invoice-related analytics
type InvoiceAnalytics struct {
	AgingReport        []InvoiceAging   `json:"aging_report"`
	StatusDistribution []StatusCount    `json:"status_distribution"`
	AveragePaymentDays float64          `json:"average_payment_days"`
	CollectionRate     float64          `json:"collection_rate"`
}

// InvoiceAging represents invoices grouped by age
type InvoiceAging struct {
	AgeBracket    string  `json:"age_bracket"`    // "0-30", "31-60", "61-90", "90+"
	InvoiceCount  int     `json:"invoice_count"`
	TotalAmount   float64 `json:"total_amount"`
}

// StatusCount represents count of invoices by status
type StatusCount struct {
	Status       string  `json:"status"`
	Count        int     `json:"count"`
	TotalAmount  float64 `json:"total_amount"`
}

// GetDashboardSummaryHandler returns comprehensive dashboard analytics
func getDashboardSummaryHandler(w http.ResponseWriter, r *http.Request) {
	summary := DashboardSummary{}

	// Total invoices
	db.QueryRow(`SELECT COUNT(*) FROM invoices`).Scan(&summary.TotalInvoices)

	// Total revenue (all invoices)
	var totalRevenue sql.NullFloat64
	db.QueryRow(`SELECT SUM(total_amount) FROM invoices`).Scan(&totalRevenue)
	if totalRevenue.Valid {
		summary.TotalRevenue = totalRevenue.Float64
	}

	// Total paid
	var totalPaid sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(paid_amount) FROM invoices WHERE status IN ('paid', 'partially_paid')
	`).Scan(&totalPaid)
	if totalPaid.Valid {
		summary.TotalPaid = totalPaid.Float64
	}

	// Total pending (not overdue)
	var totalPending sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(balance_due) FROM invoices
		WHERE status IN ('sent', 'partially_paid') AND due_date >= CURRENT_DATE
	`).Scan(&totalPending)
	if totalPending.Valid {
		summary.TotalPending = totalPending.Float64
	}

	// Total overdue
	var totalOverdue sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(balance_due) FROM invoices
		WHERE status IN ('sent', 'partially_paid', 'overdue') AND due_date < CURRENT_DATE
	`).Scan(&totalOverdue)
	if totalOverdue.Valid {
		summary.TotalOverdue = totalOverdue.Float64
	}

	// Average invoice value
	var avgValue sql.NullFloat64
	db.QueryRow(`SELECT AVG(total_amount) FROM invoices`).Scan(&avgValue)
	if avgValue.Valid {
		summary.AverageInvoiceValue = avgValue.Float64
	}

	// Average payment days (for paid invoices)
	var avgDays sql.NullFloat64
	db.QueryRow(`
		SELECT AVG(
			EXTRACT(EPOCH FROM (
				SELECT MIN(p.payment_date::date - i.issue_date::date)
				FROM payments p
				WHERE p.invoice_id = i.id
			))
		)
		FROM invoices i
		WHERE i.status = 'paid'
	`).Scan(&avgDays)
	if avgDays.Valid {
		summary.AveragePaymentDays = avgDays.Float64
	}

	// Active clients
	db.QueryRow(`SELECT COUNT(*) FROM clients WHERE is_active = true`).Scan(&summary.ActiveClients)

	// Overdue invoices count
	db.QueryRow(`
		SELECT COUNT(*) FROM invoices
		WHERE status = 'overdue' AND balance_due > 0
	`).Scan(&summary.OverdueInvoices)

	// Monthly revenue (current month)
	var monthlyRevenue sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(total_amount) FROM invoices
		WHERE EXTRACT(YEAR FROM issue_date::date) = EXTRACT(YEAR FROM CURRENT_DATE)
		AND EXTRACT(MONTH FROM issue_date::date) = EXTRACT(MONTH FROM CURRENT_DATE)
	`).Scan(&monthlyRevenue)
	if monthlyRevenue.Valid {
		summary.MonthlyRevenue = monthlyRevenue.Float64
	}

	// Monthly invoices count (current month)
	db.QueryRow(`
		SELECT COUNT(*) FROM invoices
		WHERE EXTRACT(YEAR FROM issue_date::date) = EXTRACT(YEAR FROM CURRENT_DATE)
		AND EXTRACT(MONTH FROM issue_date::date) = EXTRACT(MONTH FROM CURRENT_DATE)
	`).Scan(&summary.MonthlyInvoices)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetRevenueAnalyticsHandler returns revenue trends and growth analytics
func getRevenueAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	analytics := RevenueAnalytics{}

	// Monthly revenue for last 12 months
	rows, err := db.Query(`
		SELECT
			TO_CHAR(issue_date::date, 'YYYY-MM') as month,
			SUM(total_amount) as revenue,
			COUNT(*) as invoices,
			AVG(total_amount) as avg_value,
			SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) as paid_count,
			SUM(CASE WHEN status IN ('sent', 'partially_paid', 'draft') THEN 1 ELSE 0 END) as pending_count
		FROM invoices
		WHERE issue_date >= CURRENT_DATE - INTERVAL '12 months'
		GROUP BY TO_CHAR(issue_date::date, 'YYYY-MM')
		ORDER BY month DESC
	`)

	if err != nil {
		log.Printf("Error fetching monthly revenue: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var mr MonthlyRevenue
			var revenue, avgValue sql.NullFloat64
			err := rows.Scan(&mr.Month, &revenue, &mr.Invoices, &avgValue, &mr.PaidCount, &mr.PendingCount)
			if err == nil {
				if revenue.Valid {
					mr.Revenue = revenue.Float64
				}
				if avgValue.Valid {
					mr.AvgValue = avgValue.Float64
				}
				analytics.MonthlyRevenue = append(analytics.MonthlyRevenue, mr)
			}
		}
	}

	// Quarterly revenue for last 4 quarters
	qRows, err := db.Query(`
		SELECT
			TO_CHAR(issue_date::date, 'YYYY') || '-Q' || TO_CHAR(issue_date::date, 'Q') as quarter,
			SUM(total_amount) as revenue,
			COUNT(*) as invoices
		FROM invoices
		WHERE issue_date >= CURRENT_DATE - INTERVAL '18 months'
		GROUP BY TO_CHAR(issue_date::date, 'YYYY'), TO_CHAR(issue_date::date, 'Q')
		ORDER BY quarter DESC
		LIMIT 4
	`)

	if err != nil {
		log.Printf("Error fetching quarterly revenue: %v", err)
	} else {
		defer qRows.Close()
		for qRows.Next() {
			var qr QuarterlyRevenue
			var revenue sql.NullFloat64
			err := qRows.Scan(&qr.Quarter, &revenue, &qr.Invoices)
			if err == nil {
				if revenue.Valid {
					qr.Revenue = revenue.Float64
				}
				analytics.QuarterlyRevenue = append(analytics.QuarterlyRevenue, qr)
			}
		}
	}

	// Yearly revenue
	yRows, err := db.Query(`
		SELECT
			TO_CHAR(issue_date::date, 'YYYY') as year,
			SUM(total_amount) as revenue,
			COUNT(*) as invoices
		FROM invoices
		WHERE issue_date >= CURRENT_DATE - INTERVAL '5 years'
		GROUP BY TO_CHAR(issue_date::date, 'YYYY')
		ORDER BY year DESC
	`)

	if err != nil {
		log.Printf("Error fetching yearly revenue: %v", err)
	} else {
		defer yRows.Close()
		for yRows.Next() {
			var yr YearlyRevenue
			var revenue sql.NullFloat64
			err := yRows.Scan(&yr.Year, &revenue, &yr.Invoices)
			if err == nil {
				if revenue.Valid {
					yr.Revenue = revenue.Float64
				}
				analytics.YearlyRevenue = append(analytics.YearlyRevenue, yr)
			}
		}
	}

	// Calculate revenue growth (current month vs previous month)
	var currentMonthRevenue, previousMonthRevenue sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(total_amount) FROM invoices
		WHERE EXTRACT(YEAR FROM issue_date::date) = EXTRACT(YEAR FROM CURRENT_DATE)
		AND EXTRACT(MONTH FROM issue_date::date) = EXTRACT(MONTH FROM CURRENT_DATE)
	`).Scan(&currentMonthRevenue)

	db.QueryRow(`
		SELECT SUM(total_amount) FROM invoices
		WHERE issue_date::date >= DATE_TRUNC('month', CURRENT_DATE - INTERVAL '1 month')
		AND issue_date::date < DATE_TRUNC('month', CURRENT_DATE)
	`).Scan(&previousMonthRevenue)

	if currentMonthRevenue.Valid && previousMonthRevenue.Valid && previousMonthRevenue.Float64 > 0 {
		analytics.RevenueGrowth = ((currentMonthRevenue.Float64 - previousMonthRevenue.Float64) / previousMonthRevenue.Float64) * 100
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// GetClientAnalyticsHandler returns client-related analytics
func getClientAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	analytics := ClientAnalytics{}

	// Total and active clients
	db.QueryRow(`SELECT COUNT(*) FROM clients`).Scan(&analytics.TotalClients)
	db.QueryRow(`SELECT COUNT(*) FROM clients WHERE is_active = true`).Scan(&analytics.ActiveClients)

	// Top clients by revenue
	topRows, err := db.Query(`
		SELECT
			c.id,
			c.name,
			SUM(i.total_amount) as total_revenue,
			COUNT(i.id) as invoice_count,
			AVG(i.total_amount) as avg_value,
			MAX(i.issue_date) as last_invoice_date
		FROM clients c
		JOIN invoices i ON c.id = i.client_id
		GROUP BY c.id, c.name
		ORDER BY total_revenue DESC
		LIMIT 10
	`)

	if err != nil {
		log.Printf("Error fetching top clients: %v", err)
	} else {
		defer topRows.Close()
		for topRows.Next() {
			var tc TopClient
			var revenue, avgValue sql.NullFloat64
			err := topRows.Scan(&tc.ClientID, &tc.ClientName, &revenue, &tc.InvoiceCount, &avgValue, &tc.LastInvoiceDate)
			if err == nil {
				if revenue.Valid {
					tc.TotalRevenue = revenue.Float64
				}
				if avgValue.Valid {
					tc.AverageValue = avgValue.Float64
				}
				analytics.TopClients = append(analytics.TopClients, tc)
			}
		}
	}

	// Clients with outstanding balances
	balanceRows, err := db.Query(`
		SELECT
			c.id,
			c.name,
			SUM(i.balance_due) as total_balance,
			SUM(CASE WHEN i.status = 'overdue' THEN i.balance_due ELSE 0 END) as overdue_balance,
			COUNT(i.id) as invoice_count
		FROM clients c
		JOIN invoices i ON c.id = i.client_id
		WHERE i.balance_due > 0
		GROUP BY c.id, c.name
		ORDER BY total_balance DESC
		LIMIT 20
	`)

	if err != nil {
		log.Printf("Error fetching client balances: %v", err)
	} else {
		defer balanceRows.Close()
		for balanceRows.Next() {
			var cb ClientBalance
			var totalBalance, overdueBalance sql.NullFloat64
			err := balanceRows.Scan(&cb.ClientID, &cb.ClientName, &totalBalance, &overdueBalance, &cb.InvoiceCount)
			if err == nil {
				if totalBalance.Valid {
					cb.TotalBalance = totalBalance.Float64
				}
				if overdueBalance.Valid {
					cb.OverdueBalance = overdueBalance.Float64
				}
				analytics.ClientsWithBalance = append(analytics.ClientsWithBalance, cb)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// GetInvoiceAnalyticsHandler returns invoice aging and status analytics
func getInvoiceAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	analytics := InvoiceAnalytics{}

	// Aging report
	agingRows, err := db.Query(`
		SELECT
			CASE
				WHEN CURRENT_DATE - due_date::date <= 30 THEN '0-30'
				WHEN CURRENT_DATE - due_date::date <= 60 THEN '31-60'
				WHEN CURRENT_DATE - due_date::date <= 90 THEN '61-90'
				ELSE '90+'
			END as age_bracket,
			COUNT(*) as invoice_count,
			SUM(balance_due) as total_amount
		FROM invoices
		WHERE balance_due > 0
		GROUP BY age_bracket
		ORDER BY age_bracket
	`)

	if err != nil {
		log.Printf("Error fetching aging report: %v", err)
	} else {
		defer agingRows.Close()
		for agingRows.Next() {
			var ia InvoiceAging
			var totalAmount sql.NullFloat64
			err := agingRows.Scan(&ia.AgeBracket, &ia.InvoiceCount, &totalAmount)
			if err == nil {
				if totalAmount.Valid {
					ia.TotalAmount = totalAmount.Float64
				}
				analytics.AgingReport = append(analytics.AgingReport, ia)
			}
		}
	}

	// Status distribution
	statusRows, err := db.Query(`
		SELECT
			status,
			COUNT(*) as count,
			SUM(total_amount) as total_amount
		FROM invoices
		GROUP BY status
		ORDER BY count DESC
	`)

	if err != nil {
		log.Printf("Error fetching status distribution: %v", err)
	} else {
		defer statusRows.Close()
		for statusRows.Next() {
			var sc StatusCount
			var totalAmount sql.NullFloat64
			err := statusRows.Scan(&sc.Status, &sc.Count, &totalAmount)
			if err == nil {
				if totalAmount.Valid {
					sc.TotalAmount = totalAmount.Float64
				}
				analytics.StatusDistribution = append(analytics.StatusDistribution, sc)
			}
		}
	}

	// Average payment days
	var avgDays sql.NullFloat64
	db.QueryRow(`
		SELECT AVG(
			EXTRACT(EPOCH FROM (
				SELECT MIN(p.payment_date::date - i.issue_date::date)
				FROM payments p
				WHERE p.invoice_id = i.id
			))
		)
		FROM invoices i
		WHERE i.status = 'paid'
	`).Scan(&avgDays)
	if avgDays.Valid {
		analytics.AveragePaymentDays = avgDays.Float64
	}

	// Collection rate (paid / total issued)
	var totalIssued, totalPaid sql.NullFloat64
	db.QueryRow(`SELECT SUM(total_amount) FROM invoices WHERE status != 'draft'`).Scan(&totalIssued)
	db.QueryRow(`SELECT SUM(paid_amount) FROM invoices WHERE status != 'draft'`).Scan(&totalPaid)

	if totalIssued.Valid && totalPaid.Valid && totalIssued.Float64 > 0 {
		rate := (totalPaid.Float64 / totalIssued.Float64) * 100
		// Cap at 100% (can't collect more than 100% of invoices, even with overpayments)
		if rate > 100 {
			rate = 100
		}
		analytics.CollectionRate = rate
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}
