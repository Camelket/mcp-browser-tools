package utils

import (
	"fmt"
	"time"
)

// RetryableError is a custom error type for errors that can be retried.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error: %v", e.Err)
}

// IsRetryable checks if an error is a RetryableError.
func IsRetryable(err error) bool {
	_, ok := err.(*RetryableError)
	return ok
}

// withRetry executes a function with retry logic for transient errors.
// It retries the function up to maxRetries times with an exponential backoff.
func WithRetry(fn func() error, maxRetries int, initialDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		if IsRetryable(err) {
			fmt.Printf("Attempt %d failed with retryable error: %v. Retrying in %v...\n", i+1, err, initialDelay)
			time.Sleep(initialDelay)
			initialDelay *= 2 // Exponential backoff
			continue
		}

		// If it's not a retryable error, return immediately
		return err
	}
	return fmt.Errorf("function failed after %d retries", maxRetries)
}

// ErrorHandler provides utilities for consistent error handling.
type ErrorHandler struct{}

// NewErrorHandler creates a new instance of ErrorHandler.
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// Handle logs and processes an error.
func (eh *ErrorHandler) Handle(err error, message string) {
	if err != nil {
		fmt.Printf("ERROR: %s: %v\n", message, err)
		// In a real application, you might send this to a logging service,
		// trigger alerts, or perform other error-specific actions.
	}
}
