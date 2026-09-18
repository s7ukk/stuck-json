// Package stuckjson provides a fast, ergonomic, and zero-boilerplate JSON toolkit for Go.

// It integrates type-safe generic decoding, lightning-fast struct validation,

// automatic default-value population for unset fields, dynamic JSON path querying,

// and hardened HTTP request binding.

package stuckjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Decode reads JSON from an io.Reader directly into a new instance of generic type T.

func Decode[T any](r io.Reader) (T, error) {
	var target T

	err := json.NewDecoder(r).Decode(&target)

	return target, err
}

// DecodeBytes decodes JSON raw bytes into a new instance of generic type T.

func DecodeBytes[T any](data []byte) (T, error) {
	var target T

	err := json.Unmarshal(data, &target)

	return target, err
}

// DecodeString decodes a JSON string into a new instance of generic type T.

func DecodeString[T any](s string) (T, error) {
	return DecodeBytes[T]([]byte(s))
}

// DecodeAndValidate unmarshals JSON into T, populates any unset fields with their `default:"..."` tag values,

// and runs full tag/custom validation.

func DecodeAndValidate[T any](data []byte) (T, error) {
	val, err := DecodeBytes[T](data)
	if err != nil {
		return val, &DecodeError{Err: err, Message: "failed to unmarshal JSON"}
	}

	if err := ApplyDefaults(&val); err != nil {
		return val, fmt.Errorf("stuckjson.DecodeAndValidate: failed to apply defaults: %w", err)
	}

	if err := ValidateStruct(&val); err != nil {
		return val, err
	}

	return val, nil
}

// Encode writes value v serialized as JSON directly to an io.Writer.

func Encode(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

// ToBytes converts value v into compact JSON bytes.

func ToBytes(v any) ([]byte, error) {
	return json.Marshal(v)
}

// ToString converts value v into a compact JSON string.

func ToString(v any) (string, error) {
	b, err := ToBytes(v)

	if err != nil {
		return "", err
	}

	return string(b), nil
}

// Pretty returns formatted indented JSON string with 2-space indentation.

func Pretty(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")

	if err != nil {
		return "", err
	}

	return string(b), nil
}

// PrettyBytes formats raw JSON bytes with 2-space indentation.

func PrettyBytes(data []byte) (string, error) {
	var buf bytes.Buffer

	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Compact minifies raw JSON bytes.

func Compact(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	if err := json.Compact(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Valid checks if raw bytes form syntactically valid JSON.

func Valid(data []byte) bool {
	return json.Valid(data)
}

// Clone creates a deep copy of a data structure using JSON serialization.
func Clone[T any](src T) (T, error) {
	var dst T

	b, err := json.Marshal(src)

	if err != nil {
		return dst, err
	}

	err = json.Unmarshal(b, &dst)

	return dst, err
}

// MustToBytes marshals v into JSON bytes, panicking on serialization error.

func MustToBytes(v any) []byte {
	b, err := ToBytes(v)

	if err != nil {
		panic(fmt.Sprintf("stuckjson.MustToBytes failed: %v", err))
	}

	return b
}

// MustToString marshals v into JSON string, panicking on serialization error.

func MustToString(v any) string {
	return string(MustToBytes(v))
}

// MustDecode unmarshals bytes into T, panicking on error.

func MustDecode[T any](data []byte) T {
	val, err := DecodeBytes[T](data)

	if err != nil {
		panic(fmt.Sprintf("stuckjson.MustDecode failed: %v", err))
	}

	return val
}
