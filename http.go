package stuckjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultMaxBodySize limits incoming HTTP JSON payloads to 1MB to prevent memory exhaustion attacks.

var DefaultMaxBodySize int64 = 1024 * 1024

// Bind decodes the JSON request body into a new instance of T, checking content type and enforcing size limits.

func Bind[T any](r *http.Request) (T, error) {
	var target T

	if r.Body == nil {
		return target, errors.New("empty request body")
	}

	defer r.Body.Close()

	// Check Content-Type header if provided

	ct := r.Header.Get("Content-Type")

	if ct != "" && !strings.Contains(strings.ToLower(ct), "application/json") {
		return target, fmt.Errorf("invalid Content-Type: expected application/json, got '%s'", ct)
	}

	// Protect against oversized bodies

	limitedReader := io.LimitReader(r.Body, DefaultMaxBodySize+1)

	dec := json.NewDecoder(limitedReader)

	dec.DisallowUnknownFields() // Strict & secure parsing

	if err := dec.Decode(&target); err != nil {
		return target, &DecodeError{Err: err, Message: "failed to parse JSON request body"}
	}

	// Check if there is extra unexpected payload

	var extra json.RawMessage

	if err := dec.Decode(&extra); err != io.EOF {
		return target, errors.New("request body contains multiple JSON values or exceeds maximum allowed size")
	}

	return target, nil
}

// BindAndValidate decodes the request body into T, applies default values to unset fields,

// and runs full struct validation.

// Returns populated T and nil error on success, or validation/decode error.

func BindAndValidate[T any](r *http.Request) (T, error) {
	val, err := Bind[T](r)

	if err != nil {
		return val, err
	}

	// Apply default values to zero fields

	if err := ApplyDefaults(&val); err != nil {
		return val, fmt.Errorf("failed to apply field defaults: %w", err)
	}

	// Run validation

	if err := ValidateStruct(&val); err != nil {
		return val, err
	}

	return val, nil
}

// Write serializes value v into HTTP response with status code and Content-Type: application/json.

func Write(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if v == nil {
		_, err := w.Write([]byte("{}"))
		return err
	}

	return json.NewEncoder(w).Encode(v)
}

// WriteError formats validation or general errors into a clean, standardized JSON response.

func WriteError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if ve, ok := err.(ValidationErrors); ok {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   "validation_failed",
			"message": "Входные данные содержат ошибки",
			"details": ve.ToMap(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   "bad_request",
		"message": err.Error(),
	})
}
