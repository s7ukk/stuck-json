package stuckjson

import (
	"crypto/rand"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ApplyDefaults inspects a struct pointer and assigns default values from the `default:"..."` tag
// to any zero-value / unset fields.
// This ensures missing JSON fields are automatically loaded with sensible defaults.

func ApplyDefaults(v any) error {
	val := reflect.ValueOf(v)

	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("stuckjson.ApplyDefaults: expected non-nil pointer to struct, got %T", v)
	}

	elem := val.Elem()

	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("stuckjson.ApplyDefaults: expected pointer to struct, got pointer to %s", elem.Kind())
	}

	return setDefaultsRecursive(elem)
}

func setDefaultsRecursive(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !fieldVal.CanSet() {
			continue
		}

		// Recurse into nested structs
		if fieldVal.Kind() == reflect.Struct && fieldType.Type != reflect.TypeOf(time.Time{}) {
			if err := setDefaultsRecursive(fieldVal); err != nil {
				return err
			}
			continue
		}

		// Recurse into pointer to struct if not nil
		if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Struct {
			if err := setDefaultsRecursive(fieldVal.Elem()); err != nil {
				return err
			}
			continue
		}

		defaultTag, hasDefault := fieldType.Tag.Lookup("default")
		if !hasDefault || defaultTag == "" {
			continue
		}

		// Check if the field is currently unset (zero value)
		if isZero(fieldVal) {
			if err := assignDefaultValue(fieldVal, defaultTag); err != nil {
				return fmt.Errorf("field %s: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

func isZero(val reflect.Value) bool {
	if !val.IsValid() {
		return true
	}

	return val.IsZero()
}

func assignDefaultValue(field reflect.Value, defaultStr string) error {
	// Special dynamic values
	switch defaultStr {
	case "now":
		if field.Type() == reflect.TypeOf(time.Time{}) {
			field.Set(reflect.ValueOf(time.Now().UTC()))
			return nil
		}
	case "uuid":
		if field.Kind() == reflect.String {
			field.SetString(generateSimpleUUID())
			return nil
		}
	}

	// Type specific parsing
	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultStr)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle time.Duration
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(defaultStr)
			if err != nil {
				return fmt.Errorf("invalid duration format for default '%s': %w", defaultStr, err)
			}
			field.SetInt(int64(d))
			return nil
		}

		n, err := strconv.ParseInt(defaultStr, 10, 64)

		if err != nil {
			return fmt.Errorf("invalid int default '%s': %w", defaultStr, err)
		}

		field.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(defaultStr, 10, 64)

		if err != nil {
			return fmt.Errorf("invalid uint default '%s': %w", defaultStr, err)
		}

		field.SetUint(n)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(defaultStr, 64)

		if err != nil {
			return fmt.Errorf("invalid float default '%s': %w", defaultStr, err)
		}

		field.SetFloat(f)

	case reflect.Bool:
		b, err := strconv.ParseBool(defaultStr)

		if err != nil {
			return fmt.Errorf("invalid bool default '%s': %w", defaultStr, err)
		}

		field.SetBool(b)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			items := strings.Split(defaultStr, ",")
			slice := reflect.MakeSlice(field.Type(), len(items), len(items))

			for idx, item := range items {
				slice.Index(idx).SetString(strings.TrimSpace(item))
			}

			field.Set(slice)

			return nil
		}

		return fmt.Errorf("defaults for slice element type %v not supported", field.Type().Elem().Kind())

	default:
		return fmt.Errorf("unsupported kind %s for default tag", field.Kind())
	}

	return nil
}

func generateSimpleUUID() string {
	var b [16]byte
	
	_, _ = rand.Read(b[:])

	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant RFC 4122

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
