package main

import (
	"encoding/json"
	"net/http"
)

// Edge Case 1: Status code in variable
func handleWithVariable(w http.ResponseWriter, r *http.Request) {
	status := 200
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// Edge Case 2: Status code from function call
func getStatusCode() int {
	return 404
}

func handleWithFunction(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(getStatusCode())
	json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
}

// Edge Case 3: Conditional status code
func handleConditional(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		w.WriteHeader(201)
	} else {
		w.WriteHeader(200)
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Edge Case 4: Status in switch statement
func handleSwitch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		w.WriteHeader(201)
	case "DELETE":
		w.WriteHeader(204)
	default:
		w.WriteHeader(200)
	}
}

// Edge Case 5: Returning error with custom error type
type CustomError struct {
	Message string
}

func (e CustomError) Error() string {
	return e.Message
}

func handleCustomError(w http.ResponseWriter, r *http.Request) {
	err := CustomError{Message: "something went wrong"}
	if err.Error() != "" {
		w.WriteHeader(http.StatusOK) // This should be caught - error in context
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
}

// Edge Case 6: Multiple status codes in same error block
func handleMultipleStatuses(w http.ResponseWriter, r *http.Request) {
	if err := validate(r); err != nil {
		if err.Error() == "validation" {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
}

func validate(r *http.Request) error {
	return nil
}

// Edge Case 7: Error variable named differently
func handleDifferentErrorName(w http.ResponseWriter, r *http.Request) {
	e := validate(r)
	if e != nil {
		// Missing status code - should be detected
		json.NewEncoder(w).Encode(map[string]string{"error": e.Error()})
		return
	}
}

// Edge Case 8: Nested error checks
func handleNestedErrors(w http.ResponseWriter, r *http.Request) {
	data, err := getData()
	if err != nil {
		validationErr := validateError(err)
		if validationErr != nil {
			// Missing status code in nested error - should be detected
			json.NewEncoder(w).Encode(map[string]string{"error": validationErr.Error()})
			return
		}
	}
	// Use data to avoid "declared and not used" error
	_ = data
}

func getData() (interface{}, error) {
	return nil, nil
}

func validateError(err error) error {
	return err
}

// Edge Case 9: Comments with "error" keyword shouldn't trigger
func handleCommentedError(w http.ResponseWriter, r *http.Request) {
	// This handles error cases properly
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Edge Case 10: String containing "error" but not actually an error
func handleErrorString(w http.ResponseWriter, r *http.Request) {
	message := "No error occurred"
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
