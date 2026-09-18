package stuckjson

import (
	"fmt"
	"strings"
)

// FieldError represents a single validation failure on a struct field.

type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

// Error formats the field error into a readable string.

func (f FieldError) Error() string {
	if f.Message != "" {
		return fmt.Sprintf("field '%s': %s", f.Field, f.Message)
	}

	return fmt.Sprintf("field '%s' failed validation on rule '%s'", f.Field, f.Rule)
}

// ValidationErrors is a slice of FieldError implementing error and fmt.Stringer.

type ValidationErrors []FieldError

// Error joins all validation errors into a single human-readable string.

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("validation failed with %d error(s): ", len(ve)))

	for i, err := range ve {
		if i > 0 {
			sb.WriteString("; ")
		}

		sb.WriteString(err.Error())
	}

	return sb.String()
}

// ToMap converts the errors list into a map[field]message suitable for JSON API responses.

func (ve ValidationErrors) ToMap() map[string]string {
	m := make(map[string]string, len(ve))
	for _, err := range ve {
		m[err.Field] = err.Message
	}

	return m
}

// HasField returns true if there is an error for the given field name.

func (ve ValidationErrors) HasField(field string) bool {
	for _, err := range ve {
		if strings.EqualFold(err.Field, field) {
			return true
		}
	}

	return false
}

// DecodeError represents a failure while parsing JSON input.

type DecodeError struct {
	Err     error  `json:"-"`
	Message string `json:"message"`
	Offset  int64  `json:"offset,omitempty"`
}

func (e *DecodeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("json decode error: %s (%v)", e.Message, e.Err)
	}
	
	return fmt.Sprintf("json decode error: %s", e.Message)
}

func (e *DecodeError) Unwrap() error {
	return e.Err
}
